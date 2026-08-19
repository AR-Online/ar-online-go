package aronline_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	aronline "github.com/AR-Online/ar-online-go"
)

// A real server on a real port, like fakeapi_test.go does for /v3. The legacy
// gateway gets its own fake because what it has to get right is different: the
// credential travels raw, one family hides its status inside the body, and one
// route answers a PDF instead of JSON. A stubbed RoundTripper would prove only
// that the code calls the stub.

// gwReceived is what the fake gateway saw.
type gwReceived struct {
	Method string
	Path   string
	// EscapedPath is the path as it travelled on the wire. Path is already
	// decoded, so only this one can prove an id was escaped instead of steering
	// the call to another route.
	EscapedPath   string
	RawQuery      string
	Authorization string
	Accept        string
	ContentType   string
	Body          string
}

// fakeGateway is a one-request-at-a-time stand-in for api.ar-online.com.br.
type fakeGateway struct {
	server      *httptest.Server
	status      int
	body        string
	contentType string
	received    gwReceived
}

func newFakeGateway(t *testing.T) *fakeGateway {
	t.Helper()

	fake := &fakeGateway{status: http.StatusOK, body: "{}", contentType: "application/json"}

	fake.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)

		fake.received = gwReceived{
			Method:        r.Method,
			Path:          r.URL.Path,
			EscapedPath:   r.URL.EscapedPath(),
			RawQuery:      r.URL.RawQuery,
			Authorization: r.Header.Get("Authorization"),
			Accept:        r.Header.Get("Accept"),
			ContentType:   r.Header.Get("Content-Type"),
			Body:          string(body),
		}

		w.Header().Set("Content-Type", fake.contentType)
		w.WriteHeader(fake.status)
		_, _ = w.Write([]byte(fake.body))
	}))

	t.Cleanup(fake.server.Close)

	return fake
}

// answers replies with this JSON body and 200.
func (f *fakeGateway) answers(t *testing.T, body any) {
	t.Helper()

	encoded, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("não consegui montar o corpo do gateway de mentira: %v", err)
	}

	f.status = http.StatusOK
	f.body = string(encoded)
	f.contentType = "application/json"
}

// answersRaw replies with exactly these bytes -- a PDF, or the HTML of a proxy
// that answered in the gateway's place.
func (f *fakeGateway) answersRaw(status int, contentType, body string) {
	f.status = status
	f.body = body
	f.contentType = contentType
}

// refuses replies with the gateway's own error shape and an HTTP status.
func (f *fakeGateway) refuses(t *testing.T, status int, body map[string]any) {
	t.Helper()

	encoded, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("não consegui montar a recusa do gateway de mentira: %v", err)
	}

	f.answersRaw(status, "application/json", string(encoded))
}

// client points the legacy area at the fake and carries the gateway token.
func (f *fakeGateway) client() *aronline.Client {
	return aronline.New(aronline.Options{
		LegacyToken:   "tok-gw",
		LegacyBaseURL: f.server.URL,
	})
}

// anonymous points at the fake with no gateway credential. A separate
// constructor rather than an optional argument: a test that means to prove the
// path WITHOUT a token must not get one by default.
func (f *fakeGateway) anonymous() *aronline.Client {
	return aronline.New(aronline.Options{LegacyBaseURL: f.server.URL})
}

// legacyError pulls the *LegacyAPIError out of an error, or fails the test.
func legacyError(t *testing.T, err error) *aronline.LegacyAPIError {
	t.Helper()

	if err == nil {
		t.Fatal("esperava uma recusa do gateway, veio nil")
	}

	var failure *aronline.LegacyAPIError
	if !errorsAs(err, &failure) {
		t.Fatalf("esperava *aronline.LegacyAPIError, veio %T: %v", err, err)
	}

	return failure
}

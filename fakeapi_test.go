package aronline_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	aronline "github.com/AR-Online/ar-online-go"
)

// A real server on a real port, not a stubbed RoundTripper. What this SDK has
// to get right IS the wire: which routes wrap their answer, how a refusal
// comes back, what happens when something that is not the API answers. A stub
// would only prove that the code calls the stub.

// received is what the fake saw.
type received struct {
	Method string
	Path   string
	// EscapedPath is the path as it travelled on the wire. Path is already
	// decoded, so only this one can prove an id was escaped instead of
	// steering the call to another route.
	EscapedPath   string
	RawQuery      string
	Authorization string
	Accept        string
}

// fakeAPI is a one-request-at-a-time stand-in for the API.
type fakeAPI struct {
	server   *httptest.Server
	status   int
	body     string
	headers  map[string]string
	received received
}

func newFakeAPI(t *testing.T) *fakeAPI {
	t.Helper()

	fake := &fakeAPI{status: http.StatusOK, body: "{}"}

	fake.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fake.received = received{
			Method:        r.Method,
			Path:          r.URL.Path,
			EscapedPath:   r.URL.EscapedPath(),
			RawQuery:      r.URL.RawQuery,
			Authorization: r.Header.Get("Authorization"),
			Accept:        r.Header.Get("Accept"),
		}

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Request-Id", "req-do-cabecalho")

		for name, value := range fake.headers {
			w.Header().Set(name, value)
		}

		w.WriteHeader(fake.status)
		_, _ = w.Write([]byte(fake.body))
	}))

	t.Cleanup(fake.server.Close)

	return fake
}

// answers replies with this JSON body and 200.
func (f *fakeAPI) answers(t *testing.T, body any) {
	t.Helper()

	encoded, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("não consegui montar o corpo do servidor de mentira: %v", err)
	}

	f.status = http.StatusOK
	f.body = string(encoded)
	f.headers = nil
}

// refuses replies with the catalog's error envelope and a status.
func (f *fakeAPI) refuses(t *testing.T, status int, errorBody map[string]any, headers map[string]string) {
	t.Helper()

	encoded, err := json.Marshal(map[string]any{"error": errorBody})
	if err != nil {
		t.Fatalf("não consegui montar o erro do servidor de mentira: %v", err)
	}

	f.status = status
	f.body = string(encoded)
	f.headers = headers
}

// answersRaw replies with something that is not JSON -- a proxy in the middle.
func (f *fakeAPI) answersRaw(status int, body string) {
	f.status = status
	f.body = body
	f.headers = map[string]string{"Content-Type": "text/html"}
}

// client points at the fake and carries a token.
func (f *fakeAPI) client() *aronline.Client {
	return aronline.New(aronline.Options{Token: "tok", BaseURL: f.server.URL})
}

// anonymous points at the fake with no credential. A separate constructor
// rather than an optional argument: a test that means to prove the path
// WITHOUT a token must not get one by default.
func (f *fakeAPI) anonymous() *aronline.Client {
	return aronline.New(aronline.Options{BaseURL: f.server.URL})
}

// apiError pulls the *APIError out of an error, or fails the test.
func apiError(t *testing.T, err error) *aronline.APIError {
	t.Helper()

	if err == nil {
		t.Fatal("esperava uma recusa da API, veio nil")
	}

	var failure *aronline.APIError
	if !errorsAs(err, &failure) {
		t.Fatalf("esperava *aronline.APIError, veio %T: %v", err, err)
	}

	return failure
}

package aronline

import "context"

// Freshness says how fresh the copy of the data is.
//
// It answers in COUNTS, not in a list of tables: "46 tracked, 3 behind" is an
// answer to "is it fresh"; forty-six table names is a report nobody reads at
// the moment the question is asked.
type Freshness struct {
	// RefreshedAt is when the measurement was taken.
	RefreshedAt *string `json:"refreshed_at"`
	// LastLoadAt is the newest read mark across every tracked source.
	LastLoadAt *string `json:"last_load_at"`
	// WorstLagSeconds is nil when no source carries a read mark yet -- which
	// is not zero lag.
	WorstLagSeconds *int `json:"worst_lag_seconds"`
	// SourcesTracked is how many sources the load watches.
	SourcesTracked int `json:"sources_tracked"`
	// SourcesBehind is how many are past the threshold.
	SourcesBehind int `json:"sources_behind"`
	// SourcesNotLoaded is its own count -- it is not lag, and the fix for
	// "the load has not started" is not the fix for "it is behind".
	SourcesNotLoaded int `json:"sources_not_loaded"`
}

// FreshnessService reports how far behind the copy of the data is.
//
// It answers the practical question behind a query that returned less than
// expected: is the API wrong, or is the load late? Without this number the two
// look the same.
type FreshnessService struct {
	transport *transport
}

// Get answers the freshness, measured by the database clock.
func (s *FreshnessService) Get(ctx context.Context) (*Freshness, error) {
	var freshness Freshness

	// No envelope on this one: the route answers the object itself.
	if err := s.transport.bare(ctx, "/v3/freshness", true, &freshness); err != nil {
		return nil, err
	}

	return &freshness, nil
}

package aronline

import "context"

// BehindTable is a table whose copy is past the threshold.
type BehindTable struct {
	Legacy     string `json:"legacy"`
	LagSeconds int    `json:"lag_seconds"`
}

// Freshness says how fresh the copy of the data is.
type Freshness struct {
	RefreshedAt *string `json:"refreshed_at"`
	LastLoadAt  *string `json:"last_load_at"`
	// WorstLagSeconds is nil when no table carries a read mark yet.
	WorstLagSeconds *int `json:"worst_lag_seconds"`
	TablesTracked   int  `json:"tables_tracked"`
	// TablesNeverLoaded is its own count -- it is not lag, and the fix for
	// "the loader has not started" is not the fix for "it is behind".
	TablesNeverLoaded int `json:"tables_never_loaded"`
	// Behind lists the tables past the threshold, worst first.
	Behind []BehindTable `json:"behind"`
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

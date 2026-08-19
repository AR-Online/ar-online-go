package aronline

import "context"

// VersionInfo is what the API is running.
type VersionInfo struct {
	Version      string `json:"version"`
	MinMigration string `json:"min_migration"`
	Environment  string `json:"environment"`
}

// VersionService reports which version of the API is running.
//
// The only open route in the SDK: it sends no token and works on a client
// built without one. It is the first thing support asks for.
type VersionService struct {
	transport *transport
}

// Get answers the running version, the migration it needs and the environment.
func (s *VersionService) Get(ctx context.Context) (*VersionInfo, error) {
	var info VersionInfo

	// No envelope, and no credential.
	if err := s.transport.bare(ctx, "/v3/version", false, &info); err != nil {
		return nil, err
	}

	return &info, nil
}

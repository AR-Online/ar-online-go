package aronline

import (
	"context"
	"net/url"
)

// Tag is a label. Labels belong to a person, not to an entity.
type Tag struct {
	ID    string  `json:"id"`
	Name  string  `json:"name"`
	Color *string `json:"color"`
	// CreatedAt is ISO 8601 with the real offset.
	CreatedAt string `json:"created_at"`
}

// TagsService reaches your labels.
//
// Labels are personal: these routes answer a PERSON's token. An integration
// token gets 403 saying so, rather than an empty list -- which would read as
// "you have none".
type TagsService struct {
	transport *transport
}

// List answers your labels, ordered by name.
func (s *TagsService) List(ctx context.Context) ([]Tag, error) {
	var tags []Tag
	if err := s.transport.envelope(ctx, "/v3/tags", nil, &tags); err != nil {
		return nil, err
	}

	return tags, nil
}

// Get answers one of your labels. A label that does not exist and one that is
// not yours both answer 404.
func (s *TagsService) Get(ctx context.Context, id string) (*Tag, error) {
	var tag Tag

	path := "/v3/tags/" + url.PathEscape(id)
	if err := s.transport.envelope(ctx, path, nil, &tag); err != nil {
		return nil, err
	}

	return &tag, nil
}

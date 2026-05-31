package api

import (
	"encoding/json"

	agenterrors "github.com/shhac/agent-stripe/internal/errors"
)

type ListResponse struct {
	Object   string            `json:"object"`
	URL      string            `json:"url,omitempty"`
	HasMore  bool              `json:"has_more"`
	Data     []json.RawMessage `json:"data"`
	NextPage string            `json:"next_page,omitempty"`
}

func DecodeList(raw json.RawMessage) (*ListResponse, error) {
	var list ListResponse
	if err := json.Unmarshal(raw, &list); err != nil {
		return nil, agenterrors.Wrap(err, agenterrors.FixableByAgent)
	}
	return &list, nil
}

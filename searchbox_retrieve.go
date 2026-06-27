package mapbox

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
)

// RetrieveRequest is the request for Search Box /retrieve.
type RetrieveRequest struct {
	// MapboxID is the mapbox_id from a Suggestion (required).
	MapboxID string
	// SessionToken must match the session_token used in the preceding Suggest calls.
	SessionToken string
	// Language is the IETF language tag for the response.
	Language string
}

// RetrieveResponse is the response from Search Box /retrieve.
type RetrieveResponse struct {
	Type        string            `json:"type"`
	Features    []RetrieveFeature `json:"features"`
	Attribution string            `json:"attribution"`
}

// Retrieve resolves a suggestion to a full place record including coordinates.
// The SessionToken must match the one used in the preceding Suggest call.
func (c *Client) Retrieve(ctx context.Context, req *RetrieveRequest) (_ *RetrieveResponse, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("mapbox: retrieve: %w", err)
		}
	}()

	if req.MapboxID == "" {
		return nil, fmt.Errorf("mapbox_id is required")
	}
	if req.SessionToken == "" {
		return nil, fmt.Errorf("session token is required")
	}

	params := url.Values{}
	params.Set("session_token", req.SessionToken)
	if req.Language != "" {
		params.Set("language", req.Language)
	}

	endpoint := c.baseURL + "/search/searchbox/v1/retrieve/" + url.PathEscape(req.MapboxID) + "?" + params.Encode()
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	httpResp, err := c.do(httpReq)
	if err != nil {
		return nil, err
	}
	defer func() {
		if cerr := httpResp.Body.Close(); cerr != nil {
			slog.WarnContext(ctx, "mapbox: failed to close retrieve response body", "error", cerr)
		}
	}()

	if httpResp.StatusCode != http.StatusOK {
		return nil, newResponseError(httpResp)
	}

	var result RetrieveResponse
	if err := json.NewDecoder(httpResp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return &result, nil
}

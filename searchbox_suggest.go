package mapbox

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// SuggestRequest is the request for Search Box /suggest.
type SuggestRequest struct {
	// Query is the search string (required, max 256 chars).
	Query string
	// SessionToken is a UUIDv4 that groups suggest+retrieve calls for billing.
	// The same token must be used across the entire suggest→retrieve sequence.
	// Required.
	SessionToken string
	// Language is the IETF language tag for results (e.g. "en", "de").
	Language string
	// Limit caps the number of suggestions returned (max 10).
	Limit int
	// Proximity biases results toward a coordinate.
	Proximity *Coordinate
	// BBox restricts results to a bounding box.
	BBox *BoundingBox
	// Countries restricts results to a list of ISO 3166 alpha-2 codes.
	Countries []string
	// Types filters results to specific feature types.
	Types []string
}

// SuggestResponse is the response from Search Box /suggest.
type SuggestResponse struct {
	Suggestions []*Suggestion `json:"suggestions"`
	Attribution string        `json:"attribution"`
}

// Suggest performs a Search Box /suggest call, returning autocomplete suggestions
// for the given query. Use the same SessionToken for the paired Retrieve call.
func (c *Client) Suggest(ctx context.Context, req *SuggestRequest) (_ *SuggestResponse, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("mapbox: suggest: %w", err)
		}
	}()

	if req.Query == "" {
		return nil, fmt.Errorf("query is required")
	}
	if req.SessionToken == "" {
		return nil, fmt.Errorf("session token is required")
	}

	params := url.Values{}
	params.Set("q", req.Query)
	params.Set("session_token", req.SessionToken)
	if req.Language != "" {
		params.Set("language", req.Language)
	}
	if req.Limit > 0 {
		params.Set("limit", strconv.Itoa(req.Limit))
	}
	if req.Proximity != nil {
		params.Set("proximity", formatCoordParam(req.Proximity.Longitude, req.Proximity.Latitude))
	}
	if req.BBox != nil {
		params.Set("bbox", formatBBoxParam(req.BBox))
	}
	if len(req.Countries) > 0 {
		params.Set("country", strings.Join(req.Countries, ","))
	}
	if len(req.Types) > 0 {
		params.Set("types", strings.Join(req.Types, ","))
	}

	endpoint := c.baseURL + "/search/searchbox/v1/suggest?" + params.Encode()
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
			slog.WarnContext(ctx, "mapbox: failed to close suggest response body", "error", cerr)
		}
	}()

	if httpResp.StatusCode != http.StatusOK {
		return nil, newResponseError(httpResp)
	}

	var result SuggestResponse
	if err := json.NewDecoder(httpResp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return &result, nil
}

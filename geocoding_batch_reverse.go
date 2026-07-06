package mapbox

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
)

// BatchReverseGeocodeRequest is the request for Geocoding v6 Batch.
type BatchReverseGeocodeRequest struct {
	// Queries are the coordinates to reverse geocode. Must contain 1–1000 items.
	Queries []ReverseGeocodeQuery
	// Permanent requests Permanent tier geocoding, which permits storing results.
	// Defaults to Temporary tier when false.
	Permanent bool
}

// ReverseGeocodeQuery is a single query in a batch reverse geocode request.
type ReverseGeocodeQuery struct {
	// Longitude is the coordinate to reverse geocode.
	Longitude float64
	// Latitude is the coordinate to reverse geocode.
	Latitude float64
}

// batchGeocodeQuery is the JSON wire format for a single batch geocode query.
type batchGeocodeQuery struct {
	Types     string  `json:"types"`
	Longitude float64 `json:"longitude"`
	Latitude  float64 `json:"latitude"`
}

// BatchReverseGeocode performs a batch reverse geocode lookup using Geocoding v6 Batch.
// Returns one *[FeatureCollection] per input query, in the same order.
// Up to 1,000 queries per request.
func (c *Client) BatchReverseGeocode(ctx context.Context, req *BatchReverseGeocodeRequest) (_ []*FeatureCollection, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("mapbox: batch reverse geocode: %w", err)
		}
	}()

	if len(req.Queries) == 0 {
		return nil, fmt.Errorf("at least one query required")
	}
	if len(req.Queries) > 1000 {
		return nil, fmt.Errorf("at most 1000 queries allowed per request, got %d", len(req.Queries))
	}

	queries := make([]batchGeocodeQuery, len(req.Queries))
	for i, q := range req.Queries {
		queries[i] = batchGeocodeQuery{
			Types:     "address",
			Longitude: q.Longitude,
			Latitude:  q.Latitude,
		}
	}

	body, err := json.Marshal(queries)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	params := url.Values{}
	if req.Permanent {
		params.Set("permanent", "true")
	}
	path := "/search/geocode/v6/batch"
	if len(params) > 0 {
		path += "?" + params.Encode()
	}
	endpoint := c.baseURL + path
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	httpResp, err := c.do(httpReq)
	if err != nil {
		return nil, err
	}
	defer func() {
		if cerr := httpResp.Body.Close(); cerr != nil {
			slog.WarnContext(ctx, "mapbox: failed to close batch geocode response body", "error", cerr)
		}
	}()

	if httpResp.StatusCode != http.StatusOK {
		return nil, newResponseError(httpResp)
	}

	var wrapper struct {
		Batch []*FeatureCollection `json:"batch"`
	}
	if err := json.NewDecoder(httpResp.Body).Decode(&wrapper); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return wrapper.Batch, nil
}

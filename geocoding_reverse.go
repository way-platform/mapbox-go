package mapbox

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
)

// ReverseGeocodeRequest is the request for Geocoding v6 Reverse.
type ReverseGeocodeRequest struct {
	// Longitude is the coordinate to reverse geocode.
	Longitude float64
	// Latitude is the coordinate to reverse geocode.
	Latitude float64
	// Permanent requests Permanent tier geocoding, which permits storing results.
	// Defaults to Temporary tier when false.
	Permanent bool
}

// ReverseGeocode performs a single reverse geocode lookup using Geocoding v6.
func (c *Client) ReverseGeocode(ctx context.Context, req *ReverseGeocodeRequest) (_ *FeatureCollection, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("mapbox: reverse geocode: %w", err)
		}
	}()

	params := url.Values{}
	params.Set("longitude", strconv.FormatFloat(req.Longitude, 'f', -1, 64))
	params.Set("latitude", strconv.FormatFloat(req.Latitude, 'f', -1, 64))
	if req.Permanent {
		params.Set("permanent", "true")
	}

	endpoint := c.baseURL + "/search/geocode/v6/reverse?" + params.Encode()
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
			slog.WarnContext(ctx, "mapbox: failed to close reverse geocode response body", "error", cerr)
		}
	}()

	if httpResp.StatusCode != http.StatusOK {
		return nil, newResponseError(httpResp)
	}

	var result FeatureCollection
	if err := json.NewDecoder(httpResp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return &result, nil
}

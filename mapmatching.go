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

// Coordinate is a geographic coordinate with longitude and latitude.
type Coordinate struct {
	// Longitude is the east-west position in decimal degrees.
	Longitude float64
	// Latitude is the north-south position in decimal degrees.
	Latitude float64
}

// MapMatchRequest is the request for the Mapbox Map Matching API v5.
type MapMatchRequest struct {
	// Profile is the Mapbox routing profile.
	// Accepted values: "mapbox/driving", "mapbox/walking", "mapbox/cycling",
	// "mapbox/driving-traffic". Defaults to "mapbox/driving" if empty.
	Profile string
	// Coordinates is the sequence of GPS coordinates to match.
	// Must contain between 2 and 100 coordinates.
	Coordinates []Coordinate
	// Radiuses is the per-coordinate uncertainty radius in meters (0–50).
	// If provided, must have the same length as Coordinates.
	// 25m is a reasonable starting value for fleet GPS noise tolerance.
	Radiuses []float64
	// Timestamps is the per-coordinate Unix timestamp in seconds.
	// If provided, must have the same length as Coordinates and be in
	// ascending order. Highly recommended by Mapbox for quality matching.
	Timestamps []float64
}

// MapMatchResponse is the response from the Mapbox Map Matching API.
type MapMatchResponse struct {
	// Code is the Mapbox response code. Check Code.IsSuccess() for a successful match.
	// HTTP 200 is returned for both successful and semantic-failure cases (e.g. NoMatch).
	Code Code `json:"code"`
	// Message contains additional detail, set when Code is not Ok.
	Message string `json:"message,omitempty"`
	// Matchings contains the matched route segments, ordered by confidence descending.
	Matchings []Matching `json:"matchings"`
	// Tracepoints is a per-input-coordinate array. A nil element indicates the
	// corresponding input coordinate could not be matched to a road.
	Tracepoints []*Tracepoint `json:"tracepoints"`
}

// Matching is a single matched route segment.
type Matching struct {
	// Confidence is the match quality score in the range [0, 1].
	// Mapbox treats values below 0.5 as low-quality.
	Confidence float64 `json:"confidence"`
	// Geometry is the matched route geometry as a GeoJSON LineString.
	// Nil when overview=false was requested.
	Geometry *GeoJSONGeometry `json:"geometry"`
	// Distance is the matched distance in metres.
	Distance float64 `json:"distance"`
	// Duration is the estimated travel time in seconds.
	Duration float64 `json:"duration"`
}

// Tracepoint is a matched input coordinate. Nil tracepoints in
// MapMatchResponse.Tracepoints indicate coordinates that could not be matched.
type Tracepoint struct {
	// Location is the snapped coordinate as [longitude, latitude].
	Location []float64 `json:"location"`
	// WaypointIndex is the index of the corresponding waypoint in the matched route.
	WaypointIndex int `json:"waypoint_index"`
	// MatchingsIndex is the index into MapMatchResponse.Matchings that this point belongs to.
	MatchingsIndex int `json:"matchings_index"`
	// AlternativesCount is the number of alternative matches at this point. 0 = unambiguous.
	AlternativesCount int `json:"alternatives_count"`
	// Name is the road name at the snapped location.
	Name string `json:"name"`
}

// GeoJSONGeometry is a GeoJSON geometry object.
type GeoJSONGeometry struct {
	Type        string `json:"type"`
	Coordinates any    `json:"coordinates"`
}

// MapMatch submits GPS coordinates to the Mapbox Map Matching API.
//
// A successful HTTP response with Code "NoMatch" or "NoSegment" is returned
// as (response, nil) — these are not Go errors. Callers should inspect
// response.Code and implement a degradation policy as needed
// (e.g. fall back to the raw input path when Code is not Ok or Confidence < 0.5).
//
// HTTP 4xx/5xx responses are returned as *[Error].
func (c *Client) MapMatch(ctx context.Context, req *MapMatchRequest) (_ *MapMatchResponse, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("mapbox: map match: %w", err)
		}
	}()

	if len(req.Coordinates) < 2 {
		return nil, fmt.Errorf("at least 2 coordinates required")
	}
	if len(req.Coordinates) > 100 {
		return nil, fmt.Errorf("at most 100 coordinates allowed per request")
	}
	if len(req.Radiuses) > 0 && len(req.Radiuses) != len(req.Coordinates) {
		return nil, fmt.Errorf("radiuses length (%d) must match coordinates length (%d)", len(req.Radiuses), len(req.Coordinates))
	}
	if len(req.Timestamps) > 0 && len(req.Timestamps) != len(req.Coordinates) {
		return nil, fmt.Errorf("timestamps length (%d) must match coordinates length (%d)", len(req.Timestamps), len(req.Coordinates))
	}

	profile := req.Profile
	if profile == "" {
		profile = "mapbox/driving"
	}

	form := url.Values{}
	form.Set("coordinates", encodeCoordinates(req.Coordinates))
	if len(req.Radiuses) > 0 {
		form.Set("radiuses", encodeFloats(req.Radiuses))
	}
	if len(req.Timestamps) > 0 {
		form.Set("timestamps", encodeFloats(req.Timestamps))
	}
	form.Set("tidy", "true")
	form.Set("geometries", "geojson")
	form.Set("overview", "full")

	endpoint := fmt.Sprintf("%s/matching/v5/%s", c.baseURL, profile)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	httpResp, err := c.do(httpReq)
	if err != nil {
		return nil, err
	}
	defer func() {
		if cerr := httpResp.Body.Close(); cerr != nil {
			slog.WarnContext(ctx, "mapbox: failed to close map match response body", "error", cerr)
		}
	}()

	if httpResp.StatusCode != http.StatusOK {
		return nil, newResponseError(httpResp)
	}

	var result MapMatchResponse
	if err := json.NewDecoder(httpResp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return &result, nil
}

// encodeCoordinates formats a coordinate slice as "lng,lat;lng,lat;..." for Mapbox.
// Mapbox requires longitude before latitude within each pair.
func encodeCoordinates(coords []Coordinate) string {
	parts := make([]string, len(coords))
	for i, c := range coords {
		parts[i] = strconv.FormatFloat(c.Longitude, 'f', -1, 64) +
			"," +
			strconv.FormatFloat(c.Latitude, 'f', -1, 64)
	}
	return strings.Join(parts, ";")
}

// encodeFloats formats a float64 slice as "v1;v2;v3;..." for Mapbox.
func encodeFloats(values []float64) string {
	parts := make([]string, len(values))
	for i, v := range values {
		parts[i] = strconv.FormatFloat(v, 'f', -1, 64)
	}
	return strings.Join(parts, ";")
}

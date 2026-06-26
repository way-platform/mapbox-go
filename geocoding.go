package mapbox

import (
	"bytes"
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
}

// BatchReverseGeocodeRequest is the request for Geocoding v6 Batch.
type BatchReverseGeocodeRequest struct {
	// Queries are the coordinates to reverse geocode. Must contain 1–1000 items.
	Queries []ReverseGeocodeQuery
}

// ReverseGeocodeQuery is a single query in a batch reverse geocode request.
type ReverseGeocodeQuery struct {
	// Longitude is the coordinate to reverse geocode.
	Longitude float64
	// Latitude is the coordinate to reverse geocode.
	Latitude float64
}

// FeatureCollection is a GeoJSON FeatureCollection returned by Geocoding v6.
type FeatureCollection struct {
	Type     string    `json:"type"`
	Features []Feature `json:"features"`
}

// Feature is a GeoJSON Feature with Mapbox v6 geocoding properties.
type Feature struct {
	Type       string            `json:"type"`
	ID         string            `json:"id"`
	Geometry   GeoJSONGeometry   `json:"geometry"`
	Properties FeatureProperties `json:"properties"`
}

// FeatureProperties contains the Mapbox v6 geocoding result properties.
type FeatureProperties struct {
	// MapboxID is the unique Mapbox identifier for this feature.
	MapboxID string `json:"mapbox_id"`
	// FeatureType is the type of feature (e.g. "address", "place", "region").
	FeatureType string `json:"feature_type"`
	// Name is the primary name of the feature.
	Name string `json:"name"`
	// PlaceFormatted is the place portion of the full address.
	PlaceFormatted string `json:"place_formatted,omitempty"`
	// FullAddress is the full formatted address string.
	FullAddress string `json:"full_address,omitempty"`
	// Context contains the hierarchical address components.
	Context FeatureContext `json:"context"`
}

// FeatureContext contains the hierarchical address components from Mapbox v6.
// All fields are optional pointers — nil indicates the component is not present
// for this feature.
type FeatureContext struct {
	Address      *AddressContext `json:"address,omitempty"`
	Street       *ContextItem    `json:"street,omitempty"`
	Neighborhood *ContextItem    `json:"neighborhood,omitempty"`
	Postcode     *ContextItem    `json:"postcode,omitempty"`
	Locality     *ContextItem    `json:"locality,omitempty"`
	Place        *ContextItem    `json:"place,omitempty"`
	District     *ContextItem    `json:"district,omitempty"`
	Region       *RegionContext  `json:"region,omitempty"`
	Country      *CountryContext `json:"country,omitempty"`
}

// AddressContext contains the street-level address components.
type AddressContext struct {
	// MapboxID is the unique Mapbox identifier.
	MapboxID string `json:"mapbox_id"`
	// AddressNumber is the building/house number.
	AddressNumber string `json:"address_number"`
	// StreetName is the street name.
	StreetName string `json:"street_name"`
}

// ContextItem is a single named level in the address hierarchy.
type ContextItem struct {
	// MapboxID is the unique Mapbox identifier.
	MapboxID string `json:"mapbox_id"`
	// Name is the display name for this context level.
	Name string `json:"name"`
}

// RegionContext is the region/state/province level in the address hierarchy.
type RegionContext struct {
	// MapboxID is the unique Mapbox identifier.
	MapboxID string `json:"mapbox_id"`
	// Name is the region name.
	Name string `json:"name"`
	// RegionCode is the ISO 3166-2 region code (e.g. "DE-BY").
	RegionCode string `json:"region_code,omitempty"`
	// RegionCodeFull is the full ISO 3166-2 region code.
	RegionCodeFull string `json:"region_code_full,omitempty"`
}

// CountryContext is the country level in the address hierarchy.
type CountryContext struct {
	// MapboxID is the unique Mapbox identifier.
	MapboxID string `json:"mapbox_id"`
	// Name is the country name.
	Name string `json:"name"`
	// CountryCode is the ISO 3166-1 alpha-2 country code (e.g. "DE").
	CountryCode string `json:"country_code,omitempty"`
	// CountryCodeAlpha3 is the ISO 3166-1 alpha-3 country code (e.g. "DEU").
	CountryCodeAlpha3 string `json:"country_code_alpha_3,omitempty"`
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

	endpoint := c.baseURL + "/search/geocode/v6/batch"
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

	var results []*FeatureCollection
	if err := json.NewDecoder(httpResp.Body).Decode(&results); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return results, nil
}

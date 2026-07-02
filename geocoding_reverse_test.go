package mapbox_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	mapbox "github.com/way-platform/mapbox-go"
)

func TestReverseGeocode_QueryParams(t *testing.T) {
	var gotReq *http.Request
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotReq = r
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(mapbox.FeatureCollection{Type: "FeatureCollection"}); err != nil {
			t.Errorf("encode response: %v", err)
		}
	}))
	defer srv.Close()

	client := mapbox.NewClient(mapbox.WithAccessToken("my-token"), mapbox.WithBaseURL(srv.URL))
	_, err := client.ReverseGeocode(context.Background(), &mapbox.ReverseGeocodeRequest{
		Longitude: 13.405,
		Latitude:  52.52,
	})
	if err != nil {
		t.Fatalf("ReverseGeocode error: %v", err)
	}

	q := gotReq.URL.Query()
	if lon := q.Get("longitude"); lon != "13.405" {
		t.Errorf("longitude = %q, want %q", lon, "13.405")
	}
	if lat := q.Get("latitude"); lat != "52.52" {
		t.Errorf("latitude = %q, want %q", lat, "52.52")
	}
	if tok := q.Get("access_token"); tok != "my-token" {
		t.Errorf("access_token = %q, want %q", tok, "my-token")
	}
	if gotReq.Method != http.MethodGet {
		t.Errorf("Method = %s, want GET", gotReq.Method)
	}
}

func TestReverseGeocode_ContextParsing(t *testing.T) {
	fc := mapbox.FeatureCollection{
		Type: "FeatureCollection",
		Features: []mapbox.Feature{
			{
				Type: "Feature",
				Properties: mapbox.FeatureProperties{
					MapboxID:    "addr.123",
					FeatureType: "address",
					FullAddress: "Unter den Linden 1, 10117 Berlin, Germany",
					Context: mapbox.FeatureContext{
						Address: &mapbox.AddressContext{
							MapboxID:      "addr.123",
							AddressNumber: "1",
							StreetName:    "Unter den Linden",
						},
						Postcode: &mapbox.ContextItem{
							MapboxID: "post.10117",
							Name:     "10117",
						},
						Place: &mapbox.ContextItem{
							MapboxID: "place.berlin",
							Name:     "Berlin",
						},
						Country: &mapbox.CountryContext{
							MapboxID:          "country.de",
							Name:              "Germany",
							CountryCode:       "DE",
							CountryCodeAlpha3: "DEU",
						},
					},
				},
			},
		},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(fc); err != nil {
			t.Errorf("encode response: %v", err)
		}
	}))
	defer srv.Close()

	client := mapbox.NewClient(mapbox.WithAccessToken("t"), mapbox.WithBaseURL(srv.URL))
	result, err := client.ReverseGeocode(context.Background(), &mapbox.ReverseGeocodeRequest{
		Longitude: 13.39,
		Latitude:  52.51,
	})
	if err != nil {
		t.Fatalf("ReverseGeocode error: %v", err)
	}
	if len(result.Features) != 1 {
		t.Fatalf("len(Features) = %d, want 1", len(result.Features))
	}
	f := result.Features[0]
	if f.Properties.FullAddress != "Unter den Linden 1, 10117 Berlin, Germany" {
		t.Errorf("FullAddress = %q", f.Properties.FullAddress)
	}
	if f.Properties.Context.Address == nil {
		t.Fatal("expected Address context to be set")
	}
	if f.Properties.Context.Address.AddressNumber != "1" {
		t.Errorf("AddressNumber = %q, want %q", f.Properties.Context.Address.AddressNumber, "1")
	}
	if f.Properties.Context.Address.StreetName != "Unter den Linden" {
		t.Errorf("StreetName = %q", f.Properties.Context.Address.StreetName)
	}
	if f.Properties.Context.Country == nil {
		t.Fatal("expected Country context to be set")
	}
	if f.Properties.Context.Country.CountryCode != "DE" {
		t.Errorf("CountryCode = %q, want DE", f.Properties.Context.Country.CountryCode)
	}
}

func TestReverseGeocode_PointGeometry(t *testing.T) {
	// Mapbox returns Point geometry with flat [lon, lat] coordinates.
	// Regression: GeoJSONGeometry used [][]float64 (LineString), causing decode to fail.
	const body = `{
		"type": "FeatureCollection",
		"features": [{
			"type": "Feature",
			"geometry": {"type": "Point", "coordinates": [24.9384, 60.1699]},
			"properties": {
				"mapbox_id": "addr.1",
				"feature_type": "address",
				"full_address": "Mannerheimintie 1, 00100 Helsinki, Finland"
			}
		}]
	}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if _, err := w.Write([]byte(body)); err != nil {
			t.Errorf("write response: %v", err)
		}
	}))
	defer srv.Close()

	client := mapbox.NewClient(mapbox.WithAccessToken("t"), mapbox.WithBaseURL(srv.URL))
	result, err := client.ReverseGeocode(context.Background(), &mapbox.ReverseGeocodeRequest{
		Longitude: 24.9384,
		Latitude:  60.1699,
	})
	if err != nil {
		t.Fatalf("ReverseGeocode error: %v", err)
	}
	if len(result.Features) != 1 {
		t.Fatalf("len(Features) = %d, want 1", len(result.Features))
	}
	f := result.Features[0]
	if f.Geometry.Type != "Point" {
		t.Errorf("Geometry.Type = %q, want Point", f.Geometry.Type)
	}
	if len(f.Geometry.Coordinates) != 2 {
		t.Errorf("Geometry.Coordinates len = %d, want 2", len(f.Geometry.Coordinates))
	}
}

func TestReverseGeocode_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"message":"Not Authorized - Invalid Token"}`, http.StatusUnauthorized)
	}))
	defer srv.Close()

	client := mapbox.NewClient(mapbox.WithAccessToken("bad"), mapbox.WithBaseURL(srv.URL))
	_, err := client.ReverseGeocode(context.Background(), &mapbox.ReverseGeocodeRequest{Longitude: 0, Latitude: 0})
	if err == nil {
		t.Fatal("expected error for 401 response")
	}
	if !mapbox.IsUnauthorized(err) {
		t.Errorf("IsUnauthorized(err) = false, want true; err = %v", err)
	}
}

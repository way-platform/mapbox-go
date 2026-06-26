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
		json.NewEncoder(w).Encode(mapbox.FeatureCollection{Type: "FeatureCollection"})
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
		t.Errorf("expected GET, got %s", gotReq.Method)
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
		json.NewEncoder(w).Encode(fc)
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
		t.Fatalf("expected 1 feature, got %d", len(result.Features))
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
		t.Errorf("expected IsUnauthorized(err) = true; err = %v", err)
	}
}

func TestBatchReverseGeocode_JSONBody(t *testing.T) {
	var gotBody []map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			t.Errorf("Content-Type = %q, want application/json", ct)
		}
		if tok := r.URL.Query().Get("access_token"); tok != "batch-token" {
			t.Errorf("access_token = %q in URL", tok)
		}
		json.NewDecoder(r.Body).Decode(&gotBody)
		result := []*mapbox.FeatureCollection{
			{Type: "FeatureCollection"},
			{Type: "FeatureCollection"},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
	}))
	defer srv.Close()

	client := mapbox.NewClient(mapbox.WithAccessToken("batch-token"), mapbox.WithBaseURL(srv.URL))
	results, err := client.BatchReverseGeocode(context.Background(), &mapbox.BatchReverseGeocodeRequest{
		Queries: []mapbox.ReverseGeocodeQuery{
			{Longitude: 13.4, Latitude: 52.5},
			{Longitude: -0.12, Latitude: 51.5},
		},
	})
	if err != nil {
		t.Fatalf("BatchReverseGeocode error: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("expected 2 results, got %d", len(results))
	}

	// Verify request body format.
	if len(gotBody) != 2 {
		t.Fatalf("expected 2 body entries, got %d", len(gotBody))
	}
	for i, q := range gotBody {
		if types, ok := q["types"].(string); !ok || types != "address" {
			t.Errorf("query[%d].types = %v, want \"address\"", i, q["types"])
		}
	}
	if gotBody[0]["longitude"] != 13.4 {
		t.Errorf("query[0].longitude = %v, want 13.4", gotBody[0]["longitude"])
	}
}

func TestBatchReverseGeocode_LimitEnforcement(t *testing.T) {
	client := mapbox.NewClient(mapbox.WithAccessToken("t"))

	_, err := client.BatchReverseGeocode(context.Background(), &mapbox.BatchReverseGeocodeRequest{
		Queries: []mapbox.ReverseGeocodeQuery{},
	})
	if err == nil {
		t.Error("expected error for empty queries")
	}

	queries := make([]mapbox.ReverseGeocodeQuery, 1001)
	_, err = client.BatchReverseGeocode(context.Background(), &mapbox.BatchReverseGeocodeRequest{
		Queries: queries,
	})
	if err == nil {
		t.Error("expected error for > 1000 queries")
	}
}

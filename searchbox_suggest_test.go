package mapbox_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	mapbox "github.com/way-platform/mapbox-go"
)

func TestSuggest_QueryParams(t *testing.T) {
	var gotReq *http.Request
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotReq = r
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(mapbox.SuggestResponse{}); err != nil {
			t.Errorf("encode response: %v", err)
		}
	}))
	defer srv.Close()

	client := mapbox.NewClient(mapbox.WithAccessToken("my-token"), mapbox.WithBaseURL(srv.URL))
	_, err := client.Suggest(context.Background(), &mapbox.SuggestRequest{
		Query:        "Berlin",
		SessionToken: "test-session-uuid",
		Language:     "de",
		Limit:        5,
	})
	if err != nil {
		t.Fatalf("Suggest error: %v", err)
	}

	q := gotReq.URL.Query()
	if v := q.Get("q"); v != "Berlin" {
		t.Errorf("q = %q, want %q", v, "Berlin")
	}
	if v := q.Get("session_token"); v != "test-session-uuid" {
		t.Errorf("session_token = %q, want %q", v, "test-session-uuid")
	}
	if v := q.Get("language"); v != "de" {
		t.Errorf("language = %q, want %q", v, "de")
	}
	if v := q.Get("limit"); v != "5" {
		t.Errorf("limit = %q, want %q", v, "5")
	}
	if v := q.Get("access_token"); v != "my-token" {
		t.Errorf("access_token = %q, want %q", v, "my-token")
	}
	if gotReq.Method != http.MethodGet {
		t.Errorf("Method = %s, want GET", gotReq.Method)
	}
}

func TestSuggest_ProximityAndBBox(t *testing.T) {
	var gotReq *http.Request
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotReq = r
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(mapbox.SuggestResponse{}); err != nil {
			t.Errorf("encode response: %v", err)
		}
	}))
	defer srv.Close()

	client := mapbox.NewClient(mapbox.WithAccessToken("t"), mapbox.WithBaseURL(srv.URL))
	_, err := client.Suggest(context.Background(), &mapbox.SuggestRequest{
		Query:        "cafe",
		SessionToken: "sess",
		Proximity:    &mapbox.Coordinate{Longitude: 13.4, Latitude: 52.5},
		BBox:         &mapbox.BoundingBox{MinLongitude: 13.0, MinLatitude: 52.0, MaxLongitude: 14.0, MaxLatitude: 53.0},
		Countries:    []string{"DE", "AT"},
		Types:        []string{"poi", "address"},
	})
	if err != nil {
		t.Fatalf("Suggest error: %v", err)
	}

	q := gotReq.URL.Query()
	if v := q.Get("proximity"); v != "13.4,52.5" {
		t.Errorf("proximity = %q, want %q", v, "13.4,52.5")
	}
	if v := q.Get("bbox"); v != "13,52,14,53" {
		t.Errorf("bbox = %q, want %q", v, "13,52,14,53")
	}
	if v := q.Get("country"); v != "DE,AT" {
		t.Errorf("country = %q, want %q", v, "DE,AT")
	}
	if v := q.Get("types"); v != "poi,address" {
		t.Errorf("types = %q, want %q", v, "poi,address")
	}
}

func TestSuggest_ResponseParsing(t *testing.T) {
	resp := mapbox.SuggestResponse{
		Suggestions: []*mapbox.Suggestion{
			{
				MapboxID:       "poi.abc123",
				Name:           "Brandenburger Tor",
				FeatureType:    "poi",
				PlaceFormatted: "Berlin, Germany",
				FullAddress:    "Pariser Platz, 10117 Berlin, Germany",
			},
		},
		Attribution: "© Mapbox",
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			t.Errorf("encode response: %v", err)
		}
	}))
	defer srv.Close()

	client := mapbox.NewClient(mapbox.WithAccessToken("t"), mapbox.WithBaseURL(srv.URL))
	got, err := client.Suggest(context.Background(), &mapbox.SuggestRequest{
		Query:        "Brandenburger",
		SessionToken: "sess",
	})
	if err != nil {
		t.Fatalf("Suggest error: %v", err)
	}
	if len(got.Suggestions) != 1 {
		t.Fatalf("len(Suggestions) = %d, want 1", len(got.Suggestions))
	}
	s := got.Suggestions[0]
	if s.MapboxID != "poi.abc123" {
		t.Errorf("MapboxID = %q, want %q", s.MapboxID, "poi.abc123")
	}
	if s.Name != "Brandenburger Tor" {
		t.Errorf("Name = %q, want %q", s.Name, "Brandenburger Tor")
	}
}

func TestSuggest_ValidationErrors(t *testing.T) {
	client := mapbox.NewClient(mapbox.WithAccessToken("t"))

	_, err := client.Suggest(context.Background(), &mapbox.SuggestRequest{
		SessionToken: "sess",
	})
	if err == nil {
		t.Error("expected error for empty query")
	}

	_, err = client.Suggest(context.Background(), &mapbox.SuggestRequest{
		Query: "Berlin",
	})
	if err == nil {
		t.Error("expected error for missing session token")
	}
}

func TestSuggest_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"message":"Not Authorized"}`, http.StatusUnauthorized)
	}))
	defer srv.Close()

	client := mapbox.NewClient(mapbox.WithAccessToken("bad"), mapbox.WithBaseURL(srv.URL))
	_, err := client.Suggest(context.Background(), &mapbox.SuggestRequest{
		Query:        "Berlin",
		SessionToken: "sess",
	})
	if err == nil {
		t.Fatal("expected error for 401 response")
	}
	if !mapbox.IsUnauthorized(err) {
		t.Errorf("IsUnauthorized(err) = false, want true; err = %v", err)
	}
}

package mapbox_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	mapbox "github.com/way-platform/mapbox-go"
)

func TestRetrieve_RequestFormat(t *testing.T) {
	var gotReq *http.Request
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotReq = r
		w.Header().Set("Content-Type", "application/json")
		result := mapbox.RetrieveResponse{
			Type: "FeatureCollection",
			Features: []mapbox.RetrieveFeature{
				{
					Type: "Feature",
					Properties: mapbox.RetrieveFeatureProperties{
						MapboxID:    "poi.abc123",
						FeatureType: "poi",
						Name:        "Brandenburger Tor",
						Coordinates: mapbox.RetrieveCoordinates{
							Longitude: 13.3777,
							Latitude:  52.5163,
						},
					},
				},
			},
		}
		if err := json.NewEncoder(w).Encode(result); err != nil {
			t.Errorf("encode response: %v", err)
		}
	}))
	defer srv.Close()

	client := mapbox.NewClient(mapbox.WithAccessToken("my-token"), mapbox.WithBaseURL(srv.URL))
	got, err := client.Retrieve(context.Background(), &mapbox.RetrieveRequest{
		MapboxID:     "poi.abc123",
		SessionToken: "test-session-uuid",
	})
	if err != nil {
		t.Fatalf("Retrieve error: %v", err)
	}

	if path := gotReq.URL.Path; path != "/search/searchbox/v1/retrieve/poi.abc123" {
		t.Errorf("path = %q, want %q", path, "/search/searchbox/v1/retrieve/poi.abc123")
	}
	if v := gotReq.URL.Query().Get("session_token"); v != "test-session-uuid" {
		t.Errorf("session_token = %q, want %q", v, "test-session-uuid")
	}
	if v := gotReq.URL.Query().Get("access_token"); v != "my-token" {
		t.Errorf("access_token = %q, want %q", v, "my-token")
	}
	if gotReq.Method != http.MethodGet {
		t.Errorf("Method = %s, want GET", gotReq.Method)
	}

	if len(got.Features) != 1 {
		t.Fatalf("len(Features) = %d, want 1", len(got.Features))
	}
	f := got.Features[0]
	if f.Properties.Coordinates.Longitude != 13.3777 {
		t.Errorf("Longitude = %v, want %v", f.Properties.Coordinates.Longitude, 13.3777)
	}
	if f.Properties.Coordinates.Latitude != 52.5163 {
		t.Errorf("Latitude = %v, want %v", f.Properties.Coordinates.Latitude, 52.5163)
	}
}

func TestRetrieve_ValidationErrors(t *testing.T) {
	client := mapbox.NewClient(mapbox.WithAccessToken("t"))

	_, err := client.Retrieve(context.Background(), &mapbox.RetrieveRequest{
		SessionToken: "sess",
	})
	if err == nil {
		t.Error("expected error for missing mapbox_id")
	}

	_, err = client.Retrieve(context.Background(), &mapbox.RetrieveRequest{
		MapboxID: "poi.123",
	})
	if err == nil {
		t.Error("expected error for missing session token")
	}
}

func TestRetrieve_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"message":"Not Found"}`, http.StatusNotFound)
	}))
	defer srv.Close()

	client := mapbox.NewClient(mapbox.WithAccessToken("t"), mapbox.WithBaseURL(srv.URL))
	_, err := client.Retrieve(context.Background(), &mapbox.RetrieveRequest{
		MapboxID:     "poi.nonexistent",
		SessionToken: "sess",
	})
	if err == nil {
		t.Fatal("expected error for 404 response")
	}
	if !mapbox.IsNotFound(err) {
		t.Errorf("IsNotFound(err) = false, want true; err = %v", err)
	}
}

package mapbox_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	mapbox "github.com/way-platform/mapbox-go"
)

func TestBatchReverseGeocode_JSONBody(t *testing.T) {
	var gotBody []map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			t.Errorf("Content-Type = %q, want application/json", ct)
		}
		if tok := r.URL.Query().Get("access_token"); tok != "batch-token" {
			t.Errorf("access_token = %q in URL", tok)
		}
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Errorf("decode request body: %v", err)
		}
		result := map[string]any{
			"batch": []map[string]any{
				{"type": "FeatureCollection", "features": []any{}},
				{"type": "FeatureCollection", "features": []any{}},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(result); err != nil {
			t.Errorf("encode response: %v", err)
		}
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
		t.Errorf("len(results) = %d, want 2", len(results))
	}

	// Verify request body format.
	if len(gotBody) != 2 {
		t.Fatalf("len(body) = %d, want 2", len(gotBody))
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

func TestBatchReverseGeocode_PermanentParam(t *testing.T) {
	tests := []struct {
		name      string
		permanent bool
		wantParam string
	}{
		{"temporary omits param", false, ""},
		{"permanent sets param", true, "true"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var gotParam string
			var gotBody []map[string]any
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotParam = r.URL.Query().Get("permanent")
				if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
					t.Errorf("decode body: %v", err)
				}
				result := map[string]any{"batch": []map[string]any{{"type": "FeatureCollection", "features": []any{}}}}
				w.Header().Set("Content-Type", "application/json")
				if err := json.NewEncoder(w).Encode(result); err != nil {
					t.Errorf("encode response: %v", err)
				}
			}))
			defer srv.Close()

			client := mapbox.NewClient(mapbox.WithAccessToken("t"), mapbox.WithBaseURL(srv.URL))
			_, err := client.BatchReverseGeocode(context.Background(), &mapbox.BatchReverseGeocodeRequest{
				Queries:   []mapbox.ReverseGeocodeQuery{{Longitude: 13.4, Latitude: 52.5}},
				Permanent: tc.permanent,
			})
			if err != nil {
				t.Fatalf("BatchReverseGeocode error: %v", err)
			}
			if gotParam != tc.wantParam {
				t.Errorf("permanent param = %q, want %q", gotParam, tc.wantParam)
			}
			// permanent must not appear in the JSON body - it's a query param only
			if len(gotBody) > 0 {
				if _, ok := gotBody[0]["permanent"]; ok {
					t.Errorf("permanent field must not appear in request body")
				}
			}
		})
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

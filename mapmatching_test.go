package mapbox_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	mapbox "github.com/way-platform/mapbox-go"
)

func TestMapMatch_FormBody(t *testing.T) {
	var gotReq *http.Request
	var gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotReq = r
		body, _ := io.ReadAll(r.Body)
		gotBody = string(body)
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(mapbox.MapMatchResponse{Code: mapbox.CodeOK}); err != nil {
			t.Errorf("encode response: %v", err)
		}
	}))
	defer srv.Close()

	client := mapbox.NewClient(
		mapbox.WithAccessToken("test-token"),
		mapbox.WithBaseURL(srv.URL),
	)
	_, err := client.MapMatch(context.Background(), &mapbox.MapMatchRequest{
		Coordinates: []mapbox.Coordinate{
			{Longitude: 13.4, Latitude: 52.5},
			{Longitude: 13.5, Latitude: 52.6},
		},
	})
	if err != nil {
		t.Fatalf("MapMatch error: %v", err)
	}

	// Verify HTTP method and content-type.
	if gotReq.Method != http.MethodPost {
		t.Errorf("Method = %s, want POST", gotReq.Method)
	}
	if ct := gotReq.Header.Get("Content-Type"); ct != "application/x-www-form-urlencoded" {
		t.Errorf("Content-Type = %q, want application/x-www-form-urlencoded", ct)
	}

	// Verify access_token is in the URL query, not the body.
	if tok := gotReq.URL.Query().Get("access_token"); tok != "test-token" {
		t.Errorf("access_token = %q, want %q", tok, "test-token")
	}
	if strings.Contains(gotBody, "access_token") {
		t.Error("access_token must not appear in the form body")
	}

	// Verify form body fields.
	form, err := url.ParseQuery(gotBody)
	if err != nil {
		t.Fatalf("parse form body: %v", err)
	}
	if coords := form.Get("coordinates"); coords != "13.4,52.5;13.5,52.6" {
		t.Errorf("coordinates = %q, want %q", coords, "13.4,52.5;13.5,52.6")
	}
	if form.Get("tidy") != "true" {
		t.Errorf("tidy = %q, want %q", form.Get("tidy"), "true")
	}
	if form.Get("geometries") != "geojson" {
		t.Errorf("geometries = %q, want %q", form.Get("geometries"), "geojson")
	}
	if form.Get("overview") != "full" {
		t.Errorf("overview = %q, want %q", form.Get("overview"), "full")
	}
}

func TestMapMatch_RadiusesAndTimestamps(t *testing.T) {
	var gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		gotBody = string(body)
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(mapbox.MapMatchResponse{Code: mapbox.CodeOK}); err != nil {
			t.Errorf("encode response: %v", err)
		}
	}))
	defer srv.Close()

	client := mapbox.NewClient(mapbox.WithAccessToken("t"), mapbox.WithBaseURL(srv.URL))
	_, err := client.MapMatch(context.Background(), &mapbox.MapMatchRequest{
		Coordinates: []mapbox.Coordinate{
			{Longitude: 1.0, Latitude: 2.0},
			{Longitude: 3.0, Latitude: 4.0},
		},
		Radiuses:   []float64{25.0, 25.0},
		Timestamps: []float64{1700000000.0, 1700000010.0},
	})
	if err != nil {
		t.Fatalf("MapMatch error: %v", err)
	}

	form, _ := url.ParseQuery(gotBody)
	if r := form.Get("radiuses"); r != "25;25" {
		t.Errorf("radiuses = %q, want %q", r, "25;25")
	}
	if ts := form.Get("timestamps"); ts != "1700000000;1700000010" {
		t.Errorf("timestamps = %q, want %q", ts, "1700000000;1700000010")
	}
}

func TestMapMatch_NoMatch_NotAnError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(mapbox.MapMatchResponse{
			Code:    mapbox.CodeNoMatch,
			Message: "Could not match the trace",
		}); err != nil {
			t.Errorf("encode response: %v", err)
		}
	}))
	defer srv.Close()

	client := mapbox.NewClient(mapbox.WithAccessToken("t"), mapbox.WithBaseURL(srv.URL))
	resp, err := client.MapMatch(context.Background(), &mapbox.MapMatchRequest{
		Coordinates: []mapbox.Coordinate{{1, 2}, {3, 4}},
	})
	if err != nil {
		t.Fatalf("MapMatch error = %v, want nil", err)
	}
	if resp.Code != mapbox.CodeNoMatch {
		t.Errorf("Code = %q, want %q", resp.Code, mapbox.CodeNoMatch)
	}
	if resp.Code.IsSuccess() {
		t.Error("NoMatch.IsSuccess() = true, want false")
	}
}

func TestMapMatch_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"code":"InvalidInput","message":"bad request"}`, http.StatusUnprocessableEntity)
	}))
	defer srv.Close()

	client := mapbox.NewClient(mapbox.WithAccessToken("t"), mapbox.WithBaseURL(srv.URL))
	_, err := client.MapMatch(context.Background(), &mapbox.MapMatchRequest{
		Coordinates: []mapbox.Coordinate{{1, 2}, {3, 4}},
	})
	if err == nil {
		t.Fatal("expected error for 422 response")
	}
	if !mapbox.IsInvalidInput(err) {
		t.Errorf("IsInvalidInput(err) = false, want true; err = %v", err)
	}
}

func TestMapMatch_WithTransport_CalledOnEveryRequest(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(mapbox.MapMatchResponse{Code: mapbox.CodeOK}); err != nil {
			t.Errorf("encode response: %v", err)
		}
	}))
	defer srv.Close()

	var transportCalls int
	rt := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		transportCalls++
		// Verify access_token was injected before reaching the transport.
		if tok := req.URL.Query().Get("access_token"); tok != "my-token" {
			t.Errorf("WithTransport: expected access_token %q in URL, got %q", "my-token", tok)
		}
		return http.DefaultTransport.RoundTrip(req)
	})

	client := mapbox.NewClient(
		mapbox.WithAccessToken("my-token"),
		mapbox.WithBaseURL(srv.URL),
		mapbox.WithTransport(rt),
	)

	for range 3 {
		_, _ = client.MapMatch(context.Background(), &mapbox.MapMatchRequest{
			Coordinates: []mapbox.Coordinate{{1, 2}, {3, 4}},
		})
	}
	if transportCalls != 3 {
		t.Errorf("expected WithTransport called 3 times, got %d", transportCalls)
	}
}

func TestMapMatch_Retry_On500(t *testing.T) {
	attempts := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 3 {
			http.Error(w, "server error", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(mapbox.MapMatchResponse{Code: mapbox.CodeOK}); err != nil {
			t.Errorf("encode response: %v", err)
		}
	}))
	defer srv.Close()

	noopSleep := func(_ context.Context, _ time.Duration) bool { return true }
	client := mapbox.NewClient(
		mapbox.WithAccessToken("t"),
		mapbox.WithBaseURL(srv.URL),
		mapbox.WithRetryCount(3),
		mapbox.WithRetrySleepForTest(noopSleep),
	)
	resp, err := client.MapMatch(context.Background(), &mapbox.MapMatchRequest{
		Coordinates: []mapbox.Coordinate{{1, 2}, {3, 4}},
	})
	if err != nil {
		t.Fatalf("expected success after retries, got %v", err)
	}
	if !resp.Code.IsSuccess() {
		t.Errorf("expected Ok code, got %q", resp.Code)
	}
	if attempts != 3 {
		t.Errorf("expected 3 attempts, got %d", attempts)
	}
}

func TestMapMatch_NoRetry_On400(t *testing.T) {
	attempts := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		http.Error(w, "bad request", http.StatusBadRequest)
	}))
	defer srv.Close()

	noopSleep := func(_ context.Context, _ time.Duration) bool { return true }
	client := mapbox.NewClient(
		mapbox.WithAccessToken("t"),
		mapbox.WithBaseURL(srv.URL),
		mapbox.WithRetryCount(3),
		mapbox.WithRetrySleepForTest(noopSleep),
	)
	_, err := client.MapMatch(context.Background(), &mapbox.MapMatchRequest{
		Coordinates: []mapbox.Coordinate{{1, 2}, {3, 4}},
	})
	if err == nil {
		t.Fatal("expected error for 400 response")
	}
	if attempts != 1 {
		t.Errorf("expected exactly 1 attempt (no retry on 400), got %d", attempts)
	}
}

func TestMapMatch_ValidationErrors(t *testing.T) {
	client := mapbox.NewClient(mapbox.WithAccessToken("t"))

	_, err := client.MapMatch(context.Background(), &mapbox.MapMatchRequest{
		Coordinates: []mapbox.Coordinate{{1, 2}}, // only 1 coordinate
	})
	if err == nil {
		t.Error("expected error for < 2 coordinates")
	}

	coords := make([]mapbox.Coordinate, 101)
	_, err = client.MapMatch(context.Background(), &mapbox.MapMatchRequest{Coordinates: coords})
	if err == nil {
		t.Error("expected error for > 100 coordinates")
	}

	_, err = client.MapMatch(context.Background(), &mapbox.MapMatchRequest{
		Coordinates: []mapbox.Coordinate{{1, 2}, {3, 4}},
		Radiuses:    []float64{5.0}, // wrong length
	})
	if err == nil {
		t.Error("expected error for mismatched radiuses length")
	}
}

func TestEncodeCoordinates(t *testing.T) {
	got := encodeCoordinatesForTest(t, []mapbox.Coordinate{
		{Longitude: 13.4, Latitude: 52.5},
		{Longitude: -0.1278, Latitude: 51.5074},
	})
	want := "13.4,52.5;-0.1278,51.5074"
	if got != want {
		t.Errorf("encodeCoordinates = %q, want %q", got, want)
	}
}

func TestCodeIsSuccess(t *testing.T) {
	if !mapbox.CodeOK.IsSuccess() {
		t.Error("CodeOK.IsSuccess() should be true")
	}
	for _, code := range []mapbox.Code{
		mapbox.CodeNoMatch,
		mapbox.CodeNoSegment,
		mapbox.CodeInvalidInput,
		mapbox.CodeProfileNotFound,
		mapbox.CodeTooManyCoordinates,
	} {
		if code.IsSuccess() {
			t.Errorf("%q.IsSuccess() should be false", code)
		}
	}
}

// encodeCoordinatesForTest exercises the encoding logic through a real MapMatch call
// by capturing the form body.
func encodeCoordinatesForTest(t *testing.T, coords []mapbox.Coordinate) string {
	t.Helper()
	var captured string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		form, _ := url.ParseQuery(string(body))
		captured = form.Get("coordinates")
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(mapbox.MapMatchResponse{Code: mapbox.CodeOK}); err != nil {
			t.Errorf("encode response: %v", err)
		}
	}))
	defer srv.Close()
	client := mapbox.NewClient(mapbox.WithAccessToken("t"), mapbox.WithBaseURL(srv.URL))
	if _, err := client.MapMatch(context.Background(), &mapbox.MapMatchRequest{Coordinates: coords}); err != nil {
		t.Fatalf("MapMatch error: %v", err)
	}
	return captured
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

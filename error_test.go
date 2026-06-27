package mapbox_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	mapbox "github.com/way-platform/mapbox-go"
)

func TestError_MessageParsed(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"message":"Not Authorized - Invalid Token"}`, http.StatusUnauthorized)
	}))
	defer srv.Close()

	client := mapbox.NewClient(mapbox.WithAccessToken("bad"), mapbox.WithBaseURL(srv.URL))
	_, err := client.ReverseGeocode(t.Context(), &mapbox.ReverseGeocodeRequest{})
	if err == nil {
		t.Fatal("expected error")
	}

	var apiErr *mapbox.Error
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *mapbox.Error, got %T", err)
	}
	if apiErr.Message != "Not Authorized - Invalid Token" {
		t.Errorf("Message = %q, want %q", apiErr.Message, "Not Authorized - Invalid Token")
	}
	want := "mapbox: http 401: Not Authorized - Invalid Token"
	if got := apiErr.Error(); got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
}

func TestError_CodeOf(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"code":"TooManyCoordinates","message":"coordinates must have between 2 and 100 items"}`, http.StatusUnprocessableEntity)
	}))
	defer srv.Close()

	client := mapbox.NewClient(mapbox.WithAccessToken("t"), mapbox.WithBaseURL(srv.URL))
	_, err := client.MapMatch(t.Context(), &mapbox.MapMatchRequest{
		Coordinates: []mapbox.Coordinate{{1, 2}, {3, 4}},
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if got := mapbox.CodeOf(err); got != mapbox.CodeTooManyCoordinates {
		t.Errorf("CodeOf(err) = %q, want %q", got, mapbox.CodeTooManyCoordinates)
	}
	if !mapbox.IsInvalidInput(err) {
		t.Errorf("IsInvalidInput(err) = false, want true")
	}
}

func TestError_CodeOf_NoCode(t *testing.T) {
	if got := mapbox.CodeOf(nil); got != mapbox.CodeUnknown {
		t.Errorf("CodeOf(nil) = %q, want %q", got, mapbox.CodeUnknown)
	}
}

func TestError_NonJSONBodyFallback(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("internal server error"))
	}))
	defer srv.Close()

	client := mapbox.NewClient(mapbox.WithAccessToken("t"), mapbox.WithBaseURL(srv.URL))
	_, err := client.ReverseGeocode(t.Context(), &mapbox.ReverseGeocodeRequest{})
	if err == nil {
		t.Fatal("expected error")
	}

	var apiErr *mapbox.Error
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *mapbox.Error, got %T", err)
	}
	if apiErr.Message != "" {
		t.Errorf("Message = %q, want empty for non-JSON body", apiErr.Message)
	}
	if mapbox.CodeOf(err) != mapbox.CodeUnknown {
		t.Errorf("CodeOf(err) = %q, want CodeUnknown for non-JSON body", mapbox.CodeOf(err))
	}
	if apiErr.Body != "internal server error" {
		t.Errorf("Body = %q, want %q", apiErr.Body, "internal server error")
	}
}

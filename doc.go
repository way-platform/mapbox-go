// Package mapbox provides a Go client for the Mapbox APIs.
//
// Supported endpoints:
//   - Map Matching v5: POST /matching/v5/{profile}
//   - Geocoding v6 Reverse: GET /search/geocode/v6/reverse
//   - Geocoding v6 Batch: POST /search/geocode/v6/batch
//
// Authentication uses the Mapbox access token passed via [WithAccessToken].
// The token is appended as an ?access_token= query parameter on every request.
//
// To instrument API calls (e.g. for metrics), inject a custom [http.RoundTripper]
// via [WithTransport]. The SDK layers auth and retry on top of the provided transport.
package mapbox

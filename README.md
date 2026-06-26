# mapbox-go

Go client for the Mapbox APIs.

## Supported endpoints

| API | Endpoint |
|-----|---------|
| Map Matching v5 | `POST /matching/v5/{profile}` |
| Geocoding v6 Reverse | `GET /search/geocode/v6/reverse` |
| Geocoding v6 Batch | `POST /search/geocode/v6/batch` |

## Installation

```bash
go get github.com/way-platform/mapbox-go
```

## Usage

```go
client := mapbox.NewClient(
    mapbox.WithAccessToken(os.Getenv("MAPBOX_ACCESS_TOKEN")),
)

// Map Matching
resp, err := client.MapMatch(ctx, &mapbox.MapMatchRequest{
    Coordinates: []mapbox.Coordinate{
        {Longitude: 13.418946, Latitude: 52.500217},
        {Longitude: 13.419946, Latitude: 52.501217},
    },
    Radiuses:   []float64{25.0, 25.0},   // meters
    Timestamps: []float64{1700000000, 1700000010}, // Unix seconds
})
if err != nil {
    log.Fatal(err)
}
if !resp.Code.IsSuccess() {
    // Handle NoMatch / NoSegment per your degradation policy.
    log.Printf("no match: %s", resp.Code)
}

// Reverse geocode
fc, err := client.ReverseGeocode(ctx, &mapbox.ReverseGeocodeRequest{
    Longitude: 13.405,
    Latitude:  52.52,
})

// Batch reverse geocode (up to 1000 per request)
results, err := client.BatchReverseGeocode(ctx, &mapbox.BatchReverseGeocodeRequest{
    Queries: []mapbox.ReverseGeocodeQuery{
        {Longitude: 13.405, Latitude: 52.52},
        {Longitude: -0.1278, Latitude: 51.5074},
    },
})
```

## Observability

Inject a custom `http.RoundTripper` via `WithTransport` to observe all API calls
without adding GCP-specific code to this SDK:

```go
client := mapbox.NewClient(
    mapbox.WithAccessToken(token),
    mapbox.WithTransport(myMetricsTransport),
)
```

The transport sees every outbound request after auth (`?access_token=`) is injected.

## Retries

Retry is disabled by default. Enable with `WithRetryCount`:

```go
client := mapbox.NewClient(
    mapbox.WithAccessToken(token),
    mapbox.WithRetryCount(3), // retries on 429 and 5xx
)
```

## Error handling

HTTP errors return `*mapbox.Error`. Use the helper functions:

```go
_, err := client.MapMatch(ctx, req)
if mapbox.IsRateLimited(err) { ... }
if mapbox.IsUnauthorized(err) { ... }
```

Map Matching semantic failures (no route found) are **not** Go errors. Check
`resp.Code`:

```go
if !resp.Code.IsSuccess() {
    // resp.Code is CodeNoMatch or CodeNoSegment
}
```

## CLI

```bash
# Install
go install github.com/way-platform/mapbox-go/cmd/mapbox@latest

# Save credentials
mapbox auth login --token $MAPBOX_ACCESS_TOKEN

# Map match
mapbox map-match --input coords.json

# Reverse geocode
mapbox geocode --lon 13.405 --lat 52.52

# Batch reverse geocode
mapbox geocode-batch --input coords.json
```

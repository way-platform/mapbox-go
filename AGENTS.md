# Agent Instructions

Go SDK for the Mapbox APIs: Map Matching v5, Geocoding v6 Reverse, and
Geocoding v6 Batch.

## Build

```bash
mise install          # install tools (go 1.24, golangci-lint 2.10)
mise run build        # full CI: lint → test → tidy → cli → diff
mise run lint         # golangci-lint --fix
mise run test         # go test -count=1 -cover ./...
mise run cli          # go install ./... (builds mapbox CLI binary)
```

## Authentication

Mapbox uses `?access_token=` as a URL query parameter. `Authorization: Bearer`
is **not supported**. The `tokenTransport` in `client.go` injects the token
into every request URL automatically — never set it in headers.

## Client Architecture

```
tokenTransport       ← injects ?access_token= on every request
  └── retryTransport ← optional; enabled via WithRetryCount(n)
        └── caller transport / http.DefaultTransport
```

`WithTransport(rt)` replaces `http.DefaultTransport` as the base, allowing
callers to inject instrumentation (metrics, tracing) without any observability
code inside the SDK itself.

## Error Handling

Two layers:

- **`*Error`** — HTTP-level errors (4xx/5xx). Check with `IsNotFound`,
  `IsUnauthorized`, `IsForbidden`, `IsRateLimited`, `IsInvalidInput`,
  `IsServerError`.
- **`Code`** — in-band semantic codes from Map Matching 200 OK responses
  (`"Ok"`, `"NoMatch"`, `"NoSegment"`, etc.). These are **not** Go errors.
  Check with `resp.Code.IsSuccess()`.

## Map Matching

POST `/matching/v5/{profile}` with `application/x-www-form-urlencoded` body.
Coordinates in body as `lng,lat;lng,lat;...` (longitude first). `access_token`
stays in the URL query, not the body. Min 2, max 100 coordinates per request.

`MapMatchResponse.Tracepoints` is `[]*Tracepoint` — `nil` elements represent
unmatched coordinates. A `Code` of `"NoMatch"` or `"NoSegment"` in a 200 OK
is not a Go error; check `resp.Code.IsSuccess()`. Always sent: `tidy=true`,
`geometries=geojson`, `overview=full`.

## Search Box

- Suggest: `GET /search/searchbox/v1/suggest?q=X&session_token=Y`
- Retrieve: `GET /search/searchbox/v1/retrieve/{mapbox_id}?session_token=Y`

Both require a `session_token` (UUIDv4). The same token must be used across the
entire suggest→retrieve sequence — one session = one billable unit. Sessions
expire after 180s or once `/retrieve` is called. `mapbox_id` from a suggestion
is only valid for the duration of that session (180s); do not store it.

## Geocoding v6

- Reverse: `GET /search/geocode/v6/reverse?longitude=X&latitude=Y`
- Batch: `POST /search/geocode/v6/batch` with a JSON array of query objects;
  always include `"types": "address"`; max 1000 per request.

Response `context` is nested (v6 shape, not flat):
`context.address.address_number`, `context.address.street_name`. Nil-check
each context level before accessing fields.

## CLI Architecture

```
cli/
├── cli.go       # Credentials, Store interface, FileStore
└── command.go   # NewCommand() — map-match, geocode, geocode-batch, auth
cmd/mapbox/
└── main.go      # Thin entry point: wires FileStore to os.UserConfigDir()
```

## Skills

- **way-go-style** — `.agents/skills/way-go-style/SKILL.md`: idiomatic Go,
  naming, error handling, testing conventions.

## Conventions

- Testing: standard `testing` + `github.com/google/go-cmp/cmp` only. No
  Testify or other frameworks.
- Linting: GolangCI-Lint v2, configured in `.golangci.yml`.
- Commit subject: max 60 characters (enforced by commitlint in CI). Conventional commit prefix counts toward the limit.
- Commit subject must not be sentence-case: first word after the prefix must be lowercase (`fix(x): add foo`, not `fix(x): Add foo`).

## Retry

`retryTransport` retries on 429 and 5xx using exponential backoff with full
jitter (base 500ms, cap 10s) and respects `Retry-After`. Default retry count
is 0 (opt-in via `WithRetryCount`). Use `WithRetrySleepForTest` (exported via
`export_test.go`) to inject a no-op sleep in tests.

package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"

	"github.com/spf13/cobra"
	mapbox "github.com/way-platform/mapbox-go"
)

// NewCommand builds the Cobra command tree for the Mapbox CLI.
func NewCommand(opts ...Option) *cobra.Command {
	cfg := &config{}
	for _, opt := range opts {
		opt(cfg)
	}

	root := &cobra.Command{
		Use:   "mapbox",
		Short: "Mapbox API CLI",
		Long:  "A command-line interface for the Mapbox Map Matching and Geocoding APIs.",
	}

	root.AddGroup(
		&cobra.Group{ID: "api", Title: "API Commands:"},
		&cobra.Group{ID: "auth", Title: "Authentication:"},
		&cobra.Group{ID: "utils", Title: "Utilities:"},
	)
	root.SetHelpCommandGroupID("utils")
	root.SetCompletionCommandGroupID("utils")

	root.AddCommand(
		newMapMatchCommand(cfg),
		newGeocodeCommand(cfg),
		newGeocodeBatchCommand(cfg),
		newSuggestCommand(cfg),
		newRetrieveCommand(cfg),
		newAuthCommand(cfg),
	)
	return root
}

// newMapMatchCommand creates the map-match command.
func newMapMatchCommand(cfg *config) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "map-match",
		Short:   "Match GPS coordinates to roads",
		GroupID: "api",
		Long: `Submit GPS coordinates to the Mapbox Map Matching API.

Input file must be a JSON object with a "coordinates" array:
  {"coordinates": [{"longitude": 13.4, "latitude": 52.5}, ...],
   "radiuses": [25.0, 25.0],
   "timestamps": [1700000000, 1700000010]}

The "radiuses" and "timestamps" fields are optional.`,
	}
	tokenFlag := cmd.Flags().String("token", "", "Mapbox access token (overrides stored credentials)")
	inputFile := cmd.Flags().String("input", "", "path to JSON file with coordinates")
	_ = cmd.MarkFlagRequired("input")

	cmd.RunE = func(cmd *cobra.Command, _ []string) error {
		client, err := newClient(cfg, *tokenFlag)
		if err != nil {
			return err
		}
		data, err := os.ReadFile(*inputFile)
		if err != nil {
			return fmt.Errorf("read input file: %w", err)
		}
		var req mapbox.MapMatchRequest
		if err := json.Unmarshal(data, &req); err != nil {
			return fmt.Errorf("parse input file: %w", err)
		}
		resp, err := client.MapMatch(cmd.Context(), &req)
		if err != nil {
			return err
		}
		out, _ := json.MarshalIndent(resp, "", "  ")
		cmd.Println(string(out))
		return nil
	}
	return cmd
}

// newGeocodeCommand creates the geocode command.
func newGeocodeCommand(cfg *config) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "geocode",
		Short:   "Reverse geocode a coordinate to an address",
		GroupID: "api",
	}
	tokenFlag := cmd.Flags().String("token", "", "Mapbox access token (overrides stored credentials)")
	lonFlag := cmd.Flags().Float64("lon", 0, "longitude")
	latFlag := cmd.Flags().Float64("lat", 0, "latitude")
	_ = cmd.MarkFlagRequired("lon")
	_ = cmd.MarkFlagRequired("lat")

	cmd.RunE = func(cmd *cobra.Command, _ []string) error {
		client, err := newClient(cfg, *tokenFlag)
		if err != nil {
			return err
		}
		resp, err := client.ReverseGeocode(cmd.Context(), &mapbox.ReverseGeocodeRequest{
			Longitude: *lonFlag,
			Latitude:  *latFlag,
		})
		if err != nil {
			return err
		}
		out, _ := json.MarshalIndent(resp, "", "  ")
		cmd.Println(string(out))
		return nil
	}
	return cmd
}

// newGeocodeBatchCommand creates the geocode-batch command.
func newGeocodeBatchCommand(cfg *config) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "geocode-batch",
		Short:   "Batch reverse geocode coordinates to addresses",
		GroupID: "api",
		Long: `Submit up to 1000 coordinates for reverse geocoding.

Input file must be a JSON array of coordinate objects:
  [{"longitude": 13.4, "latitude": 52.5}, ...]`,
	}
	tokenFlag := cmd.Flags().String("token", "", "Mapbox access token (overrides stored credentials)")
	inputFile := cmd.Flags().String("input", "", "path to JSON file with coordinates array")
	_ = cmd.MarkFlagRequired("input")

	cmd.RunE = func(cmd *cobra.Command, _ []string) error {
		client, err := newClient(cfg, *tokenFlag)
		if err != nil {
			return err
		}
		data, err := os.ReadFile(*inputFile)
		if err != nil {
			return fmt.Errorf("read input file: %w", err)
		}
		var queries []mapbox.ReverseGeocodeQuery
		if err := json.Unmarshal(data, &queries); err != nil {
			return fmt.Errorf("parse input file: %w", err)
		}
		results, err := client.BatchReverseGeocode(cmd.Context(), &mapbox.BatchReverseGeocodeRequest{
			Queries: queries,
		})
		if err != nil {
			return err
		}
		out, _ := json.MarshalIndent(results, "", "  ")
		cmd.Println(string(out))
		return nil
	}
	return cmd
}

// newSuggestCommand creates the suggest command.
func newSuggestCommand(cfg *config) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "suggest",
		Short:   "Search Box autocomplete suggestions",
		GroupID: "api",
		Long: `Return autocomplete suggestions from the Mapbox Search Box API.

Use the same --session-token for the paired retrieve call. The session token
must be a UUIDv4 and is required for billing.`,
	}
	tokenFlag := cmd.Flags().String("token", "", "Mapbox access token (overrides stored credentials)")
	queryFlag := cmd.Flags().String("query", "", "search query string")
	sessionFlag := cmd.Flags().String("session-token", "", "UUIDv4 session token")
	limitFlag := cmd.Flags().Int("limit", 0, "max number of suggestions (default: API default)")
	languageFlag := cmd.Flags().String("language", "", "IETF language tag (e.g. en, de)")
	_ = cmd.MarkFlagRequired("query")
	_ = cmd.MarkFlagRequired("session-token")

	cmd.RunE = func(cmd *cobra.Command, _ []string) error {
		client, err := newClient(cfg, *tokenFlag)
		if err != nil {
			return err
		}
		req := &mapbox.SuggestRequest{
			Query:        *queryFlag,
			SessionToken: *sessionFlag,
			Language:     *languageFlag,
			Limit:        *limitFlag,
		}
		resp, err := client.Suggest(cmd.Context(), req)
		if err != nil {
			return err
		}
		out, _ := json.MarshalIndent(resp, "", "  ")
		cmd.Println(string(out))
		return nil
	}
	return cmd
}

// newRetrieveCommand creates the retrieve command.
func newRetrieveCommand(cfg *config) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "retrieve",
		Short:   "Resolve a Search Box suggestion to a full place record",
		GroupID: "api",
	}
	tokenFlag := cmd.Flags().String("token", "", "Mapbox access token (overrides stored credentials)")
	mapboxIDFlag := cmd.Flags().String("mapbox-id", "", "mapbox_id from a suggest response")
	sessionFlag := cmd.Flags().String("session-token", "", "UUIDv4 session token (must match the suggest call)")
	languageFlag := cmd.Flags().String("language", "", "IETF language tag (e.g. en, de)")
	_ = cmd.MarkFlagRequired("mapbox-id")
	_ = cmd.MarkFlagRequired("session-token")

	cmd.RunE = func(cmd *cobra.Command, _ []string) error {
		client, err := newClient(cfg, *tokenFlag)
		if err != nil {
			return err
		}
		resp, err := client.Retrieve(cmd.Context(), &mapbox.RetrieveRequest{
			MapboxID:     *mapboxIDFlag,
			SessionToken: *sessionFlag,
			Language:     *languageFlag,
		})
		if err != nil {
			return err
		}
		out, _ := json.MarshalIndent(resp, "", "  ")
		cmd.Println(string(out))
		return nil
	}
	return cmd
}

// newAuthCommand creates the auth command group with login/logout subcommands.
func newAuthCommand(cfg *config) *cobra.Command {
	auth := &cobra.Command{
		Use:     "auth",
		Short:   "Manage Mapbox API credentials",
		GroupID: "auth",
	}
	auth.AddCommand(newAuthLoginCommand(cfg), newAuthLogoutCommand(cfg))
	return auth
}

func newAuthLoginCommand(cfg *config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "login",
		Short: "Save a Mapbox access token to disk",
	}
	tokenFlag := cmd.Flags().String("token", "", "Mapbox access token to store")
	_ = cmd.MarkFlagRequired("token")

	cmd.RunE = func(cmd *cobra.Command, _ []string) error {
		if cfg.credentialStore == nil {
			return fmt.Errorf("no credential store configured")
		}
		creds := Credentials{AccessToken: *tokenFlag}
		if err := cfg.credentialStore.Write(creds); err != nil {
			return fmt.Errorf("save credentials: %w", err)
		}
		cmd.Println("Credentials saved.")
		return nil
	}
	return cmd
}

func newAuthLogoutCommand(cfg *config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "logout",
		Short: "Remove stored Mapbox credentials",
	}
	cmd.RunE = func(cmd *cobra.Command, _ []string) error {
		if cfg.credentialStore == nil {
			return fmt.Errorf("no credential store configured")
		}
		if err := cfg.credentialStore.Clear(); err != nil {
			return fmt.Errorf("clear credentials: %w", err)
		}
		cmd.Println("Credentials removed.")
		return nil
	}
	return cmd
}

// newClient constructs a mapbox.Client from the token flag or the stored credentials.
func newClient(cfg *config, tokenFlag string) (*mapbox.Client, error) {
	token := tokenFlag
	if token == "" && cfg.credentialStore != nil {
		var creds Credentials
		if err := cfg.credentialStore.Read(&creds); err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				return nil, fmt.Errorf("no credentials found; run `mapbox auth login --token <TOKEN>` first")
			}
			return nil, fmt.Errorf("read credentials: %w", err)
		}
		token = creds.AccessToken
	}
	if token == "" {
		return nil, fmt.Errorf("no access token: use --token or run `mapbox auth login`")
	}

	opts := []mapbox.Option{mapbox.WithAccessToken(token)}
	if cfg.httpClient != nil {
		opts = append(opts, mapbox.WithHTTPClient(cfg.httpClient))
	}
	return mapbox.NewClient(opts...), nil
}

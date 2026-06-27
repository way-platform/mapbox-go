package main

import (
	"context"
	"os"
	"path/filepath"

	"github.com/way-platform/mapbox-go/cli"
)

func main() {
	configDir, err := os.UserConfigDir()
	if err != nil {
		configDir = filepath.Join(os.Getenv("HOME"), ".config")
	}
	credPath := filepath.Join(configDir, "mapbox-go", "credentials.json")

	cmd := cli.NewCommand(
		cli.WithCredentialStore(cli.NewFileStore(credPath)),
	)
	if err := cmd.ExecuteContext(context.Background()); err != nil {
		os.Exit(1)
	}
}

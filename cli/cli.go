// Package cli provides credentials management and Cobra commands for the
// Mapbox API CLI.
package cli

import (
	"encoding/json"
	"errors"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
)

// Credentials holds Mapbox API credentials persisted to disk.
type Credentials struct {
	AccessToken string `json:"accessToken"`
}

// Store reads and writes JSON-serializable data to a persistent backing store.
type Store interface {
	// Read deserializes stored data into target.
	// Returns [fs.ErrNotExist] if no data is stored yet.
	Read(target any) error
	// Write serializes data and persists it.
	Write(data any) error
	// Clear removes stored data.
	Clear() error
}

// Option configures the CLI command tree.
type Option func(*config)

type config struct {
	credentialStore Store
	httpClient      *http.Client
}

// WithCredentialStore sets the credential store used to persist the access token.
func WithCredentialStore(s Store) Option {
	return func(c *config) { c.credentialStore = s }
}

// WithHTTPClient sets a custom HTTP client passed to the SDK client.
func WithHTTPClient(hc *http.Client) Option {
	return func(c *config) { c.httpClient = hc }
}

// FileStore is a JSON file-backed [Store].
type FileStore struct {
	path string
}

// NewFileStore creates a new file-backed store that persists JSON at path.
func NewFileStore(path string) *FileStore {
	return &FileStore{path: path}
}

func (s *FileStore) Read(target any) error {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fs.ErrNotExist
		}
		return err
	}
	return json.Unmarshal(data, target)
}

func (s *FileStore) Write(data any) error {
	b, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o700); err != nil {
		return err
	}
	return os.WriteFile(s.path, b, 0o600)
}

func (s *FileStore) Clear() error {
	err := os.Remove(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}

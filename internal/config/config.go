// Package config resolves the LeakRadar API key and other CLI settings from
// (in priority order) an explicit flag, the LEAKRADAR_API_KEY environment
// variable, the OS keychain, and finally a local config file — and provides
// `leakradar-cli auth set/status/clear` the means to write to the keychain or,
// where no keychain is available (e.g. headless Linux), a 0600 config file.
package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/zalando/go-keyring"
)

const (
	keyringService = "leakradar-cli"
	keyringUser    = "api-key"
	envVar         = "LEAKRADAR_API_KEY"
	appDirName     = "leakradar-cli"
	configFileName = "config.json"
)

// ErrNoAPIKey is returned when no key can be resolved from any source.
var ErrNoAPIKey = errors.New("no LeakRadar API key configured; run 'leakradar-cli auth set' or set " + envVar)

// fileConfig is the on-disk fallback used only when the OS keychain is
// unavailable. It intentionally has no other settings today.
type fileConfig struct {
	APIKey string `json:"api_key,omitempty"`
}

// Dir returns the per-OS config directory for the CLI, creating it if needed.
// os.UserConfigDir already resolves the right location per platform:
// ~/.config on Linux, ~/Library/Application Support on macOS, %AppData% on Windows.
func Dir() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("resolving config directory: %w", err)
	}
	dir := filepath.Join(base, appDirName)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", fmt.Errorf("creating config directory: %w", err)
	}
	return dir, nil
}

func filePath() (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, configFileName), nil
}

// ResolveAPIKey applies the priority order: explicit flag value (already
// handled by the caller before calling this), env var, OS keychain, config file.
func ResolveAPIKey() (string, error) {
	if v := os.Getenv(envVar); v != "" {
		return v, nil
	}
	if v, err := keyring.Get(keyringService, keyringUser); err == nil && v != "" {
		return v, nil
	}
	if fc, err := readFile(); err == nil && fc.APIKey != "" {
		return fc.APIKey, nil
	}
	return "", ErrNoAPIKey
}

// SetAPIKey stores the key in the OS keychain when available, falling back
// to a 0600 config file (e.g. headless Linux with no secret service running).
func SetAPIKey(key string) (storedInKeyring bool, err error) {
	if err := keyring.Set(keyringService, keyringUser, key); err == nil {
		return true, nil
	}
	// Keychain unavailable — fall back to the config file.
	if werr := writeFile(fileConfig{APIKey: key}); werr != nil {
		return false, fmt.Errorf("storing API key: keychain unavailable and config file write failed: %w", werr)
	}
	return false, nil
}

// ClearAPIKey removes the key from both the keychain and the config file.
func ClearAPIKey() error {
	_ = keyring.Delete(keyringService, keyringUser) // ignore "not found"
	path, err := filePath()
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("removing config file: %w", err)
	}
	return nil
}

// Source reports where the active API key came from, for `auth status`.
func Source() string {
	if os.Getenv(envVar) != "" {
		return envVar + " environment variable"
	}
	if v, err := keyring.Get(keyringService, keyringUser); err == nil && v != "" {
		return "OS keychain"
	}
	if fc, err := readFile(); err == nil && fc.APIKey != "" {
		p, _ := filePath()
		return "config file (" + p + ")"
	}
	return "none"
}

func readFile() (fileConfig, error) {
	var fc fileConfig
	path, err := filePath()
	if err != nil {
		return fc, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return fc, err
	}
	if err := json.Unmarshal(data, &fc); err != nil {
		return fc, fmt.Errorf("parsing config file %s: %w", path, err)
	}
	return fc, nil
}

func writeFile(fc fileConfig) error {
	path, err := filePath()
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(fc, "", "  ")
	if err != nil {
		return err
	}
	// 0600: only the owner can read/write. On Windows this mode is mostly a
	// no-op for ACLs, but MkdirAll(0700) on the parent still keeps other
	// local accounts without explicit access from being able to open it.
	return os.WriteFile(path, data, 0o600)
}

package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

// Config is a single registry credential set. It is stored as TOML at
// ~/.config/plusev/{label}.toml with mode 0600. Label is derived from the
// filename and is not written into the file body.
type Config struct {
	Label     string `toml:"-"`
	BaseURL   string `toml:"baseURL"`
	APIKey    string `toml:"apiKey"`
	APISecret string `toml:"apiSecret"`
}

// Dir returns the plusev config directory (~/.config/plusev on Linux).
func Dir() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("locate config dir: %w", err)
	}

	return filepath.Join(base, "plusev"), nil
}

// LoadAll reads every *.toml credential file in the config directory.
// Returns an empty map if the directory does not exist yet.
func LoadAll() (map[string]*Config, error) {
	dir, err := Dir()
	if err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]*Config{}, nil
		}

		return nil, fmt.Errorf("read config dir: %w", err)
	}

	out := map[string]*Config{}

	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".toml") {
			continue
		}

		path := filepath.Join(dir, e.Name())

		var cfg Config

		if _, err := toml.DecodeFile(path, &cfg); err != nil {
			return nil, fmt.Errorf("decode %s: %w", e.Name(), err)
		}

		cfg.Label = strings.TrimSuffix(e.Name(), ".toml")
		out[cfg.Label] = &cfg
	}

	return out, nil
}

// Save writes the config to {label}.toml with mode 0600, creating the config
// directory if needed. Returns the path written.
func Save(cfg *Config) (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}

	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", fmt.Errorf("create config dir: %w", err)
	}

	if cfg.Label == "" {
		return "", fmt.Errorf("config label is required")
	}

	path := filepath.Join(dir, cfg.Label+".toml")

	f, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0600)
	if err != nil {
		return "", fmt.Errorf("open config file: %w", err)
	}

	defer f.Close()

	if err := toml.NewEncoder(f).Encode(cfg); err != nil {
		return "", fmt.Errorf("encode config: %w", err)
	}

	return path, nil
}

// Delete removes the config file for a saved registry.
func Delete(label string) error {
	dir, err := Dir()
	if err != nil {
		return fmt.Errorf("locate config dir: %w", err)
	}

	path := filepath.Join(dir, label+".toml")

	if err := os.Remove(path); err != nil {
		return fmt.Errorf("delete %s: %w", path, err)
	}

	return nil
}

// HostFromURL derives a filesystem-safe label from a registry base URL. The
// scheme is stripped and remaining separators are replaced with dashes so two
// registries on the same host but different paths do not collide.
func HostFromURL(rawURL string) string {
	rawURL = strings.TrimSpace(rawURL)

	if i := strings.Index(rawURL, "://"); i >= 0 {
		rawURL = rawURL[i+3:]
	}

	rawURL = strings.ReplaceAll(rawURL, "/", "-")
	rawURL = strings.ReplaceAll(rawURL, ":", "-")
	rawURL = strings.Trim(rawURL, "-")

	return rawURL
}

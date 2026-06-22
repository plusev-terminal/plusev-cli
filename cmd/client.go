package cmd

import (
	"fmt"
	"strings"

	"github.com/plusev-terminal/plusev-cli/internal/api"
	"github.com/plusev-terminal/plusev-cli/internal/config"
	"github.com/plusev-terminal/plusev-cli/internal/prompt"
	"github.com/urfave/cli/v2"
)

// loadClient resolves registry credentials and returns an authenticated client.
func loadClient(c *cli.Context) (*api.Client, error) {
	cfg, err := resolveConfig(c)
	if err != nil {
		return nil, err
	}

	return api.New(cfg.BaseURL, cfg.APIKey, cfg.APISecret), nil
}

// resolveConfig picks the registry to use. Precedence:
//  1. --base-url + --api-key + --api-secret (fully headless / CI)
//  2. saved config file selected by --registry, auto when only one exists, or
//     via an interactive picker when several exist
//  3. PLUSEV_API_KEY / PLUSEV_SECRET always override credentials from a file
func resolveConfig(c *cli.Context) (*config.Config, error) {
	if baseURL := strings.TrimSpace(c.String("base-url")); baseURL != "" {
		baseURL = config.NormalizeBaseURL(baseURL)
		key := c.String("api-key")
		secret := c.String("api-secret")

		if key == "" || secret == "" {
			return nil, fmt.Errorf("--base-url requires --api-key and --api-secret (or PLUSEV_API_KEY/PLUSEV_SECRET)")
		}

		return &config.Config{BaseURL: baseURL, APIKey: key, APISecret: secret}, nil
	}

	all, err := config.LoadAll()
	if err != nil {
		return nil, err
	}

	var cfg *config.Config

	switch {
	case len(all) == 0:
		return nil, fmt.Errorf("no registry configured — run 'plusev-cli registry add --registry <url>' first")
	case len(all) == 1:
		for _, v := range all {
			cfg = v
		}
	default:
		label := c.String("registry")
		if label != "" {
			cfg = all[label]
		} else {
			opts := make([]prompt.Option, 0, len(all))
			for _, v := range all {
				opts = append(opts, prompt.Option{Value: v.Label, Label: fmt.Sprintf("%s  (%s)", v.Label, v.BaseURL)})
			}

			chosen, err := prompt.Select("Select a registry", opts)
			if err != nil {
				return nil, err
			}

			cfg = all[chosen.Value]
		}

		if cfg == nil {
			return nil, fmt.Errorf("no saved registry named %q", label)
		}
	}

	if key := c.String("api-key"); key != "" {
		cfg.APIKey = key
	}

	if secret := c.String("api-secret"); secret != "" {
		cfg.APISecret = secret
	}

	if cfg.BaseURL == "" {
		return nil, fmt.Errorf("registry %q has no base URL — re-run 'plusev-cli registry add'", cfg.Label)
	}

	if cfg.APIKey == "" || cfg.APISecret == "" {
		return nil, fmt.Errorf("registry %q has no credentials — re-run 'plusev-cli registry add'", cfg.Label)
	}

	return cfg, nil
}

// requireArg returns the first positional arg or an error naming what is missing.
func requireArg(c *cli.Context, name string) (string, error) {
	v := strings.TrimSpace(c.Args().First())
	if v == "" {
		return "", fmt.Errorf("missing required argument <%s>", name)
	}

	return v, nil
}

package cmd

import (
	"context"
	"fmt"

	"github.com/kataras/golog"
	"github.com/plusev-terminal/plusev-cli/internal/api"
	"github.com/plusev-terminal/plusev-cli/internal/config"
	"github.com/plusev-terminal/plusev-cli/internal/output"
	"github.com/plusev-terminal/plusev-cli/internal/prompt"
	"github.com/urfave/cli/v2"
)

func registryCommand() *cli.Command {
	return &cli.Command{
		Name:  "registry",
		Usage: "Manage plugin registries.",
		Subcommands: []*cli.Command{
			registryAddCommand(),
			registryListCommand(),
			registryDeleteCommand(),
			registryPruneCommand(),
		},
	}
}

func registryAddCommand() *cli.Command {
	return &cli.Command{
		Name:  "add",
		Usage: "Save credentials for a registry. Prompts interactively if --registry is not provided.",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "label",
				Usage: "Optional friendly name for this registry (defaults to the repo slug from /whoami)",
			},
		},
		Action: func(c *cli.Context) error {
			baseURL := c.String("registry")
			if baseURL == "" {
				var err error
				baseURL, err = prompt.RequiredText("Registry URL", "e.g. https://plusev-terminal.app")
				if err != nil {
					return err
				}
			}

			baseURL = config.NormalizeBaseURL(baseURL)

			apiKey, err := prompt.RequiredText("Dev key", "the dev key id issued by the repo owner")
			if err != nil {
				return err
			}

			apiSecret, err := prompt.Password("Dev key secret")
			if err != nil {
				return err
			}

			// Validate credentials and discover the repo.
			client := api.New(baseURL, apiKey, apiSecret)
			golog.Info("Validating credentials...")

			whoami, err := client.Whoami(c.Context)
			if err != nil {
				return fmt.Errorf("invalid credentials or key not associated with any repo: %w", err)
			}

			label := c.String("label")
			if label == "" {
				label = whoami.Slug
			}

			golog.Infof("Configuring registry %q (%s) as %q", whoami.Name, baseURL, label)

			cfg := &config.Config{
				Label:     label,
				BaseURL:   baseURL,
				APIKey:    apiKey,
				APISecret: apiSecret,
			}

			path, err := config.Save(cfg)
			if err != nil {
				return err
			}

			golog.Infof("Saved registry %q to %s", label, path)

			return nil
		},
	}
}

func registryListCommand() *cli.Command {
	return &cli.Command{
		Name:  "list",
		Usage: "List all saved registries.",
		Action: func(c *cli.Context) error {
			all, err := config.LoadAll()
			if err != nil {
				return err
			}

			if len(all) == 0 {
				output.Info("No registries configured. Use 'plusev-cli registry add' to add one.")
				return nil
			}

			rows := make([][]string, 0, len(all))

			for _, cfg := range all {
				rows = append(rows, []string{cfg.Label, cfg.BaseURL})
			}

			output.Table([]string{"LABEL", "URL"}, rows)

			return nil
		},
	}
}

func registryDeleteCommand() *cli.Command {
	return &cli.Command{
		Name:  "delete",
		Usage: "Delete a saved registry and optionally its remote plugins and releases.",
		Action: func(c *cli.Context) error {
			cfg, err := pickRegistry(c, "Select registry to delete")
			if err != nil {
				return err
			}

			ok, err := prompt.Confirm(fmt.Sprintf("Delete registry %q (%s)?", cfg.Label, cfg.BaseURL), false)
			if err != nil {
				return err
			}

			if !ok {
				output.Info("Cancelled.")
				return nil
			}

			cleanRemote, err := prompt.Confirm(fmt.Sprintf("Also remove all own plugins and their releases from %q?", cfg.Label), false)
			if err != nil {
				return err
			}

			if cleanRemote {
				if err := deleteAllPlugins(cfg); err != nil {
					return err
				}
			}

			if err := config.Delete(cfg.Label); err != nil {
				return fmt.Errorf("remove config file: %w", err)
			}

			output.Success(fmt.Sprintf("Deleted registry %q", cfg.Label))

			return nil
		},
	}
}

func registryPruneCommand() *cli.Command {
	return &cli.Command{
		Name:  "prune",
		Usage: "Purge all own plugins and their releases from a registry (keeps local credentials).",
		Action: func(c *cli.Context) error {
			cfg, err := pickRegistry(c, "Select registry to prune")
			if err != nil {
				return err
			}

			ok, err := prompt.Confirm(fmt.Sprintf("This will delete ALL your plugins and releases on %q. Continue?", cfg.Label), false)
			if err != nil {
				return err
			}

			if !ok {
				output.Info("Cancelled.")
				return nil
			}

			if err := deleteAllPlugins(cfg); err != nil {
				return err
			}

			output.Success(fmt.Sprintf("Pruned all plugins from %q", cfg.Label))

			return nil
		},
	}
}

// pickRegistry returns a saved Config. Uses --registry if set, otherwise
// shows an interactive picker. Returns an error if no registries exist.
func pickRegistry(c *cli.Context, title string) (*config.Config, error) {
	all, err := config.LoadAll()
	if err != nil {
		return nil, err
	}

	if len(all) == 0 {
		return nil, fmt.Errorf("no registries configured — run 'plusev-cli registry add --registry <url>' first")
	}

	label := c.String("registry")
	if label != "" {
		cfg, ok := all[label]
		if !ok {
			return nil, fmt.Errorf("no saved registry named %q", label)
		}

		return cfg, nil
	}

	if len(all) == 1 {
		for _, v := range all {
			return v, nil
		}
	}

	opts := make([]prompt.Option, 0, len(all))

	for _, v := range all {
		opts = append(opts, prompt.Option{Value: v.Label, Label: fmt.Sprintf("%s  (%s)", v.Label, v.BaseURL)})
	}

	chosen, err := prompt.Select(title, opts)
	if err != nil {
		return nil, err
	}

	return all[chosen.Value], nil
}

// deleteAllPlugins lists and deletes every plugin on the given registry.
func deleteAllPlugins(cfg *config.Config) error {
	client := api.New(cfg.BaseURL, cfg.APIKey, cfg.APISecret)

	plugins, err := client.ListPlugins(context.TODO())
	if err != nil {
		return fmt.Errorf("list plugins: %w", err)
	}

	if len(plugins) == 0 {
		output.Info("No plugins to remove.")

		return nil
	}

	for _, p := range plugins {
		if err := client.DeletePlugin(context.TODO(), p.PluginID); err != nil {
			return fmt.Errorf("delete plugin %q: %w", p.PluginID, err)
		}

		output.Info(fmt.Sprintf("Deleted plugin %q", p.PluginID))
	}

	return nil
}

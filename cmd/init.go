package cmd

import (
	"fmt"

	"github.com/kataras/golog"
	"github.com/plusev-terminal/plusev-cli/internal/config"
	"github.com/plusev-terminal/plusev-cli/internal/prompt"
	"github.com/urfave/cli/v2"
)

func initCommand() *cli.Command {
	return &cli.Command{
		Name:  "init",
		Usage: "Save credentials for a registry. Prompts for the dev key and secret.",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "label",
				Usage: "Optional friendly name for this registry (defaults to the host)",
			},
		},
		Action: func(c *cli.Context) error {
			baseURL := c.String("registry")
			if baseURL == "" {
				return fmt.Errorf("init requires --registry <url> (the registry base URL, e.g. https://host/extapi)")
			}

			label := c.String("label")
			if label == "" {
				label = config.HostFromURL(baseURL)
			}

			golog.Infof("Configuring registry %s as %q", baseURL, label)

			apiKey, err := prompt.RequiredText("Dev key", "the dev key id issued by the repo owner")
			if err != nil {
				return err
			}

			apiSecret, err := prompt.Password("Dev key secret")
			if err != nil {
				return err
			}

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

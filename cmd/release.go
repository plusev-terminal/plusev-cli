package cmd

import (
	"fmt"

	"github.com/plusev-terminal/plusev-cli/internal/api"
	"github.com/plusev-terminal/plusev-cli/internal/output"
	"github.com/plusev-terminal/plusev-cli/internal/prompt"
	"github.com/urfave/cli/v2"
	"golang.org/x/mod/semver"
)

func releaseCommand() *cli.Command {
	return &cli.Command{
		Name:  "release",
		Usage: "Manage plugin releases.",
		Subcommands: []*cli.Command{
			releasePublishCommand(),
			releaseDeleteCommand(),
			releasePruneCommand(),
		},
	}
}

func releasePublishCommand() *cli.Command {
	return &cli.Command{
		Name:  "publish",
		Usage: "Publish a release for an already-uploaded wasm binary. The plugin ID and version are read from the WASM by the server.",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "sha256", Required: true, Usage: "sha256 of a previously uploaded wasm (from 'plusev-cli publish' or files/upload)"},
			&cli.StringFlag{Name: "channel", Usage: "Release channel", Value: "stable"},
			&cli.StringFlag{Name: "changelog", Usage: "Release notes / changelog"},
		},
		Action: func(c *cli.Context) error {
			client, err := loadClient(c)
			if err != nil {
				return err
			}

			err = client.PublishRelease(c.Context, api.PublishRelease{
				Sha256:    c.String("sha256"),
				Channel:   c.String("channel"),
				Changelog: c.String("changelog"),
			})
			if err != nil {
				return err
			}

			output.Success("Release published")

			return nil
		},
	}
}

func releaseDeleteCommand() *cli.Command {
	return &cli.Command{
		Name:      "delete",
		Usage:     "Delete a release version (soft tombstone).",
		Args:      true,
		ArgsUsage: "<version>",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "plugin-id", Required: true, Usage: "The pluginId the release belongs to"},
		},
		Action: func(c *cli.Context) error {
			version, err := requireArg(c, "version")
			if err != nil {
				return err
			}

			ok, err := prompt.Confirm(fmt.Sprintf("Delete release %s of %q?", version, c.String("plugin-id")), false)
			if err != nil {
				return err
			}

			if !ok {
				output.Info("Cancelled.")
				return nil
			}

			client, err := loadClient(c)
			if err != nil {
				return err
			}

			if err := client.DeleteRelease(c.Context, c.String("plugin-id"), version); err != nil {
				return err
			}

			output.Success(fmt.Sprintf("Deleted release %s of %s", version, c.String("plugin-id")))

			return nil
		},
	}
}

func releasePruneCommand() *cli.Command {
	return &cli.Command{
		Name:      "prune",
		Usage:     "Delete all releases older than the given semver for a plugin.",
		Args:      true,
		ArgsUsage: "<olderThan>  (semver, e.g. 1.0.0)",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "plugin-id", Required: true, Usage: "The pluginId to prune"},
		},
		Action: func(c *cli.Context) error {
			olderThan, err := requireArg(c, "olderThan")
			if err != nil {
				return err
			}

			if !semver.IsValid("v" + olderThan) {
				return fmt.Errorf("<olderThan> must be valid semver, got %q", olderThan)
			}

			client, err := loadClient(c)
			if err != nil {
				return err
			}

			if err := client.PruneReleases(c.Context, c.String("plugin-id"), olderThan); err != nil {
				return err
			}

			output.Success(fmt.Sprintf("Pruned releases older than %s on %s", olderThan, c.String("plugin-id")))

			return nil
		},
	}
}

package cmd

import (
	"fmt"
	"strings"

	_ "github.com/plusev-terminal/plusev-cli/internal/api"
	"github.com/plusev-terminal/plusev-cli/internal/output"
	"github.com/plusev-terminal/plusev-cli/internal/prompt"
	"github.com/urfave/cli/v2"
)

func pluginCommand() *cli.Command {
	return &cli.Command{
		Name:  "plugin",
		Usage: "Manage plugin entries.",
		Subcommands: []*cli.Command{
			pluginListCommand(),
			pluginSetActiveCommand("deactivate", false),
			pluginSetActiveCommand("reactivate", true),
			pluginDeleteCommand(),
		},
	}
}

func pluginListCommand() *cli.Command {
	return &cli.Command{
		Name:  "list",
		Usage: "List your plugin entries.",
		Action: func(c *cli.Context) error {
			client, err := loadClient(c)
			if err != nil {
				return err
			}

			plugins, err := client.ListPlugins(c.Context)
			if err != nil {
				return err
			}

			if len(plugins) == 0 {
				output.Info("No plugins.")
				return nil
			}

			rows := make([][]string, 0, len(plugins))

			for _, p := range plugins {
				status := "active"
				if !p.Active {
					status = output.Dim("inactive")
				}

				rows = append(rows, []string{
					p.PluginID,
					p.Name,
					p.AppID,
					p.Version,
					status,
					output.JoinList(p.Tags),
				})
			}

			output.Table([]string{"PLUGIN ID", "NAME", "APP", "VERSION", "STATUS", "TAGS"}, rows)

			return nil
		},
	}
}

func pluginSetActiveCommand(name string, active bool) *cli.Command {
	usage := "Deactivate"
	if active {
		usage = "Reactivate"
	}

	return &cli.Command{
		Name:      name,
		Usage:     usage + " a plugin entry (sets Active).",
		ArgsUsage: "<pluginId>",
		Action: func(c *cli.Context) error {
			pluginID, err := requireArg(c, "pluginId")
			if err != nil {
				return err
			}

			client, err := loadClient(c)
			if err != nil {
				return err
			}

			if err := client.SetPluginActive(c.Context, pluginID, active); err != nil {
				return err
			}

			output.Success(fmt.Sprintf("%s %s", usage+"d", pluginID))

			return nil
		},
	}
}

func pluginDeleteCommand() *cli.Command {
	return &cli.Command{
		Name:      "delete",
		Usage:     "Delete a plugin entry (soft tombstone, restorable within the grace window).",
		ArgsUsage: "<pluginId>",
		Action: func(c *cli.Context) error {
			pluginID, err := requireArg(c, "pluginId")
			if err != nil {
				return err
			}

			ok, err := prompt.Confirm(fmt.Sprintf("Delete plugin %q? This starts the grace-window countdown.", pluginID), false)
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

			if err := client.DeletePlugin(c.Context, pluginID); err != nil {
				return err
			}

			output.Success(fmt.Sprintf("Scheduled deletion of %s", pluginID))

			return nil
		},
	}
}

// splitCSV parses a comma-separated flag value into a trimmed slice.
func splitCSV(v string) []string {
	if strings.TrimSpace(v) == "" {
		return nil
	}

	parts := strings.Split(v, ",")

	out := make([]string, 0, len(parts))

	for _, p := range parts {
		if t := strings.TrimSpace(p); t != "" {
			out = append(out, t)
		}
	}

	return out
}

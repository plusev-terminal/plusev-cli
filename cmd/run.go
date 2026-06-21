package cmd

import (
	"os"

	"github.com/kataras/golog"
	"github.com/urfave/cli/v2"
)

// Version is the plusev-cli version. Bumped on release.
const Version = "0.3.0"

// Run builds and runs the plusev-cli app. Mirrors terminal/cmd/run.go.
func Run() {
	app := &cli.App{
		Name:     "plusev-cli",
		Usage:    "Publish and manage PlusEV plugin_repo plugins",
		Version:  Version,
		Before:   debugBefore,
		Flags:    globalFlags(),
		Commands: commands(),
	}

	if err := app.Run(os.Args); err != nil {
		golog.Fatal(err)
	}
}

func debugBefore(c *cli.Context) error {
	if c.Bool("debug") {
		golog.Info("Debug mode is enabled")
		golog.SetLevel("debug")
	}

	return nil
}

func globalFlags() []cli.Flag {
	return []cli.Flag{
		&cli.BoolFlag{
			Name:    "debug",
			Aliases: []string{"d"},
			Usage:   "Enable debug logging (prints every HMAC-signed request and response body)",
			EnvVars: []string{"PLUSEV_CLI_DEBUG"},
		},
		// registry has two roles: for `init` it is the base URL of a new
		// registry to configure; for every other command it is the label or
		// host of a previously saved registry to use.
		&cli.StringFlag{
			Name:    "registry",
			Aliases: []string{"r"},
			Usage:   "Registry base URL (init) OR saved registry label/host to use",
			EnvVars: []string{"PLUSEV_REGISTRY"},
		},
		&cli.StringFlag{
			Name:    "base-url",
			Usage:   "Registry base URL (e.g. https://host/extapi). Skips config file lookup.",
			EnvVars: []string{"PLUSEV_BASE_URL"},
		},
		&cli.StringFlag{
			Name:    "api-key",
			Usage:   "Dev key id. Overrides the saved config and PLUSEV_API_KEY.",
			EnvVars: []string{"PLUSEV_API_KEY"},
		},
		&cli.StringFlag{
			Name:    "api-secret",
			Usage:   "Dev key secret. Overrides the saved config and PLUSEV_SECRET.",
			EnvVars: []string{"PLUSEV_SECRET"},
		},
	}
}

func commands() []*cli.Command {
	return []*cli.Command{
		initCommand(),
		publishCommand(),
		pluginCommand(),
		releaseCommand(),
		submissionCommand(),
		keyCommand(),
	}
}

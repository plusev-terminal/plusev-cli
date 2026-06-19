package cmd

import (
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/plusev-terminal/plusev-cli/internal/api"
	"github.com/plusev-terminal/plusev-cli/internal/output"
	"github.com/plusev-terminal/plusev-cli/internal/prompt"
	"github.com/urfave/cli/v2"
	"golang.org/x/mod/semver"
)

func publishCommand() *cli.Command {
	return &cli.Command{
		Name:  "publish",
		Usage: "Build the wasm, upload it, and publish a release in one step.",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "plugin-id",
				Usage:    "The pluginId of the entry to publish a release for",
				Required: true,
			},
			&cli.StringFlag{
				Name:     "version",
				Usage:    "Semver version for this release (e.g. 1.2.3)",
				Required: true,
			},
			&cli.StringFlag{
				Name:  "channel",
				Usage: "Release channel",
				Value: "stable",
			},
			&cli.StringFlag{
				Name:  "changelog",
				Usage: "Release notes / changelog",
			},
			&cli.StringFlag{
				Name:  "file",
				Usage: "Path to a prebuilt .wasm file. Skips the build step.",
			},
			&cli.StringFlag{
				Name:  "build-cmd",
				Usage: "Command used to build the wasm (run via sh)",
				Value: "./build.sh",
			},
			&cli.BoolFlag{
				Name:  "no-build",
				Usage: "Skip building; requires --file or an existing .wasm in the project",
			},
		},
		Action: func(c *cli.Context) error {
			version := c.String("version")
			if !semver.IsValid("v" + version) {
				return fmt.Errorf("--version must be a valid semver (e.g. 1.2.3), got %q", version)
			}

			client, err := loadClient(c)
			if err != nil {
				return err
			}

			ctx := c.Context

			wasmPath, err := resolveWasmFile(c)
			if err != nil {
				return err
			}

			content, err := os.ReadFile(wasmPath)
			if err != nil {
				return fmt.Errorf("read wasm file: %w", err)
			}

			output.Info(fmt.Sprintf("Uploading %s (%s)", filepath.Base(wasmPath), humanBytes(len(content))))

			upload, err := client.UploadFile(ctx, filepath.Base(wasmPath), content)
			if err != nil {
				return err
			}

			if upload.Exists {
				output.Info("File already known to the registry (deduplicated).")
			}

			output.KV("sha256", upload.Sha256)

			output.Info(fmt.Sprintf("Publishing release %s ...", version))

			err = client.PublishRelease(ctx, api.PublishRelease{
				PluginID:  c.String("plugin-id"),
				Version:   version,
				Sha256:    upload.Sha256,
				Channel:   c.String("channel"),
				Changelog: c.String("changelog"),
			})
			if err != nil {
				return err
			}

			output.Success(fmt.Sprintf("Published %s version %s", c.String("plugin-id"), version))

			return nil
		},
	}
}

// resolveWasmFile decides which .wasm to publish. If --file is set it is used
// directly. Otherwise the build command is run (unless --no-build) and a .wasm
// is located in the project; if several are found the user picks one.
func resolveWasmFile(c *cli.Context) (string, error) {
	if path := c.String("file"); path != "" {
		return path, nil
	}

	if !c.Bool("no-build") {
		cmdStr := c.String("build-cmd")

		output.Info(fmt.Sprintf("Building: %s", cmdStr))

		build := exec.CommandContext(c.Context, "sh", "-c", cmdStr)
		build.Stdout = os.Stdout
		build.Stderr = os.Stderr
		build.Stdin = os.Stdin

		if err := build.Run(); err != nil {
			return "", fmt.Errorf("build command failed: %w", err)
		}
	}

	matches := findWasmFiles(".")

	switch len(matches) {
	case 0:
		return "", fmt.Errorf("no .wasm file found — pass --file <path> or check your build output")
	case 1:
		return matches[0], nil
	default:
		opts := make([]prompt.Option, 0, len(matches))
		for _, m := range matches {
			opts = append(opts, prompt.Option{Value: m, Label: m})
		}

		chosen, err := prompt.Select("Multiple .wasm files found — pick one", opts)
		if err != nil {
			return "", err
		}

		return chosen.Value, nil
	}
}

// findWasmFiles returns wasm files in dir, skipping vendored/dependency trees.
func findWasmFiles(dir string) []string {
	var matches []string

	_ = filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}

		if d.IsDir() {
			base := d.Name()
			if base == "node_modules" || base == ".git" || base == "vendor" {
				return filepath.SkipDir
			}

			return nil
		}

		if strings.HasSuffix(strings.ToLower(path), ".wasm") {
			matches = append(matches, path)
		}

		return nil
	})

	sort.Strings(matches)

	return matches
}

func humanBytes(n int) string {
	const unit = 1024

	if n < unit {
		return fmt.Sprintf("%d B", n)
	}

	div, exp := int64(unit), 0

	for nn := int64(n); nn >= unit; nn /= unit {
		div *= unit
		exp++
	}

	return fmt.Sprintf("%.1f %ciB", float64(n)/float64(div/unit), "KMGTPE"[exp-1])
}

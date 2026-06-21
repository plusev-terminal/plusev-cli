package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	extism "github.com/extism/go-sdk"
	"github.com/plusev-terminal/plusev-cli/internal/api"
	"github.com/plusev-terminal/plusev-cli/internal/output"
	"github.com/plusev-terminal/plusev-cli/internal/prompt"
	"github.com/urfave/cli/v2"
)

// pluginMeta is the subset of the WASM meta export that publish needs.
type pluginMeta struct {
	PluginID string `json:"pluginId"`
	Name     string `json:"name"`
	Version  string `json:"version"`
}

func publishCommand() *cli.Command {
	return &cli.Command{
		Name:  "publish",
		Usage: "Build the wasm, read its metadata, upload it, and publish a release in one step. The plugin ID and version come from the plugin's meta export — they are not CLI flags.",
		Flags: []cli.Flag{
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
			client, err := loadClient(c)
			if err != nil {
				return err
			}

			ctx := c.Context

			wasmPath, err := resolveWasmFile(c)
			if err != nil {
				return err
			}

			meta, err := readPluginMeta(ctx, wasmPath)
			if err != nil {
				return fmt.Errorf("failed to read plugin metadata from %s: %w", wasmPath, err)
			}

			output.KV("plugin-id", meta.PluginID)
			output.KV("version", meta.Version)

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

			output.Info(fmt.Sprintf("Publishing release %s ...", meta.Version))

			err = client.PublishRelease(ctx, api.PublishRelease{
				Sha256:    upload.Sha256,
				Channel:   c.String("channel"),
				Changelog: c.String("changelog"),
			})
			if err != nil {
				return err
			}

			output.Success(fmt.Sprintf("Published %s version %s", meta.PluginID, meta.Version))

			return nil
		},
	}
}

// readPluginMeta instantiates the WASM binary in a throwaway Extism instance
// and calls the "meta" export to extract plugin identity fields.
//
// The meta function itself only uses pdk.OutputJSON (which lives in the
// extism:host/env module that the SDK provides), so we don't need real
// implementations of any extism:host/user functions. We provide empty stubs
// for every known host symbol from go-plugin-common so the WASM module
// imports resolve, even if the plugin's source references them transitively
// (e.g. via logging or requester packages that get dead-code-eliminated
// from the final binary in many cases — but not always).
func readPluginMeta(ctx context.Context, wasmPath string) (*pluginMeta, error) {
	manifest := extism.Manifest{
		Wasm: []extism.Wasm{
			extism.WasmFile{Path: wasmPath},
		},
	}

	compiled, err := extism.NewCompiledPlugin(ctx, manifest, extism.PluginConfig{
		EnableWasi: true,
	}, stubHostFunctions())
	if err != nil {
		return nil, fmt.Errorf("compile plugin: %w", err)
	}
	defer compiled.Close(ctx)

	inst, err := compiled.Instance(ctx, extism.PluginInstanceConfig{})
	if err != nil {
		return nil, fmt.Errorf("instantiate plugin: %w", err)
	}
	defer inst.Close(ctx)

	if !inst.FunctionExists("meta") {
		return nil, fmt.Errorf("plugin has no meta export")
	}

	_, output, err := inst.Call("meta", nil)
	if err != nil {
		return nil, fmt.Errorf("call meta: %w", err)
	}

	var meta pluginMeta
	if err := json.Unmarshal(output, &meta); err != nil {
		return nil, fmt.Errorf("parse meta JSON: %w", err)
	}

	if meta.PluginID == "" {
		return nil, fmt.Errorf("plugin meta is missing pluginId")
	}

	if meta.Version == "" {
		return nil, fmt.Errorf("plugin meta is missing version")
	}

	return &meta, nil
}

// stubHostFunctions returns no-op implementations of every extism:host/user
// symbol declared by go-plugin-common. The meta export never calls any of
// them, but the WASM module imports them at link time, so they have to
// resolve. None of these should be invoked in practice — if a plugin's
// meta somehow triggers one of these, the no-op is safe (it just returns 0).
func stubHostFunctions() []extism.HostFunction {
	noop := func(_ context.Context, _ *extism.CurrentPlugin, _ []uint64) {}

	names := []string{
		"log_record",
		"http_request",
		"time_now",
		"sleep_ms",
		"submit_trade",
		"submit_transfer",
		"get_records",
		"get_settings",
		"submit_report_entry",
		"report_progress",
	}

	out := make([]extism.HostFunction, 0, len(names))
	for _, name := range names {
		// All go-plugin-common host functions are (i64) -> (i64) per their
		// //go:wasmimport declarations. The no-op body returns zero and ignores
		// inputs — meta never actually calls any of these.
		out = append(out, extism.NewHostFunctionWithStack(name, noop,
			[]extism.ValueType{extism.ValueTypeI64},
			[]extism.ValueType{extism.ValueTypeI64}))
	}
	return out
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

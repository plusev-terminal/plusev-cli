# plusev-cli

Command line tool for publishing and managing plugins in a
[PlusEV Terminal](https://github.com/plusev-terminal) Plugin Registry.

It speaks the HMAC-authenticated `plugin_repo` write endpoints exposed by a
Terminal instance and lets a third-party developer ship plugins to a registry
they do **not** own — after the repo owner has issued them a dev key.

## Install

```bash
go install github.com/plusev-terminal/plusev-cli@latest
```

Or build from source:

```bash
git clone https://github.com/plusev-terminal/plusev-cli
cd plusev-cli
go build -o plusev-cli .
```

## Quick start

```bash
# 1. Configure a registry (prompts for registry url, the dev key id + secret the owner issued)
plusev-cli registry add

# 2. From your plugin project, build + upload + publish in one step
plusev-cli publish --plugin-id my-cool-plugin --version 1.0.0

# 3. Or manage pieces individually
plusev-cli plugin init --plugin-id my-cool-plugin --app-id datapipes --name "My Plugin"
plusev-cli plugin list
plusev-cli release publish --plugin-id my-cool-plugin --version 1.0.0 --sha256 <hash>
plusev-cli submission list
```

## Credentials

`registry add` stores one TOML file per registry at `~/.config/plusev/{label}.toml`
(mode `0600`):

```toml
baseURL = "https://terminal.example.com/extapi"
apiKey = "dev-key-id"
apiSecret = "dev-key-secret"
```

- The label defaults to the host (e.g. `terminal.example.com-extapi`); override
  with `--label`.
- When several registries are saved, `plusev-cli` asks which one to use, or you
  pass `--registry <label>` (`-r`).

### CI / non-interactive

Set env vars instead of a config file. This overrides everything:

```bash
export PLUSEV_API_KEY="dev-key-id"
export PLUSEV_SECRET="dev-key-secret"
plusev-cli publish --base-url https://terminal.example.com/extapi --plugin-id p --version 1.0.0
```

You can also mix: keep a saved config for the base URL and override only the
credentials via `PLUSEV_API_KEY` / `PLUSEV_SECRET`.

## Commands

```
plusev-cli registry add --registry <url> [--label <name>]      configure a registry
plusev-cli registry list                                        list saved registries
plusev-cli registry delete [--registry <label>]                 delete a registry
plusev-cli registry prune [--registry <label>]                  purge all plugins from a registry
plusev-cli publish                                      build + upload + publish a release
plusev-cli plugin init                                  create/update a plugin entry
plusev-cli plugin list                                  list your plugins
plusev-cli plugin deactivate <pluginId>                hide a plugin (Active=false)
plusev-cli plugin reactivate <pluginId>                unhide a plugin (Active=true)
plusev-cli plugin delete <pluginId>                    soft-delete (grace window)
plusev-cli release publish                             publish a release (sha256 of uploaded wasm)
plusev-cli release delete <version> --plugin-id <id>   delete a release
plusev-cli release prune <olderThan> --plugin-id <id>  delete releases older than a semver
plusev-cli submission list [--status --kind --plugin-id]
plusev-cli submission cancel <submissionId>
plusev-cli key email [new@email]                       request an email change for your dev key
```

`publish` runs the build command (default `./build.sh`, override with
`--build-cmd`) and publishes the resulting `.wasm`. Pass `--file path/to.wasm`
(or `--no-build` with an existing wasm) to skip the build step.

## Global flags

| Flag | Env | Purpose |
|------|-----|---------|
| `--debug`, `-d` | `PLUSEV_CLI_DEBUG` | Verbose logging |
| `--registry`, `-r` | `PLUSEV_REGISTRY` | Saved registry label/host (or, for `registry add`, the base URL to configure) |
| `--base-url` | `PLUSEV_BASE_URL` | Registry base URL; skips config file lookup |
| `--api-key` | `PLUSEV_API_KEY` | Dev key id (overrides config) |
| `--api-secret` | `PLUSEV_SECRET` | Dev key secret (overrides config) |

## How auth works

Every request is signed with HMAC-SHA256 over `timestamp + method + path + body`,
matching the Terminal `externalapi` middleware verification. The dev key's scope
(`plugin_repo.{slug}.write`) determines which repo it may publish to; the CLI
never types or sends the slug.

## Status

This CLI targets the write endpoints specified in
[PlusEV/terminal#186](https://gitea.codeblob.work/PlusEV/terminal/issues/186).
Those endpoints are implemented on the Terminal side; until they are, runtime
calls will return 404/auth errors.

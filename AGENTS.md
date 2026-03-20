# AGENTS.md — agent-plugins

Instructions for AI coding agents working in this repository.

## What this repo is

A collection of plugin package bundles for the `agent` framework. Each top-level directory is a self-contained plugin that can be installed into an `agent` instance or served by `agent-registry`.

This repo is **data**, not code. There is no Go module here. No build step. No tests to run.

## What this repo does NOT own

| Concern | Where it lives |
|---|---|
| Agent runtime and CLI | `../agent` |
| Registry server | `../agent-registry` |
| Documentation | `../agent-docs` |

## Structure of a plugin

Every plugin directory must have:

```
<plugin-name>/
  plugin.yaml           ← required — name, version, runtime type
  tools/                ← tool definitions (YAML)
  prompts/              ← prompt files (Markdown)
  policies/             ← policy definitions (YAML)
  profiles/             ← profile templates (YAML)
  examples/profiles/    ← ready-to-run example profiles
```

Only `plugin.yaml` is strictly required. Everything else is optional depending on what the plugin contributes.

## plugin.yaml minimum structure

```yaml
apiVersion: agent/v1
kind: Plugin
metadata:
  name: my-plugin          # must match directory name
  version: 0.1.0
  description: What this plugin does
spec:
  category: integration    # integration | bridge | orchestration | utility
  runtime:
    type: http             # http | mcp | command | host
  requires:
    framework: ">=0.1.0"
```

## Runtime types

| Type | When to use |
|---|---|
| `http` | Plugin runs as a separate HTTP service |
| `mcp` | Plugin bridges to an MCP server (stdio, HTTP, or SSE) |
| `command` | Plugin is a CLI binary or script (argv-template or JSON stdin/stdout) |
| `host` | Plugin runs inside the agent process (Go code, e.g. spawn-sub-agent) |

## How plugins are discovered

The `agent-registry` server scans this directory at startup. A directory is included if:
- It is not hidden (no `.` prefix)
- It is not named `registry`
- It contains a valid `plugin.yaml`

The `agent` CLI also scans this directory when configured as a filesystem plugin source.

## Naming conventions

- Directory name must match `metadata.name` in `plugin.yaml`
- Use lowercase and hyphens only: `my-plugin`, not `MyPlugin` or `my_plugin`
- Tool IDs use slash-namespacing: `email/send`, `ddg/search`, `agent/spawn`

## How to validate a plugin

From the `../agent` repo:

```bash
go run ./cmd/agent plugins validate ../agent-plugins/my-plugin
```

## How to test a plugin end-to-end

1. Start the registry:
   ```bash
   cd ../agent-registry
   go run ./cmd/registry-server --plugin-root ../agent-plugins --addr 127.0.0.1:9080
   ```
2. Install and run from the agent:
   ```bash
   cd ../agent
   go run ./cmd/agent plugins install my-plugin --source official
   go run ./cmd/agent run --profile ./_testing/profiles/coding/profile.yaml "test my-plugin"
   ```

## Things to watch out for

- Do not add a `go.mod` or any Go source to this repo — it's a data repo
- Keep plugin names globally unique across the collection
- `plugin.yaml` is parsed strictly — missing required fields cause the registry to skip the plugin
- Example profiles in `examples/profiles/` should be runnable without modification for demo purposes
- The `grafana-mcp` plugin uses `envMapping` to inject credentials — check `plugin.yaml` for the pattern
- The `ddg-research`, `grafana-alerts`, `slack`, and `json-tool` plugins require a compiled binary to be present at the path specified in `runtime.command`

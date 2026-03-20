# WHERE-WE-ARE

This file is a fast handoff for work in the `agent-plugins` directory.

## What this repo owns

- plugin package bundles
- plugin-owned example profiles
- package-side docs/examples for plugins

This repo does **not** own:

- the core runtime and CLI (`../agent`)
- the plugin registry server (`../agent-registry`)

## Current purpose

This directory is the package source set for the plugin ecosystem.

It currently acts as:

- local development source for `plugins install ../agent-plugins/<name> --link`
- local search source when `agent` is configured with a filesystem plugin source
- the package root that `agent-registry` scans to build the registry index

## Current layout

Packages currently here:

- `core-tools/`
- `github-cli/`
- `json-tool/`
- `mcp-filesystem/`
- `python-tool/`
- `send-email/`
- `spawn-sub-agent/`
- `web-research/`

Plugin-owned example profiles now live with packages, for example:

- `send-email/examples/profiles/`
- `web-research/examples/profiles/`
- `spawn-sub-agent/examples/profiles/`

## Relationship to the other repos

- `../agent` loads, validates, installs, enables, and runs these packages
- `../agent-registry` scans this directory and serves package index/metadata/artifacts

## What is true right now

- plugin bundles were intentionally moved out of `agent`
- this separation is part of the larger registry/package plan
- package bundles here are the future installable unit

## What this repo likely needs next

Not all of this has to happen immediately, but this is the expected direction:

1. add better package-level docs per plugin
2. decide whether packages need a `README.md` each
3. possibly add package metadata beyond `plugin.yaml` later if registry-specific fields become necessary
4. keep package names and versions aligned with registry expectations

## Current packaging assumptions

- package name should match `metadata.name` in `plugin.yaml`
- version comes from `metadata.version`
- package archive shape should be the package directory itself
- the registry server should be able to tarball the package as-is

## Important docs to read

- `README.md`
- `../agent/docs/architecture/plans/go-agent-plugin-package-model.md`
- `../agent-registry/docs/plugin-registry-contract.md`
- `../agent-registry/docs/plugin-registry-server-plan.md`
- `../agent-registry/docs/registry-server-build-guide.md`

## What should not be added here

- core CLI/runtime code
- registry server code
- framework-owned test profiles

Those belong in sibling repos.

## Quick validation commands

From `../agent`:

```bash
go run ./cmd/agent plugins sources add local-dev ../agent-plugins
go run ./cmd/agent plugins search
go run ./cmd/agent plugins install send-email --link
go run ./cmd/agent plugins validate ../agent-plugins/send-email
```

From `../agent-registry`:

```bash
go run ./cmd/registry-server --plugin-root ../agent-plugins --addr 127.0.0.1:9080
```

## Short summary

This directory is now the package source home for plugins.
Treat it as the package inventory that both the core CLI and the registry server depend on.

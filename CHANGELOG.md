# Changelog — agent-plugins

All notable changes to the official plugin package collection.

---

## Unreleased

---

## v0.2.0

### Added

- **`ddg-research`** — Real DuckDuckGo search via HTML scraping + page content fetch. Go binary runtime. Tools: `ddg/search`, `ddg/fetch`. Supports `df=` date filtering.
- **`grafana-mcp`** — MCP bridge to `mcp-grafana`. Covers dashboards, alerts, incidents, OnCall, Prometheus, and Loki. Uses `envMapping` for auth token injection.
- **`grafana-alerts`** — Direct Grafana REST API access via Go binary. Tools for alert events, PromQL, and LogQL queries. Tested end-to-end against real alert streams.
- **`kubectl`** — Kubernetes CLI integration via argv-template command runtime. Wraps `kubectl` for pods, deployments, events, logs, and namespaces.
- **`slack`** — Slack messaging via Incoming Webhooks or Web API. Go binary runtime. Supports plain text and Block Kit blocks.
- **`spawn-sub-agent`** — Added `agent/spawn-parallel` tool for concurrent sub-agent execution alongside the existing sequential `agent/spawn`.

### Changed

- `spawn-sub-agent` — expanded to support both sequential and parallel delegation patterns

---

## v0.1.0

Initial plugin set.

### Added

- **`core-tools`** — Host runtime bundle providing `read`, `write`, `edit`, `bash`, `glob`, `grep`
- **`github-cli`** — GitHub CLI wrapper via argv-template command runtime. Requires `gh` installed.
- **`json-tool`** — Example command plugin using JSON stdin/stdout mode with a custom Go binary
- **`mcp-filesystem`** — MCP bridge for the official `@modelcontextprotocol/server-filesystem` server
- **`python-tool`** — Example command plugin using a Python script with JSON stdin/stdout
- **`send-email`** — HTTP runtime plugin for email drafting and sending. Supports SMTP, Resend, SendGrid.
- **`spawn-sub-agent`** — Host runtime plugin for agent orchestration and sub-agent delegation
- **`web-research`** — HTTP runtime plugin for web search and page content fetching

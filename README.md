# agent-plugins

Official plugin packages for the [agent](https://github.com/bitop-dev/agent) framework.

Each directory is a self-contained plugin bundle with a `plugin.yaml` manifest, tools, prompts, policies, and example profiles. These packages are served by [agent-registry](https://github.com/bitop-dev/agent-registry) and installed by the `agent` CLI.

## Plugins

| Plugin | Runtime | Description |
|---|---|---|
| `core-tools` | host | Core local file and shell tools |
| `ddg-research` | command | Web search via DuckDuckGo + page content fetch |
| `github-cli` | command | GitHub CLI integration via argv-template |
| `grafana-alerts` | command | Alert events, PromQL, LogQL via Grafana REST API |
| `grafana-mcp` | mcp | Grafana MCP bridge — dashboards, alerts, Prometheus, Loki |
| `json-tool` | command | JSON stdin/stdout example plugin |
| `kubectl` | command | Kubernetes — pods, deployments, logs, events via kubectl |
| `mcp-filesystem` | mcp | Official filesystem MCP server bridge |
| `python-tool` | command | Python script with JSON stdin/stdout |
| `send-email` | http | Email drafting and sending |
| `slack` | command | Post messages via Slack Webhook or Web API |
| `spawn-sub-agent` | host | Orchestration plugin for spawning sub-agents |
| `web-research` | http | Web search and fetch |

## Using these plugins

```bash
# Via the registry (start agent-registry first)
agent plugins search email
agent plugins install send-email --source official

# Direct local install
agent plugins install ../agent-plugins/send-email --link
```

## Plugin structure

```
my-plugin/
  plugin.yaml           ← required: name, version, runtime
  tools/                ← tool definitions (.yaml)
  prompts/              ← prompt files (.md)
  policies/             ← policy definitions (.yaml)
  profiles/             ← profile templates
  examples/
    profiles/           ← example runnable profiles
  README.md             ← optional but recommended
```

## Adding a plugin

1. Create a directory with your plugin name (lowercase, hyphens only, e.g. `my-tool`)
2. Add `plugin.yaml` — see any existing plugin as a reference
3. Add tools, prompts, policies as needed
4. Restart the registry server — it picks up new plugins automatically

See the [building-plugins guide](https://github.com/bitop-dev/agent-docs/blob/main/core/building-plugins.md) for full details.

## Related repos

| Repo | Purpose |
|---|---|
| [agent](https://github.com/bitop-dev/agent) | Framework and CLI that installs and runs these plugins |
| [agent-registry](https://github.com/bitop-dev/agent-registry) | Registry server that serves these packages |
| [agent-docs](https://github.com/bitop-dev/agent-docs) | Full documentation |

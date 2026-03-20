# agent-plugins

Plugin package bundles for the agent framework.

**Full documentation:** https://github.com/bitop-dev/agent-docs/blob/main/plugins/overview.md

## Packages

| Plugin | Runtime | Category | Description |
|---|---|---|---|
| `core-tools` | host | integration | Core local file and shell tools |
| `github-cli` | command | integration | GitHub CLI integration |
| `json-tool` | command | integration | JSON stdin/stdout tool |
| `mcp-filesystem` | mcp | bridge | Filesystem MCP bridge |
| `python-tool` | command | integration | Python script runner |
| `send-email` | http | integration | Email drafting and sending |
| `spawn-sub-agent` | host | orchestration | Sub-agent delegation |
| `web-research` | http | integration | Web research and search |

## Adding a plugin

Create a directory with a `plugin.yaml`:

```yaml
apiVersion: agent/v1
kind: Plugin
metadata:
  name: my-plugin
  version: 0.1.0
  description: What this plugin does
spec:
  category: integration
  runtime:
    type: http        # http | mcp | command | host
  requires:
    framework: ">=0.1.0"
```

See [building-plugins](https://github.com/bitop-dev/agent-docs/blob/main/core/building-plugins.md) for the full guide.

## Related repos

| Repo | Purpose |
|---|---|
| [agent-docs](https://github.com/bitop-dev/agent-docs) | All documentation |
| [agent-registry](https://github.com/bitop-dev/agent-registry) | Serves these packages over HTTP |
| [agent](https://github.com/bitop-dev/agent) | Installs and runs these packages |

See [WHERE-WE-ARE.md](WHERE-WE-ARE.md) for current status.

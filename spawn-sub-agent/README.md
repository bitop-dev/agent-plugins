# spawn-sub-agent

Core orchestration plugin for multi-agent workflows. Discover available agents,
spawn sub-agents, run them in parallel, build pipelines, and manage agent memory.

## Tools

| Tool | Description |
|------|-------------|
| `agent/discover` | Find available agent profiles and their capabilities |
| `agent/spawn` | Spawn a sub-agent with a specific profile and task |
| `agent/spawn-parallel` | Run multiple sub-agents concurrently |
| `agent/pipeline` | Chain agents in sequence with variable routing |
| `agent/remember` | Store a fact in persistent agent memory |
| `agent/recall` | Retrieve facts from agent memory |

## Usage

This plugin is used by orchestrator profiles to delegate work to specialist agents.
It requires the `host` runtime (built into the agent framework).

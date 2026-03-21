# agent-plugins

Plugin packages for the [Agent Platform](https://github.com/bitop-dev/agent).

## Available plugins

| Plugin | Tools | Runtime | Description |
|---|---|---|---|
| **ddg-research** | ddg/search, ddg/fetch | Go binary | Web search via DuckDuckGo |
| **github** | github/repo, github/issues, github/pulls, github/pr-diff, github/search, github/file | Go binary | GitHub API integration |
| **http-request** | http/request | Go binary | Generic HTTP client |
| **csv-tool** | csv/parse, csv/query | Go binary | CSV/TSV data processing |
| **docker** | docker/ps, docker/logs, docker/inspect, docker/images | Go binary | Docker container operations |
| **kubectl** | k8s/get-pods, k8s/get-deployments, k8s/get-events, k8s/logs, k8s/get-namespaces, k8s/describe | command | Kubernetes CLI |
| **send-email** | email/draft, email/send | http | SMTP email |
| **slack** | slack/post-message, slack/post-blocks | command | Slack messaging |
| **spawn-sub-agent** | agent/discover, agent/spawn, agent/spawn-parallel, agent/pipeline, agent/remember, agent/recall | host | Multi-agent orchestration |

## Building Go plugins

```bash
cd github/cmd/github-tool
go build -o github-tool .
```

## Publishing

```bash
tar czf - github | curl -X POST http://registry:9080/v1/packages \
  -H "Authorization: Bearer <token>" --data-binary @-
```

# docker

Docker container operations for DevOps agents. List containers, view logs,
inspect configuration, and list images. Read-only — no start/stop/rm operations.

## Tools

| Tool | Description |
|------|-------------|
| `docker/ps` | List running containers |
| `docker/logs` | View container logs |
| `docker/inspect` | Detailed container/image info |
| `docker/images` | List images |

Requires `docker` CLI available on the host.

## Build

```bash
cd cmd/docker-tool && go build -o docker-tool .
```

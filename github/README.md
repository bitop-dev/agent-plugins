# github

GitHub integration via the GitHub REST API. Self-contained Go binary — no `gh`
CLI required. Works with or without a token (unauthenticated has lower rate limits).

## Tools

| Tool | Description |
|------|-------------|
| `github/repo` | Repository info (description, stars, forks, language) |
| `github/issues` | List issues with state, labels, assignees |
| `github/pulls` | List pull requests with review status |
| `github/pr-diff` | Get the full diff for a pull request |
| `github/search` | Search repos, code, issues, or users |
| `github/file` | Get file contents from a repository |

## Configuration

Set `GITHUB_TOKEN` environment variable for higher rate limits and private repo access:

```bash
agent plugins config set github token ghp_abc123
```

## Build

```bash
cd cmd/github-tool && go build -o github-tool .
```

# searxng

Web search and page fetching via [SearXNG](https://github.com/searxng/searxng)
metasearch engine. Aggregates results from Google, Bing, DuckDuckGo, Wikipedia,
and dozens of other sources — no API keys required.

## Tools

| Tool | Description |
|------|-------------|
| `web/search` | Search the web. Returns titles, URLs, snippets from multiple engines. |
| `web/fetch` | Fetch a URL and extract readable text content. |

## Configuration

Set the SearXNG instance URL (defaults to `http://searxng:8888` for k8s):

```bash
agent plugins config set searxng searxngURL http://localhost:8888
```

Or via environment: `SEARXNG_URL=http://searxng:8888`

## Search options

- **categories**: `general`, `news`, `science`, `it`, `images`
- **time_range**: `day`, `week`, `month`, `year` (empty = all time)
- **max_results**: limit number of results (default: 10)

## Build

```bash
cd cmd/searxng-tool && go build -o searxng-tool .
```

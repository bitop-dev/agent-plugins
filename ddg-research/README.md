# ddg-research

Web search and page content fetching via DuckDuckGo HTML search. No API key
required — works out of the box.

## Tools

| Tool | Description |
|------|-------------|
| `ddg/search` | Search the web via DuckDuckGo. Returns titles, URLs, and snippets. |
| `ddg/fetch` | Fetch and extract readable content from a URL. |

## Build

```bash
cd cmd/web-tool && go build -o web-tool .
```

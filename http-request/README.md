# http-request

Make HTTP requests to any API endpoint. Supports all methods, custom headers,
bearer token auth, and automatic JSON response parsing.

## Tool: `http/request`

```
method:       GET, POST, PUT, PATCH, DELETE
url:          https://api.example.com/data
headers:      {"X-Custom": "value"}
body:         '{"key": "value"}'
bearer_token: sk-abc123
timeout:      30
insecure:     false
```

## Examples

**GET a JSON API:**
```json
{"method": "GET", "url": "https://api.github.com/repos/golang/go"}
```

**POST with auth:**
```json
{
  "method": "POST",
  "url": "https://api.example.com/data",
  "bearer_token": "sk-abc123",
  "body": "{\"name\": \"test\"}"
}
```

## Build

```bash
cd cmd/http-tool && go build -o http-tool .
```

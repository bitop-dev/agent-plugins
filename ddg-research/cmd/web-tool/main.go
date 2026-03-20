// web-tool: JSON-stdin/stdout command plugin binary.
// Operations: search (DuckDuckGo HTML), fetch (page text extraction)
package main

import (
	"encoding/json"
	htmlpkg "html"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"
	"fmt"
)

// ─── Protocol types ────────────────────────────────────────────────────────

type request struct {
	Plugin    string         `json:"plugin"`
	Tool      string         `json:"tool"`
	Operation string         `json:"operation"`
	Arguments map[string]any `json:"arguments"`
	Config    map[string]any `json:"config"`
}

type response struct {
	Output string         `json:"output,omitempty"`
	Data   map[string]any `json:"data,omitempty"`
	Error  string         `json:"error,omitempty"`
}

// ─── HTTP client ───────────────────────────────────────────────────────────

var client = &http.Client{
	Timeout: 20 * time.Second,
	CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if len(via) >= 5 {
			return http.ErrUseLastResponse
		}
		return nil
	},
}

const userAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"

// ─── Entry point ───────────────────────────────────────────────────────────

func main() {
	raw, err := io.ReadAll(os.Stdin)
	if err != nil {
		writeError(fmt.Sprintf("read stdin: %v", err))
		return
	}
	var req request
	if err := json.Unmarshal(raw, &req); err != nil {
		writeError(fmt.Sprintf("parse request: %v", err))
		return
	}
	switch req.Operation {
	case "search":
		handleSearch(req)
	case "fetch":
		handleFetch(req)
	default:
		writeError(fmt.Sprintf("unknown operation: %q", req.Operation))
	}
}

// ─── Search ────────────────────────────────────────────────────────────────

type searchResult struct {
	Title   string `json:"title"`
	URL     string `json:"url"`
	Snippet string `json:"snippet"`
}

func handleSearch(req request) {
	query, _ := req.Arguments["query"].(string)
	if strings.TrimSpace(query) == "" {
		writeError("query is required")
		return
	}
	topK := 5
	if v, ok := req.Arguments["topK"].(float64); ok && v > 0 {
		topK = int(v)
	}
	timeRange, _ := req.Arguments["timeRange"].(string)

	results, err := ddgSearch(query, topK, timeRange)
	if err != nil {
		writeError(fmt.Sprintf("search failed: %v", err))
		return
	}

	lines := make([]string, 0, len(results))
	for i, r := range results {
		lines = append(lines, fmt.Sprintf("%d. %s\n   %s\n   %s", i+1, r.Title, r.URL, r.Snippet))
	}

	// Convert results to []any for JSON output
	data := make([]any, len(results))
	for i, r := range results {
		data[i] = map[string]string{
			"title":   r.Title,
			"url":     r.URL,
			"snippet": r.Snippet,
		}
	}

	writeResponse(response{
		Output: fmt.Sprintf("Search results for %q:\n\n%s", query, strings.Join(lines, "\n\n")),
		Data:   map[string]any{"query": query, "count": len(results), "results": data},
	})
}

// timeRangeParam maps friendly names to DDG df= values.
// DDG supports: d=day, w=week, m=month, y=year
func timeRangeParam(tr string) string {
	switch strings.ToLower(strings.TrimSpace(tr)) {
	case "day", "d":
		return "d"
	case "week", "w":
		return "w"
	case "month", "m":
		return "m"
	case "year", "y":
		return "y"
	default:
		return "" // no filter
	}
}

func ddgSearch(query string, topK int, timeRange string) ([]searchResult, error) {
	u := "https://html.duckduckgo.com/html/?q=" + url.QueryEscape(query)
	if df := timeRangeParam(timeRange); df != "" {
		u += "&df=" + df
	}
	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return parseDDGHTML(string(body), topK), nil
}

// DDG HTML patterns.
// Title link:   <a rel="nofollow" class="result__a" href="//duckduckgo.com/l/?uddg=ENCODED&...">TITLE</a>
// Snippet link: <a class="result__snippet" href="...">SNIPPET</a>
var (
	titleRe   = regexp.MustCompile(`(?i)<a[^>]+class="result__a"[^>]*href="([^"]+)"[^>]*>([\s\S]*?)</a>`)
	snippetRe = regexp.MustCompile(`(?i)<a[^>]+class="result__snippet"[^>]*>([\s\S]*?)</a>`)
)

func parseDDGHTML(body string, topK int) []searchResult {
	titleMatches := titleRe.FindAllStringSubmatch(body, -1)
	snippetMatches := snippetRe.FindAllStringSubmatch(body, -1)

	var results []searchResult
	for i, tm := range titleMatches {
		if len(results) >= topK {
			break
		}
		rawHref := tm[1]
		title := cleanText(tm[2])
		if title == "" {
			continue
		}
		actualURL := extractDDGURL(rawHref)
		if actualURL == "" || strings.Contains(actualURL, "duckduckgo.com/y.js") {
			continue // skip ads
		}
		snippet := ""
		if i < len(snippetMatches) {
			snippet = cleanText(snippetMatches[i][1])
		}
		results = append(results, searchResult{
			Title:   title,
			URL:     actualURL,
			Snippet: snippet,
		})
	}
	return results
}

// extractDDGURL decodes the actual destination URL from a DDG redirect href.
// DDG hrefs look like: //duckduckgo.com/l/?uddg=https%3A%2F%2F...&rut=...
func extractDDGURL(href string) string {
	// Parse as URL — prepend https: if scheme-relative
	if strings.HasPrefix(href, "//") {
		href = "https:" + href
	}
	parsed, err := url.Parse(href)
	if err != nil {
		return href
	}
	uddg := parsed.Query().Get("uddg")
	if uddg != "" {
		decoded, err := url.QueryUnescape(uddg)
		if err == nil {
			return decoded
		}
	}
	return href
}

// ─── Fetch ─────────────────────────────────────────────────────────────────

const maxFetchBytes = 5000

func handleFetch(req request) {
	rawURL, _ := req.Arguments["url"].(string)
	if strings.TrimSpace(rawURL) == "" {
		writeError("url is required")
		return
	}

	text, err := fetchPage(rawURL)
	if err != nil {
		writeError(fmt.Sprintf("fetch %s: %v", rawURL, err))
		return
	}

	writeResponse(response{
		Output: text,
		Data:   map[string]any{"url": rawURL, "length": len(text)},
	})
}

func fetchPage(rawURL string) (string, error) {
	req, err := http.NewRequest("GET", rawURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,text/plain;q=0.9")

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	limited := io.LimitReader(resp.Body, 512*1024) // 512KB cap
	body, err := io.ReadAll(limited)
	if err != nil {
		return "", err
	}

	text := extractText(string(body))
	if len(text) > maxFetchBytes {
		text = text[:maxFetchBytes] + fmt.Sprintf("\n\n[truncated — %d chars total]", len(text))
	}
	return text, nil
}

// ─── HTML text extraction ──────────────────────────────────────────────────

var (
	scriptRe  = regexp.MustCompile(`(?is)<script[\s\S]*?</script>`)
	styleRe   = regexp.MustCompile(`(?is)<style[\s\S]*?</style>`)
	tagRe     = regexp.MustCompile(`<[^>]+>`)
	spaceRe   = regexp.MustCompile(`[ \t]+`)
	newlineRe = regexp.MustCompile(`\n{3,}`)
)

func extractText(h string) string {
	h = scriptRe.ReplaceAllString(h, " ")
	h = styleRe.ReplaceAllString(h, " ")
	h = tagRe.ReplaceAllString(h, " ")
	h = htmlpkg.UnescapeString(h)
	h = spaceRe.ReplaceAllString(h, " ")
	lines := strings.Split(h, "\n")
	kept := lines[:0]
	for _, l := range lines {
		if strings.TrimSpace(l) != "" {
			kept = append(kept, strings.TrimSpace(l))
		}
	}
	h = strings.Join(kept, "\n")
	h = newlineRe.ReplaceAllString(h, "\n\n")
	return strings.TrimSpace(h)
}

// cleanText strips inline HTML tags and decodes entities from short strings.
func cleanText(s string) string {
	s = tagRe.ReplaceAllString(s, "")
	s = htmlpkg.UnescapeString(s)
	return strings.TrimSpace(s)
}

// ─── Output helpers ────────────────────────────────────────────────────────

func writeResponse(r response) {
	json.NewEncoder(os.Stdout).Encode(r)
}

func writeError(msg string) {
	json.NewEncoder(os.Stdout).Encode(response{Error: msg})
}

package main

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

type request struct {
	Tool      string         `json:"tool"`
	Arguments map[string]any `json:"arguments"`
}

type response struct {
	Status  string         `json:"status"`
	Content map[string]any `json:"content"`
}

type searxResult struct {
	Title   string `json:"title"`
	URL     string `json:"url"`
	Content string `json:"content"`
	Engine  string `json:"engine"`
}

type searxResponse struct {
	Results []searxResult `json:"results"`
	Query   string        `json:"query"`
}

var baseURL string

func main() {
	baseURL = os.Getenv("SEARXNG_URL")
	if baseURL == "" {
		baseURL = "http://searxng:8888"
	}

	var req request
	if err := json.NewDecoder(os.Stdin).Decode(&req); err != nil {
		writeError(fmt.Sprintf("invalid input: %v", err))
		return
	}

	switch req.Tool {
	case "web/search":
		handleSearch(req.Arguments)
	case "web/fetch":
		handleFetch(req.Arguments)
	default:
		writeError(fmt.Sprintf("unknown tool: %s", req.Tool))
	}
}

func handleSearch(args map[string]any) {
	query := getString(args, "query", "")
	if query == "" {
		writeError("query is required")
		return
	}

	categories := getString(args, "categories", "general")
	maxResults := getInt(args, "max_results", 10)
	timeRange := getString(args, "time_range", "")

	params := url.Values{
		"q":          {query},
		"format":     {"json"},
		"categories": {categories},
	}
	if timeRange != "" {
		params.Set("time_range", timeRange)
	}

	searchURL := strings.TrimRight(baseURL, "/") + "/search?" + params.Encode()
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Get(searchURL)
	if err != nil {
		writeError(fmt.Sprintf("search request failed: %v", err))
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		writeError(fmt.Sprintf("read response: %v", err))
		return
	}

	if resp.StatusCode != 200 {
		writeError(fmt.Sprintf("searxng returned %d: %s", resp.StatusCode, string(body)[:200]))
		return
	}

	var sr searxResponse
	if err := json.Unmarshal(body, &sr); err != nil {
		writeError(fmt.Sprintf("parse response: %v", err))
		return
	}

	// Limit results
	results := sr.Results
	if len(results) > maxResults {
		results = results[:maxResults]
	}

	// Format output
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Search results for %q:\n\n", query))
	for i, r := range results {
		sb.WriteString(fmt.Sprintf("%d. %s\n   %s\n", i+1, r.Title, r.URL))
		if r.Content != "" {
			sb.WriteString(fmt.Sprintf("   %s\n", r.Content))
		}
		sb.WriteString("\n")
	}

	// Structured data
	var resultData []map[string]string
	for _, r := range results {
		resultData = append(resultData, map[string]string{
			"title":   r.Title,
			"url":     r.URL,
			"snippet": r.Content,
			"engine":  r.Engine,
		})
	}

	writeResult(map[string]any{
		"results": sb.String(),
		"data":    resultData,
		"count":   len(results),
		"query":   query,
	})
}

func handleFetch(args map[string]any) {
	rawURL := getString(args, "url", "")
	if rawURL == "" {
		writeError("url is required")
		return
	}

	client := &http.Client{
		Timeout: 15 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}

	req, err := http.NewRequest("GET", rawURL, nil)
	if err != nil {
		writeError(fmt.Sprintf("invalid url: %v", err))
		return
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; AgentPlatform/1.0)")

	resp, err := client.Do(req)
	if err != nil {
		writeError(fmt.Sprintf("fetch failed: %v", err))
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 500*1024)) // 500KB limit
	if err != nil {
		writeError(fmt.Sprintf("read body: %v", err))
		return
	}

	// Basic HTML text extraction
	text := extractText(string(body))
	if len(text) > 50000 {
		text = text[:50000] + "\n... (truncated)"
	}

	writeResult(map[string]any{
		"url":         rawURL,
		"status_code": resp.StatusCode,
		"content":     text,
		"length":      len(text),
	})
}

func extractText(html string) string {
	// Remove script and style tags
	for _, tag := range []string{"script", "style", "noscript"} {
		for {
			start := strings.Index(strings.ToLower(html), "<"+tag)
			if start == -1 {
				break
			}
			end := strings.Index(strings.ToLower(html[start:]), "</"+tag+">")
			if end == -1 {
				break
			}
			html = html[:start] + html[start+end+len("</"+tag+">"):]
		}
	}

	// Remove HTML tags
	var result strings.Builder
	inTag := false
	lastWasSpace := false
	for _, c := range html {
		if c == '<' {
			inTag = true
			continue
		}
		if c == '>' {
			inTag = false
			if !lastWasSpace {
				result.WriteRune(' ')
				lastWasSpace = true
			}
			continue
		}
		if !inTag {
			if c == '\n' || c == '\r' || c == '\t' {
				if !lastWasSpace {
					result.WriteRune(' ')
					lastWasSpace = true
				}
			} else {
				result.WriteRune(c)
				lastWasSpace = c == ' '
			}
		}
	}

	// Collapse whitespace
	text := result.String()
	for strings.Contains(text, "  ") {
		text = strings.ReplaceAll(text, "  ", " ")
	}
	return strings.TrimSpace(text)
}

func getString(m map[string]any, key, def string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return def
}

func getInt(m map[string]any, key string, def int) int {
	if v, ok := m[key]; ok {
		if n, ok := v.(float64); ok {
			return int(n)
		}
	}
	return def
}

func writeResult(content map[string]any) {
	json.NewEncoder(os.Stdout).Encode(response{Status: "ok", Content: content})
}

func writeError(msg string) {
	json.NewEncoder(os.Stdout).Encode(response{Status: "error", Content: map[string]any{"error": msg}})
}

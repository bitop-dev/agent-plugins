package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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

var apiBase = "https://api.github.com"
var token string

func main() {
	token = os.Getenv("GITHUB_TOKEN")

	var req request
	if err := json.NewDecoder(os.Stdin).Decode(&req); err != nil {
		writeError(fmt.Sprintf("invalid input: %v", err))
		return
	}

	switch req.Tool {
	case "github/repo":
		githubRepo(req.Arguments)
	case "github/issues":
		githubIssues(req.Arguments)
	case "github/pulls":
		githubPulls(req.Arguments)
	case "github/pr-diff":
		githubPRDiff(req.Arguments)
	case "github/search":
		githubSearch(req.Arguments)
	case "github/file":
		githubFile(req.Arguments)
	default:
		writeError(fmt.Sprintf("unknown tool: %s", req.Tool))
	}
}

func githubRepo(args map[string]any) {
	repo := getString(args, "repo", "")
	if repo == "" {
		writeError("repo is required (e.g. 'golang/go')")
		return
	}
	data, err := ghGet("/repos/" + repo)
	if err != nil {
		writeError(err.Error())
		return
	}
	writeResult(data)
}

func githubIssues(args map[string]any) {
	repo := getString(args, "repo", "")
	if repo == "" {
		writeError("repo is required")
		return
	}
	state := getString(args, "state", "open")
	limit := getInt(args, "limit", 10)
	data, err := ghGet(fmt.Sprintf("/repos/%s/issues?state=%s&per_page=%d", repo, state, limit))
	if err != nil {
		writeError(err.Error())
		return
	}
	writeResult(map[string]any{"issues": data})
}

func githubPulls(args map[string]any) {
	repo := getString(args, "repo", "")
	if repo == "" {
		writeError("repo is required")
		return
	}
	state := getString(args, "state", "open")
	limit := getInt(args, "limit", 10)
	data, err := ghGet(fmt.Sprintf("/repos/%s/pulls?state=%s&per_page=%d", repo, state, limit))
	if err != nil {
		writeError(err.Error())
		return
	}
	writeResult(map[string]any{"pulls": data})
}

func githubPRDiff(args map[string]any) {
	repo := getString(args, "repo", "")
	number := getInt(args, "number", 0)
	if repo == "" || number == 0 {
		writeError("repo and number are required")
		return
	}

	url := fmt.Sprintf("%s/repos/%s/pulls/%d", apiBase, repo, number)
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Accept", "application/vnd.github.diff")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		writeError(err.Error())
		return
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 500000))

	diff := string(body)
	if len(diff) > 100000 {
		diff = diff[:100000] + "\n... (truncated)"
	}
	writeResult(map[string]any{"diff": diff, "pr": number, "repo": repo})
}

func githubSearch(args map[string]any) {
	query := getString(args, "query", "")
	searchType := getString(args, "type", "repositories")
	limit := getInt(args, "limit", 10)
	if query == "" {
		writeError("query is required")
		return
	}
	data, err := ghGet(fmt.Sprintf("/search/%s?q=%s&per_page=%d",
		searchType, strings.ReplaceAll(query, " ", "+"), limit))
	if err != nil {
		writeError(err.Error())
		return
	}
	writeResult(data)
}

func githubFile(args map[string]any) {
	repo := getString(args, "repo", "")
	path := getString(args, "path", "")
	ref := getString(args, "ref", "")
	if repo == "" || path == "" {
		writeError("repo and path are required")
		return
	}
	endpoint := fmt.Sprintf("/repos/%s/contents/%s", repo, path)
	if ref != "" {
		endpoint += "?ref=" + ref
	}
	data, err := ghGet(endpoint)
	if err != nil {
		writeError(err.Error())
		return
	}
	writeResult(data)
}

func ghGet(endpoint string) (map[string]any, error) {
	url := apiBase + endpoint
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("GitHub API %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var result any
	if err := json.Unmarshal(body, &result); err != nil {
		return map[string]any{"raw": string(body)}, nil
	}
	switch v := result.(type) {
	case map[string]any:
		return v, nil
	default:
		return map[string]any{"data": v}, nil
	}
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

func getBool(m map[string]any, key string, def bool) bool {
	if v, ok := m[key]; ok {
		if b, ok := v.(bool); ok {
			return b
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

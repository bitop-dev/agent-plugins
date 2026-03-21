package main

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

type request struct {
	Tool      string            `json:"tool"`
	Arguments map[string]any    `json:"arguments"`
}

type response struct {
	Status  string         `json:"status"`
	Content map[string]any `json:"content"`
}

func main() {
	var req request
	if err := json.NewDecoder(os.Stdin).Decode(&req); err != nil {
		writeError(fmt.Sprintf("invalid input: %v", err))
		return
	}

	switch req.Tool {
	case "http/request":
		doRequest(req.Arguments)
	default:
		writeError(fmt.Sprintf("unknown tool: %s", req.Tool))
	}
}

func doRequest(args map[string]any) {
	method := strings.ToUpper(getString(args, "method", "GET"))
	url := getString(args, "url", "")
	if url == "" {
		writeError("url is required")
		return
	}

	body := getString(args, "body", "")
	headersRaw, _ := args["headers"].(map[string]any)
	timeoutSec := getInt(args, "timeout", 30)
	insecure := getBool(args, "insecure", false)

	var bodyReader io.Reader
	if body != "" {
		bodyReader = strings.NewReader(body)
	}

	httpReq, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		writeError(fmt.Sprintf("failed to create request: %v", err))
		return
	}

	// Set headers
	for k, v := range headersRaw {
		httpReq.Header.Set(k, fmt.Sprintf("%v", v))
	}
	if httpReq.Header.Get("Content-Type") == "" && body != "" {
		httpReq.Header.Set("Content-Type", "application/json")
	}

	// Auth from args
	bearerToken := getString(args, "bearer_token", "")
	if bearerToken != "" {
		httpReq.Header.Set("Authorization", "Bearer "+bearerToken)
	}

	client := &http.Client{Timeout: time.Duration(timeoutSec) * time.Second}
	if insecure {
		client.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
	}

	resp, err := client.Do(httpReq)
	if err != nil {
		writeError(fmt.Sprintf("request failed: %v", err))
		return
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20)) // 1MB limit
	if err != nil {
		writeError(fmt.Sprintf("failed to read response: %v", err))
		return
	}

	// Try to parse as JSON
	var jsonBody any
	if err := json.Unmarshal(respBody, &jsonBody); err == nil {
		writeResult(map[string]any{
			"status_code": resp.StatusCode,
			"status":      resp.Status,
			"headers":     flattenHeaders(resp.Header),
			"body":        jsonBody,
		})
	} else {
		// Return as text
		bodyStr := string(respBody)
		if len(bodyStr) > 50000 {
			bodyStr = bodyStr[:50000] + "\n... (truncated)"
		}
		writeResult(map[string]any{
			"status_code": resp.StatusCode,
			"status":      resp.Status,
			"headers":     flattenHeaders(resp.Header),
			"body":        bodyStr,
		})
	}
}

func flattenHeaders(h http.Header) map[string]string {
	out := make(map[string]string)
	for k, v := range h {
		out[k] = strings.Join(v, ", ")
	}
	return out
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
		switch n := v.(type) {
		case float64:
			return int(n)
		case int:
			return n
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

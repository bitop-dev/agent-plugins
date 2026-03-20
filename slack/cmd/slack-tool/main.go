// slack-tool: JSON-stdin/stdout command plugin for posting to Slack via Incoming Webhooks.
//
// Operations:
//   post-message  — post a message to a channel or webhook
//   post-blocks   — post a message with Block Kit blocks
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// ── Protocol types ────────────────────────────────────────────────────────────

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

var httpClient = &http.Client{Timeout: 15 * time.Second}

// ── Entry point ───────────────────────────────────────────────────────────────

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
	case "post-message":
		handlePostMessage(req)
	case "post-blocks":
		handlePostBlocks(req)
	default:
		writeError(fmt.Sprintf("unknown operation: %q", req.Operation))
	}
}

// ── post-message ──────────────────────────────────────────────────────────────

func handlePostMessage(req request) {
	webhookURL := configString(req.Config, "webhookURL")
	token := configString(req.Config, "token")
	channel, _ := req.Arguments["channel"].(string)
	text, _ := req.Arguments["text"].(string)
	username, _ := req.Arguments["username"].(string)
	iconEmoji, _ := req.Arguments["icon_emoji"].(string)

	if strings.TrimSpace(text) == "" {
		writeError("text is required")
		return
	}

	payload := map[string]any{"text": text}
	if channel != "" {
		payload["channel"] = channel
	}
	if username != "" {
		payload["username"] = username
	}
	if iconEmoji != "" {
		payload["icon_emoji"] = iconEmoji
	}

	if err := postToSlack(webhookURL, token, payload); err != nil {
		writeError(fmt.Sprintf("post message: %v", err))
		return
	}
	writeResponse(response{
		Output: fmt.Sprintf("message posted to Slack%s", channelSuffix(channel)),
		Data:   map[string]any{"channel": channel, "text": text},
	})
}

// ── post-blocks ───────────────────────────────────────────────────────────────

func handlePostBlocks(req request) {
	webhookURL := configString(req.Config, "webhookURL")
	token := configString(req.Config, "token")
	channel, _ := req.Arguments["channel"].(string)
	text, _ := req.Arguments["text"].(string)   // fallback text
	blocksRaw := req.Arguments["blocks"]         // []any of Block Kit blocks

	if blocksRaw == nil {
		writeError("blocks is required")
		return
	}

	payload := map[string]any{"blocks": blocksRaw}
	if text != "" {
		payload["text"] = text
	}
	if channel != "" {
		payload["channel"] = channel
	}

	if err := postToSlack(webhookURL, token, payload); err != nil {
		writeError(fmt.Sprintf("post blocks: %v", err))
		return
	}
	writeResponse(response{
		Output: fmt.Sprintf("block message posted to Slack%s", channelSuffix(channel)),
		Data:   map[string]any{"channel": channel},
	})
}

// ── Slack HTTP helpers ────────────────────────────────────────────────────────

func postToSlack(webhookURL, token string, payload map[string]any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	if strings.TrimSpace(webhookURL) == "" && strings.TrimSpace(token) == "" {
		return fmt.Errorf("either webhookURL or token must be configured")
	}

	// Prefer webhook URL — simpler and doesn't require channel permission.
	if strings.TrimSpace(webhookURL) != "" {
		return postWebhook(webhookURL, body)
	}
	return postAPI(token, body)
}

func postWebhook(url string, body []byte) error {
	resp, err := httpClient.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return fmt.Errorf("webhook returned %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}
	// Webhooks return "ok" on success
	if msg := strings.TrimSpace(string(respBody)); msg != "ok" && msg != "" {
		return fmt.Errorf("webhook error: %s", msg)
	}
	return nil
}

func postAPI(token string, body []byte) error {
	req, err := http.NewRequest("POST", "https://slack.com/api/chat.postMessage", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	var result struct {
		OK    bool   `json:"ok"`
		Error string `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	if !result.OK {
		return fmt.Errorf("slack API error: %s", result.Error)
	}
	return nil
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func configString(cfg map[string]any, key string) string {
	v, _ := cfg[key].(string)
	return v
}

func channelSuffix(channel string) string {
	if channel == "" {
		return ""
	}
	return " (channel: " + channel + ")"
}

func writeResponse(r response) {
	json.NewEncoder(os.Stdout).Encode(r)
}

func writeError(msg string) {
	json.NewEncoder(os.Stdout).Encode(response{Error: msg})
}

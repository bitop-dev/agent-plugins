// grafana-tool: JSON-stdin/stdout command plugin for querying Grafana alert events,
// Prometheus metrics, and Loki logs directly via the Grafana REST API.
//
// Operations:
//   alert-events   — fetch alert state changes for a team over a time range
//   query-metrics  — run PromQL against a Mimir/Prometheus datasource
//   query-logs     — run LogQL against a Loki datasource
//   datasources    — list datasources (optionally filtered by type)
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strconv"
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

// ── Grafana HTTP client ───────────────────────────────────────────────────────

type client struct {
	baseURL string
	apiKey  string
	http    *http.Client
}

func newClient(baseURL, apiKey string) *client {
	return &client{
		baseURL: strings.TrimRight(baseURL, "/"),
		apiKey:  apiKey,
		http:    &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *client) get(path string, params url.Values) ([]byte, error) {
	u := c.baseURL + path
	if len(params) > 0 {
		u += "?" + params.Encode()
	}
	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Accept", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("GET %s: %w", path, err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("GET %s: HTTP %d: %s", path, resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return body, nil
}

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

	grafanaURL, _ := req.Config["grafanaURL"].(string)
	grafanaAPIKey, _ := req.Config["grafanaAPIKey"].(string)
	if strings.TrimSpace(grafanaURL) == "" {
		writeError("config.grafanaURL is required")
		return
	}
	if strings.TrimSpace(grafanaAPIKey) == "" {
		writeError("config.grafanaAPIKey is required")
		return
	}

	c := newClient(grafanaURL, grafanaAPIKey)

	switch req.Operation {
	case "alert-events":
		handleAlertEvents(c, req.Arguments)
	case "query-metrics":
		handleQueryMetrics(c, req.Arguments)
	case "query-logs":
		handleQueryLogs(c, req.Arguments)
	case "datasources":
		handleDatasources(c, req.Arguments)
	default:
		writeError(fmt.Sprintf("unknown operation: %q", req.Operation))
	}
}

// ── alert-events ──────────────────────────────────────────────────────────────
//
// Combines two sources:
//  1. Alertmanager API — currently active, silenced, and inhibited alerts
//  2. Prometheus rules API — rule instances with activeAt timestamps, for rules
//     whose labels match the requested team

type alertEvent struct {
	Name       string            `json:"name"`
	State      string            `json:"state"`
	ActiveAt   string            `json:"activeAt"`
	EndsAt     string            `json:"endsAt,omitempty"`
	Labels     map[string]string `json:"labels"`
	Source     string            `json:"source"` // "alertmanager" or "rules"
}

func handleAlertEvents(c *client, args map[string]any) {
	team, _ := args["team"].(string)
	fromStr, _ := args["from"].(string)
	toStr, _ := args["to"].(string)

	if team == "" {
		writeError("team is required")
		return
	}

	var from, to time.Time
	var err error
	if fromStr != "" {
		from, err = time.Parse(time.RFC3339, fromStr)
		if err != nil {
			writeError(fmt.Sprintf("invalid from: %v", err))
			return
		}
	} else {
		from = time.Now().Add(-7 * 24 * time.Hour)
	}
	if toStr != "" {
		to, err = time.Parse(time.RFC3339, toStr)
		if err != nil {
			writeError(fmt.Sprintf("invalid to: %v", err))
			return
		}
	} else {
		to = time.Now()
	}

	var events []alertEvent

	// Source 1: Alertmanager — active, silenced, and inhibited alerts
	amEvents, err := fetchAlertmanagerAlerts(c, team)
	if err != nil {
		// Non-fatal — log in output and continue
		events = append(events, alertEvent{
			Name:   "⚠ alertmanager fetch error",
			State:  "error",
			Labels: map[string]string{"error": err.Error()},
			Source: "alertmanager",
		})
	} else {
		events = append(events, amEvents...)
	}

	// Source 2: Prometheus rules API — alert instances with activeAt in range
	ruleEvents, err := fetchRuleAlerts(c, team, from, to)
	if err != nil {
		events = append(events, alertEvent{
			Name:   "⚠ rules API fetch error",
			State:  "error",
			Labels: map[string]string{"error": err.Error()},
			Source: "rules",
		})
	} else {
		events = append(events, ruleEvents...)
	}

	// Sort by activeAt descending
	sort.Slice(events, func(i, j int) bool {
		return events[i].ActiveAt > events[j].ActiveAt
	})

	// Build text summary
	var lines []string
	lines = append(lines, fmt.Sprintf("Alert events for team=%s  from=%s  to=%s",
		team, from.Format("2006-01-02"), to.Format("2006-01-02")))
	lines = append(lines, fmt.Sprintf("Total: %d events", len(events)))
	lines = append(lines, "")

	stateCounts := map[string]int{}
	for _, e := range events {
		stateCounts[e.State]++
		crit := e.Labels["criticality"]
		if crit == "" {
			crit = "?"
		}
		lines = append(lines, fmt.Sprintf("[%s] %-12s crit=%-2s  %s  (since %s)",
			e.Source[:2], e.State, crit, e.Name, e.ActiveAt))
	}

	lines = append(lines, "")
	lines = append(lines, "State summary:")
	for state, count := range stateCounts {
		lines = append(lines, fmt.Sprintf("  %-20s %d", state, count))
	}

	writeResponse(response{
		Output: strings.Join(lines, "\n"),
		Data: map[string]any{
			"team":        team,
			"from":        from.Format(time.RFC3339),
			"to":          to.Format(time.RFC3339),
			"total":       len(events),
			"stateCounts": stateCounts,
			"events":      events,
		},
	})
}

func fetchAlertmanagerAlerts(c *client, team string) ([]alertEvent, error) {
	params := url.Values{
		"filter":   []string{"team=" + team},
		"active":   []string{"true"},
		"silenced": []string{"true"},
		"inhibited": []string{"true"},
	}
	body, err := c.get("/api/alertmanager/grafana/api/v2/alerts", params)
	if err != nil {
		return nil, err
	}

	var raw []struct {
		Labels    map[string]string `json:"labels"`
		StartsAt  string            `json:"startsAt"`
		EndsAt    string            `json:"endsAt"`
		Status    struct {
			State string `json:"state"`
		} `json:"status"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("parse alertmanager response: %w", err)
	}

	var events []alertEvent
	for _, a := range raw {
		name := a.Labels["alertname"]
		if name == "" {
			name = a.Labels["__alert_rule_uid__"]
		}
		events = append(events, alertEvent{
			Name:     name,
			State:    a.Status.State,
			ActiveAt: a.StartsAt,
			EndsAt:   a.EndsAt,
			Labels:   a.Labels,
			Source:   "alertmanager",
		})
	}
	return events, nil
}

func fetchRuleAlerts(c *client, team string, from, to time.Time) ([]alertEvent, error) {
	body, err := c.get("/api/prometheus/grafana/api/v1/rules", nil)
	if err != nil {
		return nil, err
	}

	var raw struct {
		Status string `json:"status"`
		Data   struct {
			Groups []struct {
				Name  string `json:"name"`
				Rules []struct {
					Type   string            `json:"type"`
					Name   string            `json:"name"`
					State  string            `json:"state"`
					Labels map[string]string `json:"labels"`
					Alerts []struct {
						Labels    map[string]string `json:"labels"`
						State     string            `json:"state"`
						ActiveAt  string            `json:"activeAt"`
						Value     string            `json:"value"`
					} `json:"alerts"`
				} `json:"rules"`
			} `json:"groups"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("parse rules response: %w", err)
	}

	var events []alertEvent
	for _, group := range raw.Data.Groups {
		for _, rule := range group.Rules {
			if rule.Type != "alerting" {
				continue
			}
			// Check rule-level labels
			ruleTeam := rule.Labels["team"]
			for _, alert := range rule.Alerts {
				// Check alert-level labels too
				alertTeam := alert.Labels["team"]
				if ruleTeam != team && alertTeam != team {
					continue
				}
				// Filter by time range if activeAt is parseable
				if alert.ActiveAt != "" {
					t, err := time.Parse(time.RFC3339, alert.ActiveAt)
					if err == nil {
						if t.Before(from) || t.After(to) {
							continue
						}
					}
				}
				// Merge rule and alert labels
				merged := make(map[string]string)
				for k, v := range rule.Labels {
					merged[k] = v
				}
				for k, v := range alert.Labels {
					merged[k] = v
				}
				events = append(events, alertEvent{
					Name:     rule.Name,
					State:    alert.State,
					ActiveAt: alert.ActiveAt,
					Labels:   merged,
					Source:   "rules",
				})
			}
		}
	}
	return events, nil
}

// ── query-metrics ─────────────────────────────────────────────────────────────
//
// Runs a PromQL range query via the Grafana datasource proxy and returns
// a human-readable summary: series count, min/max/avg per series, last value.

func handleQueryMetrics(c *client, args map[string]any) {
	uid, _ := args["datasource_uid"].(string)
	expr, _ := args["expr"].(string)
	fromStr, _ := args["from"].(string)
	toStr, _ := args["to"].(string)
	step, _ := args["step"].(string)

	if uid == "" {
		writeError("datasource_uid is required")
		return
	}
	if expr == "" {
		writeError("expr is required")
		return
	}

	from, to, err := parseTimeRange(fromStr, toStr)
	if err != nil {
		writeError(err.Error())
		return
	}
	if step == "" {
		// Auto step: aim for ~200 data points
		dur := to.Sub(from)
		autoStep := int(dur.Seconds() / 200)
		if autoStep < 60 {
			autoStep = 60
		}
		step = fmt.Sprintf("%ds", autoStep)
	}

	params := url.Values{
		"query": []string{expr},
		"start": []string{fmt.Sprintf("%d", from.Unix())},
		"end":   []string{fmt.Sprintf("%d", to.Unix())},
		"step":  []string{step},
	}
	path := fmt.Sprintf("/api/datasources/proxy/uid/%s/api/v1/query_range", uid)
	body, err := c.get(path, params)
	if err != nil {
		writeError(fmt.Sprintf("prometheus query failed: %v", err))
		return
	}

	var result struct {
		Status string `json:"status"`
		Data   struct {
			ResultType string `json:"resultType"`
			Result     []struct {
				Metric map[string]string `json:"metric"`
				Values [][]any           `json:"values"`
			} `json:"result"`
		} `json:"data"`
		Error string `json:"error"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		writeError(fmt.Sprintf("parse prometheus response: %v", err))
		return
	}
	if result.Status != "success" {
		writeError(fmt.Sprintf("prometheus error: %s", result.Error))
		return
	}

	type seriesSummary struct {
		Labels map[string]string `json:"labels"`
		Min    float64           `json:"min"`
		Max    float64           `json:"max"`
		Avg    float64           `json:"avg"`
		Last   float64           `json:"last"`
		Points int               `json:"points"`
	}

	var summaries []seriesSummary
	var lines []string
	lines = append(lines, fmt.Sprintf("PromQL: %s", expr))
	lines = append(lines, fmt.Sprintf("Range:  %s → %s  step=%s", from.Format("2006-01-02 15:04"), to.Format("2006-01-02 15:04"), step))
	lines = append(lines, fmt.Sprintf("Series: %d", len(result.Data.Result)))
	lines = append(lines, "")

	for _, series := range result.Data.Result {
		if len(series.Values) == 0 {
			continue
		}
		mn := math.MaxFloat64
		mx := -math.MaxFloat64
		var sum, last float64
		var count int
		for _, pt := range series.Values {
			if len(pt) < 2 {
				continue
			}
			vs, ok := pt[1].(string)
			if !ok {
				continue
			}
			v, err := strconv.ParseFloat(vs, 64)
			if err != nil || math.IsNaN(v) || math.IsInf(v, 0) {
				continue
			}
			if v < mn {
				mn = v
			}
			if v > mx {
				mx = v
			}
			sum += v
			last = v
			count++
		}
		if count == 0 {
			continue
		}
		avg := sum / float64(count)

		// Format metric labels for display
		labelParts := make([]string, 0, len(series.Metric))
		for k, v := range series.Metric {
			if k == "__name__" {
				continue
			}
			labelParts = append(labelParts, fmt.Sprintf("%s=%q", k, v))
		}
		sort.Strings(labelParts)
		labelStr := strings.Join(labelParts, " ")
		if labelStr == "" {
			labelStr = "(no labels)"
		}

		summaries = append(summaries, seriesSummary{
			Labels: series.Metric,
			Min:    mn,
			Max:    mx,
			Avg:    avg,
			Last:   last,
			Points: count,
		})
		lines = append(lines, fmt.Sprintf("  {%s}", labelStr))
		lines = append(lines, fmt.Sprintf("    last=%-12.4g  avg=%-12.4g  min=%-12.4g  max=%-12.4g  (%d points)",
			last, avg, mn, mx, count))
	}

	if len(summaries) == 0 {
		lines = append(lines, "  (no data returned)")
	}

	writeResponse(response{
		Output: strings.Join(lines, "\n"),
		Data: map[string]any{
			"expr":    expr,
			"from":    from.Format(time.RFC3339),
			"to":      to.Format(time.RFC3339),
			"step":    step,
			"series":  len(summaries),
			"results": summaries,
		},
	})
}

// ── query-logs ────────────────────────────────────────────────────────────────
//
// Runs a LogQL range query via the Grafana datasource proxy.
// Returns log lines with timestamps and an error/warning count summary.

func handleQueryLogs(c *client, args map[string]any) {
	uid, _ := args["datasource_uid"].(string)
	query, _ := args["query"].(string)
	fromStr, _ := args["from"].(string)
	toStr, _ := args["to"].(string)
	limit := 100
	if v, ok := args["limit"].(float64); ok && v > 0 {
		limit = int(v)
	}

	if uid == "" {
		writeError("datasource_uid is required")
		return
	}
	if query == "" {
		writeError("query is required")
		return
	}

	from, to, err := parseTimeRange(fromStr, toStr)
	if err != nil {
		writeError(err.Error())
		return
	}

	params := url.Values{
		"query":     []string{query},
		"start":     []string{fmt.Sprintf("%d", from.UnixNano())},
		"end":       []string{fmt.Sprintf("%d", to.UnixNano())},
		"limit":     []string{fmt.Sprintf("%d", limit)},
		"direction": []string{"backward"},
	}
	path := fmt.Sprintf("/api/datasources/proxy/uid/%s/loki/api/v1/query_range", uid)
	body, err := c.get(path, params)
	if err != nil {
		writeError(fmt.Sprintf("loki query failed: %v", err))
		return
	}

	var result struct {
		Status string `json:"status"`
		Data   struct {
			ResultType string `json:"resultType"`
			Result     []struct {
				Stream map[string]string `json:"stream"`
				Values [][]string        `json:"values"` // [nanotimestamp, line]
			} `json:"result"`
		} `json:"data"`
		Error string `json:"error"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		writeError(fmt.Sprintf("parse loki response: %v", err))
		return
	}
	if result.Status != "success" {
		writeError(fmt.Sprintf("loki error: %s", result.Error))
		return
	}

	type logLine struct {
		Timestamp string            `json:"timestamp"`
		Stream    map[string]string `json:"stream"`
		Line      string            `json:"line"`
	}

	var allLines []logLine
	errorCount, warnCount := 0, 0

	for _, stream := range result.Data.Result {
		for _, v := range stream.Values {
			if len(v) < 2 {
				continue
			}
			// Loki timestamps are nanoseconds
			ns, err := strconv.ParseInt(v[0], 10, 64)
			if err != nil {
				continue
			}
			ts := time.Unix(0, ns).UTC().Format("2006-01-02 15:04:05")
			line := v[1]
			lower := strings.ToLower(line)
			if strings.Contains(lower, "error") || strings.Contains(lower, "exception") || strings.Contains(lower, "fatal") {
				errorCount++
			} else if strings.Contains(lower, "warn") {
				warnCount++
			}
			allLines = append(allLines, logLine{
				Timestamp: ts,
				Stream:    stream.Stream,
				Line:      truncate(line, 200),
			})
		}
	}

	// Sort by timestamp descending
	sort.Slice(allLines, func(i, j int) bool {
		return allLines[i].Timestamp > allLines[j].Timestamp
	})

	var textLines []string
	textLines = append(textLines, fmt.Sprintf("LogQL: %s", query))
	textLines = append(textLines, fmt.Sprintf("Range: %s → %s", from.Format("2006-01-02 15:04"), to.Format("2006-01-02 15:04")))
	textLines = append(textLines, fmt.Sprintf("Lines: %d  (errors: %d  warnings: %d)", len(allLines), errorCount, warnCount))
	textLines = append(textLines, "")

	shown := allLines
	if len(shown) > 50 {
		shown = shown[:50]
		textLines = append(textLines, fmt.Sprintf("(showing most recent 50 of %d lines)", len(allLines)))
	}
	for _, l := range shown {
		textLines = append(textLines, fmt.Sprintf("[%s] %s", l.Timestamp, l.Line))
	}

	writeResponse(response{
		Output: strings.Join(textLines, "\n"),
		Data: map[string]any{
			"query":      query,
			"from":       from.Format(time.RFC3339),
			"to":         to.Format(time.RFC3339),
			"totalLines": len(allLines),
			"errors":     errorCount,
			"warnings":   warnCount,
			"lines":      allLines,
		},
	})
}

// ── datasources ───────────────────────────────────────────────────────────────
//
// Lists Grafana datasources, optionally filtered by type (e.g. "prometheus", "loki").

func handleDatasources(c *client, args map[string]any) {
	typeFilter, _ := args["type"].(string)

	body, err := c.get("/api/datasources", nil)
	if err != nil {
		writeError(fmt.Sprintf("list datasources: %v", err))
		return
	}

	var raw []struct {
		ID        int    `json:"id"`
		UID       string `json:"uid"`
		Name      string `json:"name"`
		Type      string `json:"type"`
		IsDefault bool   `json:"isDefault"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		writeError(fmt.Sprintf("parse datasources: %v", err))
		return
	}

	type ds struct {
		ID        int    `json:"id"`
		UID       string `json:"uid"`
		Name      string `json:"name"`
		Type      string `json:"type"`
		IsDefault bool   `json:"isDefault"`
	}

	var results []ds
	for _, d := range raw {
		if typeFilter != "" && !strings.Contains(strings.ToLower(d.Type), strings.ToLower(typeFilter)) {
			continue
		}
		results = append(results, ds{
			ID:        d.ID,
			UID:       d.UID,
			Name:      d.Name,
			Type:      d.Type,
			IsDefault: d.IsDefault,
		})
	}

	// Sort by name
	sort.Slice(results, func(i, j int) bool { return results[i].Name < results[j].Name })

	var lines []string
	lines = append(lines, fmt.Sprintf("Datasources: %d", len(results)))
	if typeFilter != "" {
		lines = append(lines, fmt.Sprintf("Filter: type contains %q", typeFilter))
	}
	lines = append(lines, "")
	for _, d := range results {
		def := ""
		if d.IsDefault {
			def = " [default]"
		}
		lines = append(lines, fmt.Sprintf("  %-50s  uid=%-28s  type=%s%s", d.Name, d.UID, d.Type, def))
	}

	writeResponse(response{
		Output: strings.Join(lines, "\n"),
		Data: map[string]any{
			"count":       len(results),
			"datasources": results,
		},
	})
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func parseTimeRange(fromStr, toStr string) (time.Time, time.Time, error) {
	var from, to time.Time
	var err error

	if fromStr != "" {
		from, err = time.Parse(time.RFC3339, fromStr)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("invalid from %q: use RFC3339 e.g. 2026-03-13T00:00:00Z", fromStr)
		}
	} else {
		from = time.Now().Add(-7 * 24 * time.Hour)
	}

	if toStr != "" {
		to, err = time.Parse(time.RFC3339, toStr)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("invalid to %q: use RFC3339 e.g. 2026-03-20T23:59:59Z", toStr)
		}
	} else {
		to = time.Now()
	}

	return from, to, nil
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "…"
}

func writeResponse(r response) {
	json.NewEncoder(os.Stdout).Encode(r)
}

func writeError(msg string) {
	json.NewEncoder(os.Stdout).Encode(response{Error: msg})
}

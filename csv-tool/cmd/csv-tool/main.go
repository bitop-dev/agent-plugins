package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
)

type request struct {
	Tool      string         `json:"tool"`
	Arguments map[string]any `json:"arguments"`
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
	case "csv/parse":
		csvParse(req.Arguments)
	case "csv/query":
		csvQuery(req.Arguments)
	default:
		writeError(fmt.Sprintf("unknown tool: %s", req.Tool))
	}
}

func csvParse(args map[string]any) {
	data := getString(args, "data", "")
	if data == "" {
		writeError("data is required")
		return
	}

	delimiter := getString(args, "delimiter", ",")
	hasHeader := getBool(args, "has_header", true)
	format := getString(args, "format", "json")
	maxRows := getInt(args, "max_rows", 1000)

	sep := ','
	if delimiter == "\\t" || delimiter == "tab" {
		sep = '\t'
	} else if len(delimiter) == 1 {
		sep = rune(delimiter[0])
	}

	reader := csv.NewReader(strings.NewReader(data))
	reader.Comma = sep
	reader.LazyQuotes = true

	records, err := reader.ReadAll()
	if err != nil {
		writeError(fmt.Sprintf("CSV parse error: %v", err))
		return
	}

	if len(records) == 0 {
		writeResult(map[string]any{"rows": 0, "data": []any{}})
		return
	}

	var headers []string
	startRow := 0
	if hasHeader && len(records) > 0 {
		headers = records[0]
		startRow = 1
	} else {
		for i := range records[0] {
			headers = append(headers, fmt.Sprintf("col%d", i+1))
		}
	}

	// Limit rows
	endRow := len(records)
	if maxRows > 0 && endRow-startRow > maxRows {
		endRow = startRow + maxRows
	}

	switch format {
	case "json":
		var rows []map[string]any
		for _, record := range records[startRow:endRow] {
			row := make(map[string]any)
			for j, val := range record {
				if j < len(headers) {
					// Try to parse as number
					if n, err := strconv.ParseFloat(val, 64); err == nil {
						row[headers[j]] = n
					} else {
						row[headers[j]] = val
					}
				}
			}
			rows = append(rows, row)
		}
		writeResult(map[string]any{
			"rows":    len(rows),
			"columns": headers,
			"data":    rows,
		})

	case "markdown":
		var sb strings.Builder
		sb.WriteString("| " + strings.Join(headers, " | ") + " |\n")
		sb.WriteString("| " + strings.Repeat("--- | ", len(headers)) + "\n")
		for _, record := range records[startRow:endRow] {
			sb.WriteString("| " + strings.Join(record, " | ") + " |\n")
		}
		writeResult(map[string]any{
			"rows":     endRow - startRow,
			"columns":  headers,
			"markdown": sb.String(),
		})

	case "summary":
		writeResult(map[string]any{
			"rows":    endRow - startRow,
			"columns": headers,
			"preview": records[startRow:min(startRow+5, endRow)],
		})

	default:
		writeError(fmt.Sprintf("unknown format: %s (use json, markdown, or summary)", format))
	}
}

func csvQuery(args map[string]any) {
	data := getString(args, "data", "")
	if data == "" {
		writeError("data is required")
		return
	}

	columns := getStringSlice(args, "columns")
	filterCol := getString(args, "filter_column", "")
	filterVal := getString(args, "filter_value", "")
	sortCol := getString(args, "sort_column", "")
	limit := getInt(args, "limit", 0)

	reader := csv.NewReader(strings.NewReader(data))
	reader.LazyQuotes = true
	records, err := reader.ReadAll()
	if err != nil {
		writeError(fmt.Sprintf("CSV parse error: %v", err))
		return
	}
	if len(records) < 2 {
		writeResult(map[string]any{"rows": 0, "data": []any{}})
		return
	}

	headers := records[0]
	headerIdx := make(map[string]int)
	for i, h := range headers {
		headerIdx[h] = i
	}

	// Filter
	var filtered [][]string
	for _, row := range records[1:] {
		if filterCol != "" && filterVal != "" {
			if idx, ok := headerIdx[filterCol]; ok && idx < len(row) {
				if !strings.Contains(strings.ToLower(row[idx]), strings.ToLower(filterVal)) {
					continue
				}
			}
		}
		filtered = append(filtered, row)
	}

	// Sort (simple string sort)
	if sortCol != "" {
		if idx, ok := headerIdx[sortCol]; ok {
			sortRows(filtered, idx)
		}
	}

	// Limit
	if limit > 0 && len(filtered) > limit {
		filtered = filtered[:limit]
	}

	// Select columns
	var selectedHeaders []string
	var colIndices []int
	if len(columns) > 0 {
		for _, c := range columns {
			if idx, ok := headerIdx[c]; ok {
				selectedHeaders = append(selectedHeaders, c)
				colIndices = append(colIndices, idx)
			}
		}
	} else {
		selectedHeaders = headers
		for i := range headers {
			colIndices = append(colIndices, i)
		}
	}

	var rows []map[string]any
	for _, row := range filtered {
		r := make(map[string]any)
		for i, idx := range colIndices {
			if idx < len(row) {
				if n, err := strconv.ParseFloat(row[idx], 64); err == nil {
					r[selectedHeaders[i]] = n
				} else {
					r[selectedHeaders[i]] = row[idx]
				}
			}
		}
		rows = append(rows, r)
	}

	writeResult(map[string]any{
		"rows":    len(rows),
		"columns": selectedHeaders,
		"data":    rows,
	})
}

func sortRows(rows [][]string, idx int) {
	for i := 1; i < len(rows); i++ {
		for j := 0; j < len(rows)-i; j++ {
			a, b := "", ""
			if idx < len(rows[j]) {
				a = rows[j][idx]
			}
			if idx < len(rows[j+1]) {
				b = rows[j+1][idx]
			}
			if a > b {
				rows[j], rows[j+1] = rows[j+1], rows[j]
			}
		}
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

func getStringSlice(m map[string]any, key string) []string {
	if v, ok := m[key]; ok {
		if arr, ok := v.([]any); ok {
			var out []string
			for _, item := range arr {
				if s, ok := item.(string); ok {
					out = append(out, s)
				}
			}
			return out
		}
	}
	return nil
}

func writeResult(content map[string]any) {
	json.NewEncoder(os.Stdout).Encode(response{Status: "ok", Content: content})
}

func writeError(msg string) {
	json.NewEncoder(os.Stdout).Encode(response{Status: "error", Content: map[string]any{"error": msg}})
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

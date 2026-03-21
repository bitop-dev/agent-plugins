package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
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
	case "docker/ps":
		dockerPS(req.Arguments)
	case "docker/logs":
		dockerLogs(req.Arguments)
	case "docker/inspect":
		dockerInspect(req.Arguments)
	case "docker/images":
		dockerImages(req.Arguments)
	default:
		writeError(fmt.Sprintf("unknown tool: %s", req.Tool))
	}
}

func dockerPS(args map[string]any) {
	cmdArgs := []string{"ps", "--format", "{{json .}}"}
	if getBool(args, "all", false) {
		cmdArgs = append(cmdArgs, "-a")
	}
	if f := getString(args, "filter", ""); f != "" {
		cmdArgs = append(cmdArgs, "--filter", f)
	}

	out, err := runDocker(cmdArgs...)
	if err != nil {
		writeError(err.Error())
		return
	}

	var containers []map[string]any
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		if line == "" {
			continue
		}
		var c map[string]any
		if json.Unmarshal([]byte(line), &c) == nil {
			containers = append(containers, c)
		}
	}
	writeResult(map[string]any{"containers": containers, "count": len(containers)})
}

func dockerLogs(args map[string]any) {
	container := getString(args, "container", "")
	if container == "" {
		writeError("container name or ID is required")
		return
	}

	cmdArgs := []string{"logs"}
	if n := getInt(args, "tail", 100); n > 0 {
		cmdArgs = append(cmdArgs, "--tail", fmt.Sprintf("%d", n))
	}
	if getBool(args, "timestamps", false) {
		cmdArgs = append(cmdArgs, "-t")
	}
	cmdArgs = append(cmdArgs, container)

	out, err := runDocker(cmdArgs...)
	if err != nil {
		writeError(err.Error())
		return
	}

	lines := strings.Split(strings.TrimSpace(out), "\n")
	writeResult(map[string]any{
		"container": container,
		"lines":     len(lines),
		"logs":      out,
	})
}

func dockerInspect(args map[string]any) {
	target := getString(args, "target", "")
	if target == "" {
		writeError("target container/image name or ID is required")
		return
	}

	out, err := runDocker("inspect", target)
	if err != nil {
		writeError(err.Error())
		return
	}

	var result []map[string]any
	if json.Unmarshal([]byte(out), &result) == nil && len(result) > 0 {
		writeResult(map[string]any{"inspect": result[0]})
	} else {
		writeResult(map[string]any{"raw": out})
	}
}

func dockerImages(args map[string]any) {
	cmdArgs := []string{"images", "--format", "{{json .}}"}
	if f := getString(args, "filter", ""); f != "" {
		cmdArgs = append(cmdArgs, "--filter", f)
	}

	out, err := runDocker(cmdArgs...)
	if err != nil {
		writeError(err.Error())
		return
	}

	var images []map[string]any
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		if line == "" {
			continue
		}
		var img map[string]any
		if json.Unmarshal([]byte(line), &img) == nil {
			images = append(images, img)
		}
	}
	writeResult(map[string]any{"images": images, "count": len(images)})
}

func runDocker(args ...string) (string, error) {
	cmd := exec.Command("docker", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("docker %s: %s", strings.Join(args, " "), strings.TrimSpace(stderr.String()))
	}
	return stdout.String(), nil
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

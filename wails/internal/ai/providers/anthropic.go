package providers

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"draftline/internal/types"
)

// Claude sends the request to the Anthropic Messages API with SSE streaming.
// The model and API key arrive pre-resolved in the Request.
func Claude(req Request) types.AIRewriteResult {
	req.emit("ai:log", "Connecting to Claude API…")
	result, err := StreamAnthropic(req)
	if err != nil {
		return types.AIRewriteResult{Error: err.Error()}
	}
	return types.AIRewriteResult{Result: result}
}

// StreamAnthropic calls the Anthropic Messages API with SSE streaming.
// It emits ai:token events for each text delta so the frontend can show
// tokens as they arrive. Returns the full accumulated text.
func StreamAnthropic(req Request) (string, error) {
	reqBody, _ := json.Marshal(map[string]any{
		"model":      req.Model,
		"max_tokens": 8192,
		"stream":     true,
		"system":     req.System,
		"messages":   []map[string]string{{"role": "user", "content": req.UserMsg}},
	})

	httpReq, err := http.NewRequestWithContext(req.Ctx, "POST", "https://api.anthropic.com/v1/messages", bytes.NewBuffer(reqBody))
	if err != nil {
		return "", err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", req.APIKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")

	client := &http.Client{}
	resp, err := client.Do(httpReq)
	if err != nil {
		if errors.Is(req.Ctx.Err(), context.Canceled) {
			return "", fmt.Errorf("cancelled")
		}
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != 200 {
		body, _ := readAIResponseBody(resp.Body)
		var apiErr struct {
			Error struct {
				Message string `json:"message"`
			} `json:"error"`
		}
		if json.Unmarshal(body, &apiErr) == nil && apiErr.Error.Message != "" {
			return "", fmt.Errorf("%s", apiErr.Error.Message)
		}
		return "", fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	var sb strings.Builder
	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 64*1024), 64*1024)

	for scanner.Scan() {
		if req.Ctx.Err() != nil {
			return "", fmt.Errorf("cancelled")
		}
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")

		var event struct {
			Type  string `json:"type"`
			Delta struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"delta"`
			Error struct {
				Message string `json:"message"`
			} `json:"error"`
		}
		if json.Unmarshal([]byte(data), &event) != nil {
			continue
		}
		switch event.Type {
		case "content_block_delta":
			if event.Delta.Type == "text_delta" {
				sb.WriteString(event.Delta.Text)
				req.emit("ai:token", event.Delta.Text)
			}
		case "error":
			return "", fmt.Errorf("%s", event.Error.Message)
		}
	}

	if err := scanner.Err(); err != nil {
		if errors.Is(req.Ctx.Err(), context.Canceled) {
			return "", fmt.Errorf("cancelled")
		}
		return "", err
	}

	return strings.TrimSpace(sb.String()), nil
}

package providers

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"
)

// ListModels returns the model ids an endpoint advertises. Anthropic and the
// OpenAI-compatible APIs answer /models in the same shape.
//
// An endpoint that cannot be reached returns nil rather than an error: model
// choice must degrade to a sensible guess, never fail the writer's request.
func ListModels(baseURL, apiKey string, anthropic bool) map[string]bool {
	baseURL = strings.TrimSpace(baseURL)
	if baseURL == "" {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", strings.TrimRight(baseURL, "/")+"/models", nil)
	if err != nil {
		return nil
	}
	if anthropic {
		req.Header.Set("x-api-key", apiKey)
		req.Header.Set("anthropic-version", "2023-06-01")
	} else if apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode >= 400 {
		return nil
	}
	body, err := readAIResponseBody(resp.Body)
	if err != nil {
		return nil
	}
	var list struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if json.Unmarshal(body, &list) != nil || len(list.Data) == 0 {
		return nil
	}
	out := make(map[string]bool, len(list.Data))
	for _, m := range list.Data {
		if m.ID != "" {
			out[m.ID] = true
		}
	}
	return out
}

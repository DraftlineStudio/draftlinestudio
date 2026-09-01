package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"draftline/internal/types"
)

// OpenAI sends the request to the OpenAI chat-completions API.
func OpenAI(req Request) types.AIRewriteResult {
	return openAICompatible(req, "https://api.openai.com/v1/chat/completions", "OpenAI", true)
}

// Grok sends the request to the xAI (Grok) OpenAI-compatible API.
func Grok(req Request) types.AIRewriteResult {
	return openAICompatible(req, "https://api.x.ai/v1/chat/completions", "Grok", true)
}

// Local sends the request to a local OpenAI-compatible endpoint (Ollama,
// LM Studio, …). No auth header is sent; the model comes from settings with a
// llama3 fallback.
func Local(req Request) types.AIRewriteResult {
	req.Model = req.Settings.AILocalModel
	if req.Model == "" {
		req.Model = "llama3"
	}
	endpoint := strings.TrimRight(req.Settings.AILocalEndpoint, "/") + "/chat/completions"
	return openAICompatible(req, endpoint, "local AI", false)
}

// openAICompatible implements the shared chat-completions transport used by
// OpenAI, Grok, and local endpoints. withAuth controls whether the API key is
// sent as a Bearer token (local endpoints receive no credentials).
func openAICompatible(req Request, url, providerLabel string, withAuth bool) types.AIRewriteResult {
	reqBody, _ := json.Marshal(map[string]any{
		"model": req.Model,
		"messages": []map[string]string{
			{"role": "system", "content": req.System},
			{"role": "user", "content": req.UserMsg},
		},
	})

	httpReq, err := http.NewRequestWithContext(req.Ctx, "POST", url, bytes.NewBuffer(reqBody))
	if err != nil {
		return types.AIRewriteResult{Error: err.Error()}
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if withAuth {
		httpReq.Header.Set("Authorization", "Bearer "+req.APIKey)
	}

	// No client timeout: the request context's 180-second deadline governs,
	// and a shorter competing timeout would mask cancellation.
	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		if errors.Is(req.Ctx.Err(), context.Canceled) {
			return types.AIRewriteResult{Error: "cancelled"}
		}
		return types.AIRewriteResult{Error: providerLabel + " request failed: " + err.Error()}
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := readAIResponseBody(resp.Body)
	if err != nil {
		return types.AIRewriteResult{Error: err.Error()}
	}

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return types.AIRewriteResult{Error: "failed to parse " + providerLabel + " response"}
	}
	if result.Error != nil && result.Error.Message != "" {
		return types.AIRewriteResult{Error: result.Error.Message}
	}
	if len(result.Choices) == 0 {
		return types.AIRewriteResult{Error: "empty response from " + providerLabel}
	}
	return types.AIRewriteResult{Result: strings.TrimSpace(result.Choices[0].Message.Content)}
}

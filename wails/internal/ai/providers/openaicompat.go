package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"draftline/internal/types"
)

// OpenAI sends the request to the OpenAI chat-completions API.
func OpenAI(req Request) types.AIRewriteResult {
	return openAICompatible(req, "https://api.openai.com/v1/chat/completions", "OpenAI", true)
}

// Custom sends the request to an endpoint the writer configured. Cloud
// providers get the key as a bearer token; local ones get no credentials.
func Custom(req Request, p types.AIProvider) types.AIRewriteResult {
	req.Model = p.Model
	label := strings.TrimSpace(p.Nickname)
	if label == "" {
		label = "the configured provider"
	}
	endpoint := strings.TrimRight(p.BaseURL, "/") + "/chat/completions"
	return openAICompatible(req, endpoint, label, p.Kind == "cloud")
}

// openAICompatible is the shared chat-completions transport. withAuth controls
// whether the key is sent as a bearer token.
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

	// No client timeout: the request context's deadline governs.
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
	parsed := json.Unmarshal(body, &result) == nil
	if parsed && result.Error != nil && result.Error.Message != "" {
		return types.AIRewriteResult{Error: result.Error.Message}
	}
	// A wrong path or a proxy answers in HTML or plain text, so the status has
	// to carry the message when the body cannot.
	if resp.StatusCode >= 400 {
		return types.AIRewriteResult{Error: fmt.Sprintf("%s returned %d: %s", providerLabel, resp.StatusCode, bodySnippet(body))}
	}
	if !parsed {
		return types.AIRewriteResult{Error: "failed to parse " + providerLabel + " response"}
	}
	if len(result.Choices) == 0 {
		return types.AIRewriteResult{Error: "empty response from " + providerLabel}
	}
	return types.AIRewriteResult{Result: strings.TrimSpace(result.Choices[0].Message.Content)}
}

func bodySnippet(body []byte) string {
	s := strings.TrimSpace(string(body))
	if len(s) > 200 {
		return s[:200] + "…"
	}
	return s
}

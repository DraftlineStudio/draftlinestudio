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

// Gemini sends the request to the Google Gemini generateContent API.
func Gemini(req Request) types.AIRewriteResult {
	// Gemini API uses a different format
	reqBody, _ := json.Marshal(map[string]any{
		"contents": []map[string]any{
			{
				"role":  "user",
				"parts": []map[string]string{{"text": req.UserMsg}},
			},
		},
		"systemInstruction": map[string]any{
			"parts": []map[string]string{{"text": req.System}},
		},
		"generationConfig": map[string]any{
			"temperature":     0.7,
			"maxOutputTokens": 8192,
		},
	})

	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent", req.Model)
	httpReq, err := http.NewRequestWithContext(req.Ctx, "POST", url, bytes.NewBuffer(reqBody))
	if err != nil {
		return types.AIRewriteResult{Error: err.Error()}
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-goog-api-key", req.APIKey)

	// No client timeout: the request context's 180-second deadline governs.
	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		if errors.Is(req.Ctx.Err(), context.Canceled) {
			return types.AIRewriteResult{Error: "cancelled"}
		}
		return types.AIRewriteResult{Error: "Failed to connect to Gemini API: " + err.Error()}
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := readAIResponseBody(resp.Body)
	if err != nil {
		return types.AIRewriteResult{Error: err.Error()}
	}

	var result struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return types.AIRewriteResult{Error: "Failed to parse Gemini response"}
	}
	if result.Error != nil {
		return types.AIRewriteResult{Error: result.Error.Message}
	}
	if len(result.Candidates) == 0 || len(result.Candidates[0].Content.Parts) == 0 {
		return types.AIRewriteResult{Error: "Empty response from Gemini"}
	}
	return types.AIRewriteResult{Result: strings.TrimSpace(result.Candidates[0].Content.Parts[0].Text)}
}

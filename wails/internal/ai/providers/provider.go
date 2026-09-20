// Package providers holds the AI transport layer: HTTP clients for Anthropic,
// OpenAI and user-configured endpoints, plus the helpers the CLI drivers share.
package providers

import (
	"context"
	"fmt"
	"io"

	"draftline/internal/types"
)

// Request carries everything a provider needs. The app resolves the model and
// key, and injects emit so the Wails runtime never leaks in here.
type Request struct {
	Ctx      context.Context
	System   string
	UserMsg  string
	Model    string // already resolved (lightweight profile applied)
	APIKey   string
	Settings types.AppSettings // local endpoint/model
	Emit     func(event string, data any)
}

// emit tolerates a nil Emit so providers can be tested without an event sink.
func (r Request) emit(event string, data any) {
	if r.Emit != nil {
		r.Emit(event, data)
	}
}

// maxAIResponseBytes stops a hostile or broken endpoint exhausting memory.
const maxAIResponseBytes = 16 << 20 // 16 MB

// readAIResponseBody reads a response body up to maxAIResponseBytes.
func readAIResponseBody(r io.Reader) ([]byte, error) {
	body, err := io.ReadAll(io.LimitReader(r, maxAIResponseBytes+1))
	if err != nil {
		return nil, err
	}
	if len(body) > maxAIResponseBytes {
		return nil, fmt.Errorf("response too large (exceeds %d bytes)", maxAIResponseBytes)
	}
	return body, nil
}

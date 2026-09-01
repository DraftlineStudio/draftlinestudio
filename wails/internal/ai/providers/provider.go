// Package providers contains the AI provider transport layer: direct HTTP
// clients for the Anthropic, OpenAI, Gemini, Grok, and local
// (OpenAI-compatible) endpoints, plus the pure helpers shared by the CLI
// drivers that remain in package main. It imports only the standard library
// and internal/types — never the Wails runtime; the app injects an emit
// closure so runtime events stay decoupled.
package providers

import (
	"context"
	"fmt"
	"io"

	"draftline/internal/types"
)

// Request carries everything a provider needs; app.go resolves the model and
// API key and injects an emit closure so the Wails runtime never leaks in here.
type Request struct {
	Ctx      context.Context
	System   string
	UserMsg  string
	Model    string // already resolved (lightweight profile applied)
	APIKey   string
	Settings types.AppSettings // local endpoint/model
	Emit     func(event string, data any)
}

// emit forwards an event to the app's emit closure, tolerating a nil Emit so
// providers can be exercised in tests without wiring up an event sink.
func (r Request) emit(event string, data any) {
	if r.Emit != nil {
		r.Emit(event, data)
	}
}

// maxAIResponseBytes caps how much of a provider HTTP response body we will
// read into memory. It sits comfortably above any plausible max-output-tokens
// payload while preventing a hostile or malfunctioning endpoint from exhausting
// memory via an unbounded body (audit Sol SEC-007).
const maxAIResponseBytes = 16 << 20 // 16 MB

// readAIResponseBody reads a provider response body up to maxAIResponseBytes.
// If the body exceeds the cap it returns a clear "response too large" error
// instead of buffering it all.
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

// Test-only stub sidecar: speaks the Draftline sidecar protocol (JSON-RPC
// 2.0, one object per line over stdio) with behavior selected by the
// STUB_BEHAVIOR environment variable. Built on demand by the supervisor
// tests; never ships.
package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
)

type msg struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      *int64          `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func main() {
	if os.Getenv("STUB_BEHAVIOR") == "crash" {
		fmt.Fprintln(os.Stderr, "stub crashing on purpose")
		os.Exit(1)
	}
	out := json.NewEncoder(os.Stdout)
	// Startup notification (no id).
	_ = out.Encode(msg{JSONRPC: "2.0", Method: "ready", Params: json.RawMessage(`{"ok":true}`)})

	scanner := bufio.NewScanner(os.Stdin)
	scanner.Buffer(make([]byte, 64*1024), 4<<20)
	for scanner.Scan() {
		var in msg
		if err := json.Unmarshal(scanner.Bytes(), &in); err != nil {
			continue
		}
		switch in.Method {
		case "echo":
			_ = out.Encode(msg{JSONRPC: "2.0", ID: in.ID, Result: in.Params})
		case "fail":
			_ = out.Encode(msg{JSONRPC: "2.0", ID: in.ID, Error: &rpcError{Code: -32000, Message: "deliberate failure"}})
		case "shutdown":
			_ = out.Encode(msg{JSONRPC: "2.0", ID: in.ID, Result: json.RawMessage(`null`)})
			return
		default:
			_ = out.Encode(msg{JSONRPC: "2.0", ID: in.ID, Error: &rpcError{Code: -32601, Message: "method not found"}})
		}
	}
}

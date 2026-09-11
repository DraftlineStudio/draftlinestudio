# Sidecar Reference

A sidecar is a plugin's backend: a standalone executable Draftline launches
and supervises. It runs in its own OS process — crash isolation is the
point — and talks to the host over stdin/stdout. Write it in any language
that can read lines and emit JSON.

## Wire protocol

**JSON-RPC 2.0, one JSON object per line (NDJSON), UTF-8.**

- Host → sidecar **requests** arrive on **stdin**, one per line, each with a
  numeric `id`:

  ```json
  {"jsonrpc":"2.0","id":7,"method":"recount","params":{"text":"…"}}
  ```

- The sidecar writes **responses** to **stdout**, echoing the `id`:

  ```json
  {"jsonrpc":"2.0","id":7,"result":{"words":1234}}
  {"jsonrpc":"2.0","id":7,"error":{"code":-32000,"message":"no text given"}}
  ```

- The sidecar may write **notifications** (no `id`) at any time; they surface
  in the plugin's frontend as `host.events.on('<method>', cb)` with `params`
  as the payload:

  ```json
  {"jsonrpc":"2.0","method":"progress","params":{"pct":40}}
  ```

- **stderr** is captured line-by-line into Draftline's log, tagged with the
  plugin id. Log there freely; never write protocol to stderr or logs to
  stdout.

Rules:

- One complete JSON object per line; no pretty-printing across lines. A
  protocol line is capped at **4 MB** — the pipe is for control and status,
  never bulk data. Serve bulk bytes (models, audio, big blobs) from your own
  loopback socket and hand the URL over RPC, the way Draftline's own TTS
  does.
- Unknown methods should answer error `-32601`.
- Requests may arrive while earlier ones are still being processed; respond
  in any order (ids are matched, not sequenced).

## Lifecycle

1. **Start** — lazily, on the plugin's first `host.backend.invoke`. The
   working directory is the plugin's folder. On Windows the process is
   started hidden (no console window). The environment is inherited from
   Draftline.
2. **Serve** — read stdin until EOF, answering requests and emitting
   notifications.
3. **Shutdown** — the host sends a `shutdown` request and then closes stdin.
   Respond and exit promptly; after a **3-second grace** the process is
   killed. EOF on stdin alone must also make you exit (belt and braces).

A minimal main loop therefore is: *read line → dispatch → write line;
`shutdown` or EOF → exit 0*.

## Supervision, timeouts, crashes

- A single request is given **120 seconds**; the host abandons it after
  that (your process keeps running). Long jobs should return immediately
  and report progress via notifications.
- If the process exits unexpectedly, in-flight calls fail with a crash
  report and the sidecar is relaunched on the next invoke.
- **Three** consecutive crashes bench the plugin: further invokes fail fast
  with an explanatory error until the user disables and re-enables it (or
  restarts the app).
- Disabling the plugin, and app quit, both run the shutdown sequence.

## Skeleton (Go)

```go
package main

import (
    "bufio"
    "encoding/json"
    "os"
)

type msg struct {
    JSONRPC string          `json:"jsonrpc"`
    ID      *int64          `json:"id,omitempty"`
    Method  string          `json:"method,omitempty"`
    Params  json.RawMessage `json:"params,omitempty"`
    Result  any             `json:"result,omitempty"`
    Error   *rpcErr         `json:"error,omitempty"`
}
type rpcErr struct {
    Code    int    `json:"code"`
    Message string `json:"message"`
}

func main() {
    out := json.NewEncoder(os.Stdout)
    in := bufio.NewScanner(os.Stdin)
    in.Buffer(make([]byte, 64*1024), 4<<20)
    for in.Scan() {
        var req msg
        if json.Unmarshal(in.Bytes(), &req) != nil {
            continue
        }
        switch req.Method {
        case "recount":
            out.Encode(msg{JSONRPC: "2.0", ID: req.ID, Result: map[string]int{"words": 1234}})
        case "shutdown":
            out.Encode(msg{JSONRPC: "2.0", ID: req.ID, Result: nil})
            return
        default:
            out.Encode(msg{JSONRPC: "2.0", ID: req.ID, Error: &rpcErr{-32601, "method not found"}})
        }
    }
}
```

The same shape in Rust, Python (compiled), C#, etc. works identically — the
host only sees pipes and JSON.

## Packaging

Name your binaries per platform in the manifest's `sidecar` map
(`windows-amd64`, `darwin-arm64`, …). Ship only the platforms you build;
on others the plugin's frontend still runs and `host.backend.invoke`
rejects with a clear message (check `has_sidecar` in your UI if you want to
degrade gracefully).

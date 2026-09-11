package plugins

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"sync"
	"sync/atomic"
	"time"

	"draftline/internal/platform"
)

// Sidecar wire protocol: JSON-RPC 2.0, one JSON object per line (NDJSON) on
// stdin/stdout. Host→plugin requests carry an id and expect a response;
// plugin→host messages without an id are notifications (events) forwarded to
// the frontend. stderr is line-logged, never parsed. The full contract is
// documented in docs/plugins/SIDECAR.md — keep the two in lockstep.

const (
	// shutdownGrace is how long a sidecar gets to exit after the "shutdown"
	// request before it is killed.
	shutdownGrace = 3 * time.Second
	// maxLine caps a single protocol line (a response may carry, e.g., a
	// download status — never bulk data; bulk bytes go over their own
	// channel, like the Read Aloud synthesis socket).
	maxLine = 4 << 20 // 4 MB
	// crashLimit consecutive failed starts/crashes mark the plugin failed
	// until re-enabled.
	crashLimit = 3
)

type rpcRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      *int64          `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type rpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      *int64          `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"` // set on notifications
	Params  json.RawMessage `json:"params,omitempty"` // notification payload
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (e *rpcError) Error() string { return fmt.Sprintf("plugin error %d: %s", e.Code, e.Message) }

// NotifyFunc receives plugin→host notifications (events).
type NotifyFunc func(pluginID, event string, payload json.RawMessage)

// LogFunc receives supervisor and sidecar-stderr log lines.
type LogFunc func(pluginID, line string)

// Supervisor owns every running sidecar. One supervisor per app process.
type Supervisor struct {
	notify NotifyFunc
	logf   LogFunc

	mu       sync.Mutex
	sidecars map[string]*sidecar
	strikes  map[string]int
}

func NewSupervisor(notify NotifyFunc, logf LogFunc) *Supervisor {
	if notify == nil {
		notify = func(string, string, json.RawMessage) {}
	}
	if logf == nil {
		logf = func(string, string) {}
	}
	return &Supervisor{notify: notify, logf: logf, sidecars: map[string]*sidecar{}, strikes: map[string]int{}}
}

type sidecar struct {
	id    string
	cmd   *exec.Cmd
	stdin io.WriteCloser

	writeMu sync.Mutex
	enc     *json.Encoder

	callMu  sync.Mutex
	nextID  int64
	pending map[int64]chan rpcResponse

	done chan struct{} // closed when the reader loop exits
	exit atomic.Bool
}

// Invoke sends one JSON-RPC request to the plugin's sidecar, starting it if
// necessary, and waits for the response or ctx expiry. inst must have a
// sidecar for the running platform.
func (s *Supervisor) Invoke(ctx context.Context, inst Installed, method string, params json.RawMessage) (json.RawMessage, error) {
	sc, err := s.ensure(inst)
	if err != nil {
		return nil, err
	}
	res, err := sc.call(ctx, method, params)
	if err != nil && sc.exit.Load() {
		// The process died under this call; count the strike so a
		// crash-looping plugin gets benched instead of relaunched forever.
		s.mu.Lock()
		s.strikes[inst.Manifest.ID]++
		delete(s.sidecars, inst.Manifest.ID)
		s.mu.Unlock()
	}
	return res, err
}

// Running reports whether the plugin's sidecar is currently up.
func (s *Supervisor) Running(pluginID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	sc, ok := s.sidecars[pluginID]
	return ok && !sc.exit.Load()
}

// ResetStrikes clears the crash counter (called when the user re-enables a
// benched plugin).
func (s *Supervisor) ResetStrikes(pluginID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.strikes[pluginID] = 0
}

// Stop shuts one sidecar down: a "shutdown" request, then a kill after the
// grace period. Safe to call for plugins that aren't running.
func (s *Supervisor) Stop(pluginID string) {
	s.mu.Lock()
	sc, ok := s.sidecars[pluginID]
	delete(s.sidecars, pluginID)
	s.mu.Unlock()
	if !ok {
		return
	}
	sc.shutdown(s.logf)
}

// StopAll shuts every sidecar down (app quit).
func (s *Supervisor) StopAll() {
	s.mu.Lock()
	all := make([]*sidecar, 0, len(s.sidecars))
	for _, sc := range s.sidecars {
		all = append(all, sc)
	}
	s.sidecars = map[string]*sidecar{}
	s.mu.Unlock()
	var wg sync.WaitGroup
	for _, sc := range all {
		wg.Add(1)
		go func(sc *sidecar) {
			defer wg.Done()
			sc.shutdown(s.logf)
		}(sc)
	}
	wg.Wait()
}

func (s *Supervisor) ensure(inst Installed) (*sidecar, error) {
	id := inst.Manifest.ID
	s.mu.Lock()
	defer s.mu.Unlock()
	if sc, ok := s.sidecars[id]; ok && !sc.exit.Load() {
		return sc, nil
	}
	if s.strikes[id] >= crashLimit {
		return nil, fmt.Errorf("plugin %s: sidecar crashed %d times and is disabled until re-enabled", id, s.strikes[id])
	}
	bin := inst.SidecarPath()
	if bin == "" {
		return nil, fmt.Errorf("plugin %s: no sidecar for platform %s", id, PlatformKey())
	}
	sc, err := s.start(id, bin, inst.Dir)
	if err != nil {
		s.strikes[id]++
		return nil, err
	}
	s.sidecars[id] = sc
	return sc, nil
}

func (s *Supervisor) start(id, bin, dir string) (*sidecar, error) {
	// The sidecar's lifetime is supervisor-managed, not call-scoped, so it is
	// deliberately not tied to a request context.
	cmd := exec.Command(bin)
	cmd.Dir = dir
	platform.HideWindow(cmd)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("plugin %s: starting sidecar: %w", id, err)
	}
	sc := &sidecar{
		id:      id,
		cmd:     cmd,
		stdin:   stdin,
		enc:     json.NewEncoder(stdin),
		pending: map[int64]chan rpcResponse{},
		done:    make(chan struct{}),
	}
	go sc.readLoop(stdout, s.notify, s.logf)
	go drainStderr(id, stderr, s.logf)
	s.logf(id, "sidecar started: "+bin)
	return sc, nil
}

func (sc *sidecar) call(ctx context.Context, method string, params json.RawMessage) (json.RawMessage, error) {
	sc.callMu.Lock()
	sc.nextID++
	id := sc.nextID
	ch := make(chan rpcResponse, 1)
	sc.pending[id] = ch
	sc.callMu.Unlock()

	defer func() {
		sc.callMu.Lock()
		delete(sc.pending, id)
		sc.callMu.Unlock()
	}()

	req := rpcRequest{JSONRPC: "2.0", ID: &id, Method: method, Params: params}
	sc.writeMu.Lock()
	err := sc.enc.Encode(req) // Encode appends the newline framing
	sc.writeMu.Unlock()
	if err != nil {
		return nil, fmt.Errorf("plugin %s: write: %w", sc.id, err)
	}

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-sc.done:
		return nil, fmt.Errorf("plugin %s: sidecar exited during call %q", sc.id, method)
	case res := <-ch:
		if res.Error != nil {
			return nil, res.Error
		}
		return res.Result, nil
	}
}

// notifyShutdown sends the shutdown request without waiting for a response
// beyond the grace period.
func (sc *sidecar) shutdown(logf LogFunc) {
	ctx, cancel := context.WithTimeout(context.Background(), shutdownGrace)
	defer cancel()
	_, _ = sc.call(ctx, "shutdown", nil)
	_ = sc.stdin.Close() // EOF is the secondary quit signal
	select {
	case <-sc.done:
	case <-ctx.Done():
		_ = sc.cmd.Process.Kill()
		<-sc.done
	}
	_ = sc.cmd.Wait()
	logf(sc.id, "sidecar stopped")
}

func (sc *sidecar) readLoop(stdout io.Reader, notify NotifyFunc, logf LogFunc) {
	defer func() {
		sc.exit.Store(true)
		close(sc.done)
	}()
	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 64*1024), maxLine)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var msg rpcResponse
		if err := json.Unmarshal(line, &msg); err != nil {
			logf(sc.id, "unparseable protocol line dropped: "+err.Error())
			continue
		}
		switch {
		case msg.ID != nil:
			sc.callMu.Lock()
			ch, ok := sc.pending[*msg.ID]
			sc.callMu.Unlock()
			if ok {
				ch <- msg
			}
		case msg.Method != "":
			notify(sc.id, msg.Method, msg.Params)
		}
	}
	if err := scanner.Err(); err != nil && !errors.Is(err, io.EOF) {
		logf(sc.id, "sidecar stream ended: "+err.Error())
	}
}

func drainStderr(id string, r io.Reader, logf LogFunc) {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 64*1024), maxLine)
	for scanner.Scan() {
		logf(id, scanner.Text())
	}
}

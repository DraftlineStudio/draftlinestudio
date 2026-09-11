package plugins

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// ── Manifest ─────────────────────────────────────────────────────────────────

func validManifest() Manifest {
	return Manifest{
		Schema:     1,
		ID:         "acme.word-goals",
		Name:       "Word Goals",
		Version:    "1.0.0",
		Publisher:  "Acme",
		APIVersion: 1,
		Frontend:   &FrontendSpec{Entry: "frontend/index.js"},
	}
}

func TestManifestValidateAcceptsMinimalFrontendPlugin(t *testing.T) {
	if err := validManifest().Validate(); err != nil {
		t.Fatalf("valid manifest rejected: %v", err)
	}
}

func TestManifestValidateRejections(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(*Manifest)
	}{
		{"wrong schema", func(m *Manifest) { m.Schema = 2 }},
		{"bad id no dot", func(m *Manifest) { m.ID = "wordgoals" }},
		{"bad id uppercase", func(m *Manifest) { m.ID = "Acme.WordGoals" }},
		{"missing name", func(m *Manifest) { m.Name = " " }},
		{"missing version", func(m *Manifest) { m.Version = "" }},
		{"missing publisher", func(m *Manifest) { m.Publisher = "" }},
		{"missing api version", func(m *Manifest) { m.APIVersion = 0 }},
		{"unknown permission", func(m *Manifest) { m.Permissions = []string{"filesystem.everything"} }},
		{"unknown activation", func(m *Manifest) { m.Activation = []string{"onWhatever"} }},
		{"absolute entry", func(m *Manifest) { m.Frontend.Entry = "/etc/passwd" }},
		{"traversal entry", func(m *Manifest) { m.Frontend.Entry = "../outside.js" }},
		{"backslash entry", func(m *Manifest) { m.Frontend.Entry = `frontend\index.js` }},
		{"no code at all", func(m *Manifest) { m.Frontend = nil; m.Sidecar = nil }},
		{"traversal sidecar", func(m *Manifest) {
			m.Sidecar = map[string]string{"windows-amd64": "bin/../../evil.exe"}
		}},
		{"dock bar without id", func(m *Manifest) {
			m.Contributes.EditorDockBars = []DockBarContribution{{}}
		}},
		{"settings section without title", func(m *Manifest) {
			m.Contributes.SettingsSections = []SettingsSectionContribution{{ID: "x"}}
		}},
	}
	for _, tc := range cases {
		m := validManifest()
		tc.mutate(&m)
		if err := m.Validate(); err == nil {
			t.Errorf("%s: expected validation error, got none", tc.name)
		}
	}
}

func TestParseManifestRejectsUnknownFields(t *testing.T) {
	dir := t.TempDir()
	writeManifest(t, dir, `{"schema":1,"id":"a.b","name":"A","version":"1","publisher":"A",
		"api_version":1,"frontend":{"entry":"frontend/index.js"},"totally_new_field":true}`)
	if _, err := ParseManifest(dir); err == nil {
		t.Fatal("unknown manifest field accepted")
	}
}

// ── Discovery ────────────────────────────────────────────────────────────────

func writeManifest(t *testing.T, dir, content string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "manifest.json"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func manifestJSON(id string) string {
	return fmt.Sprintf(`{"schema":1,"id":"%s","name":"P","version":"1.0","publisher":"Pub",
		"api_version":1,"frontend":{"entry":"frontend/index.js"}}`, id)
}

func TestDiscoverPrecedenceAndBrokenPlugins(t *testing.T) {
	dev := t.TempDir()
	shared := t.TempDir()
	user := t.TempDir()

	writeManifest(t, filepath.Join(dev, "acme.tool"), manifestJSON("acme.tool"))
	writeManifest(t, filepath.Join(shared, "acme.tool"), manifestJSON("acme.tool"))
	writeManifest(t, filepath.Join(shared, "acme.other"), manifestJSON("acme.other"))
	writeManifest(t, filepath.Join(user, "acme.broken"), `{"schema":1,`) // invalid JSON
	// A non-plugin directory (no manifest at all) must be skipped silently.
	if err := os.MkdirAll(filepath.Join(shared, ".staging"), 0o755); err != nil {
		t.Fatal(err)
	}

	roots := []Root{{Name: "dev", Dir: dev}, {Name: "shared", Dir: shared}, {Name: "user", Dir: user}}
	got := Discover(roots)

	byID := map[string]Installed{}
	for _, inst := range got {
		byID[inst.Manifest.ID] = inst
	}
	if len(got) != 3 {
		t.Fatalf("expected 3 plugins, got %d: %+v", len(got), got)
	}
	if byID["acme.tool"].Root != "dev" {
		t.Errorf("dev root must shadow shared for the same id; got root %q", byID["acme.tool"].Root)
	}
	if byID["acme.other"].Root != "shared" {
		t.Errorf("acme.other expected from shared, got %q", byID["acme.other"].Root)
	}
	if byID["acme.broken"].LoadError == "" {
		t.Error("broken plugin should be reported with LoadError, not hidden")
	}
}

func TestDiscoverMissingRootsAreSilent(t *testing.T) {
	got := Discover([]Root{{Name: "shared", Dir: filepath.Join(t.TempDir(), "nope")}})
	if len(got) != 0 {
		t.Fatalf("expected nothing, got %+v", got)
	}
}

// ── Supervisor (against the stub sidecar in stub_sidecar_test.go) ───────────

func stubInstalled(t *testing.T, behavior string) Installed {
	t.Helper()
	bin := buildStubSidecar(t)
	dir := filepath.Dir(bin)
	rel := filepath.Base(bin)
	writeManifest(t, dir, fmt.Sprintf(`{"schema":1,"id":"test.stub","name":"Stub","version":"1",
		"publisher":"Test","api_version":1,"sidecar":{"%s":"%s"}}`, PlatformKey(), rel))
	inst, err := ParseManifest(dir)
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("STUB_BEHAVIOR", behavior)
	return Installed{Dir: dir, Manifest: inst, Root: "dev"}
}

func TestSupervisorInvokeRoundTrip(t *testing.T) {
	var events []string
	sup := NewSupervisor(func(id, ev string, _ json.RawMessage) {
		events = append(events, id+":"+ev)
	}, nil)
	defer sup.StopAll()
	inst := stubInstalled(t, "echo")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	res, err := sup.Invoke(ctx, inst, "echo", json.RawMessage(`{"hello":"world"}`))
	if err != nil {
		t.Fatalf("invoke: %v", err)
	}
	var got map[string]string
	if err := json.Unmarshal(res, &got); err != nil || got["hello"] != "world" {
		t.Fatalf("echo result wrong: %s (%v)", res, err)
	}
	if !sup.Running("test.stub") {
		t.Error("sidecar should be running after a call")
	}
	// The stub emits a "ready" notification on startup.
	deadline := time.Now().Add(2 * time.Second)
	for len(events) == 0 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if len(events) == 0 || events[0] != "test.stub:ready" {
		t.Errorf("expected ready notification, got %v", events)
	}
}

func TestSupervisorErrorsAreReturned(t *testing.T) {
	sup := NewSupervisor(nil, nil)
	defer sup.StopAll()
	inst := stubInstalled(t, "echo")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_, err := sup.Invoke(ctx, inst, "fail", nil)
	if err == nil {
		t.Fatal("expected an rpc error")
	}
}

func TestSupervisorStopThenRestart(t *testing.T) {
	sup := NewSupervisor(nil, nil)
	defer sup.StopAll()
	inst := stubInstalled(t, "echo")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if _, err := sup.Invoke(ctx, inst, "echo", json.RawMessage(`{}`)); err != nil {
		t.Fatal(err)
	}
	sup.Stop("test.stub")
	if sup.Running("test.stub") {
		t.Fatal("still running after Stop")
	}
	if _, err := sup.Invoke(ctx, inst, "echo", json.RawMessage(`{}`)); err != nil {
		t.Fatalf("restart after clean stop failed: %v", err)
	}
}

func TestSupervisorCrashStrikesBench(t *testing.T) {
	sup := NewSupervisor(nil, nil)
	defer sup.StopAll()
	inst := stubInstalled(t, "crash")
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	for i := 0; i < crashLimit; i++ {
		if _, err := sup.Invoke(ctx, inst, "echo", json.RawMessage(`{}`)); err == nil {
			t.Fatalf("call %d against crashing sidecar unexpectedly succeeded", i)
		}
	}
	_, err := sup.Invoke(ctx, inst, "echo", json.RawMessage(`{}`))
	if err == nil {
		t.Fatal("benched plugin should refuse to start")
	}
	sup.ResetStrikes("test.stub")
	t.Setenv("STUB_BEHAVIOR", "echo")
	if _, err := sup.Invoke(ctx, inst, "echo", json.RawMessage(`{}`)); err != nil {
		t.Fatalf("re-enabled plugin should run again: %v", err)
	}
}

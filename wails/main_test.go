package main

import (
	"os"
	"testing"
)

// Every test in this package runs against a throwaway configuration directory.
//
// The bound methods write the writer's real settings.json — that is their job —
// and several tests call them. Without this, running the suite silently
// replaced a real Draftline setup with whatever the test happened to be
// asserting about: autosave off, plugins off, test endpoints in the provider
// list. It did exactly that once, which is why this exists.
//
// Anything else a test could reach on the real machine belongs here too. The OS
// keyring is the one that is not covered: no test may call SetAPIKey,
// ClearAPIKey or SetProviderKey, because those reach the live keyring and this
// cannot redirect them.
func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "draftline-test-config-*")
	if err != nil {
		panic("test config directory: " + err.Error())
	}
	configRoot = func() (string, error) { return dir, nil }

	code := m.Run()

	_ = os.RemoveAll(dir)
	os.Exit(code)
}

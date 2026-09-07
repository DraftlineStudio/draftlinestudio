package main

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestProbeCodexCLIUsesLoginStatusInsteadOfCredentialFile(t *testing.T) {
	var calls [][]string
	run := func(_ context.Context, _ string, args ...string) ([]byte, error) {
		calls = append(calls, append([]string(nil), args...))
		if len(args) == 1 && args[0] == "--version" {
			return []byte("codex-cli 0.151.0\n"), nil
		}
		if len(args) == 2 && args[0] == "login" && args[1] == "status" {
			return []byte("Logged in using ChatGPT\n"), nil
		}
		return nil, errors.New("unexpected command")
	}

	version, authenticated := probeCodexCLI(context.Background(), "codex", run)
	if version != "codex-cli 0.151.0" || !authenticated {
		t.Fatalf("unexpected probe result: version=%q authenticated=%v", version, authenticated)
	}
	if len(calls) != 2 || strings.Join(calls[1], " ") != "login status" {
		t.Fatalf("authentication probe did not use the CLI status command: %#v", calls)
	}
}

func TestProbeCodexCLITreatsNonzeroLoginStatusAsSignedOut(t *testing.T) {
	run := func(_ context.Context, _ string, args ...string) ([]byte, error) {
		if len(args) == 1 && args[0] == "--version" {
			return []byte("codex-cli 0.151.0"), nil
		}
		return []byte("Not logged in"), errors.New("exit status 1")
	}

	version, authenticated := probeCodexCLI(context.Background(), "codex", run)
	if version == "" || authenticated {
		t.Fatalf("unexpected signed-out probe result: version=%q authenticated=%v", version, authenticated)
	}
}

func TestPrepareCodexHostCommandSuppliesHomeAndManagedNodePath(t *testing.T) {
	cmd := exec.Command("codex")
	prepareCodexHostCommand(cmd)
	env := envMap(cmd.Env)

	home, err := os.UserHomeDir()
	if err == nil && home != "" {
		if env["HOME"] != home || env["USERPROFILE"] != home {
			t.Fatalf("host home not propagated: HOME=%q USERPROFILE=%q want %q", env["HOME"], env["USERPROFILE"], home)
		}
	}
	if !strings.HasPrefix(env["PATH"], nodeInstallBinDir()) {
		t.Fatalf("managed Node directory is not first on PATH: %q", env["PATH"])
	}
}

func envMap(entries []string) map[string]string {
	result := make(map[string]string, len(entries))
	for _, entry := range entries {
		key, value, found := strings.Cut(entry, "=")
		if found {
			result[strings.ToUpper(key)] = value
		}
	}
	return result
}

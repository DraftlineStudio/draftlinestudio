package main

import (
	"os"
	"runtime"
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

func TestPreferX11OnKDEWayland(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("linux-only decision")
	}
	env := func(vars map[string]string) func(string) string {
		return func(key string) string { return vars[key] }
	}

	kdeWayland := map[string]string{
		"XDG_SESSION_TYPE":    "wayland",
		"XDG_CURRENT_DESKTOP": "KDE",
		"DISPLAY":             ":1",
	}
	if !preferX11OnKDEWayland(env(kdeWayland)) {
		t.Error("KDE Wayland with XWayland available should prefer X11")
	}

	cases := map[string]map[string]string{
		"explicit backend wins": {
			"XDG_SESSION_TYPE": "wayland", "XDG_CURRENT_DESKTOP": "KDE",
			"DISPLAY": ":1", "GDK_BACKEND": "wayland",
		},
		"no XWayland, nothing to prefer": {
			"XDG_SESSION_TYPE": "wayland", "XDG_CURRENT_DESKTOP": "KDE",
		},
		"GNOME negotiates decorations fine": {
			"XDG_SESSION_TYPE": "wayland", "XDG_CURRENT_DESKTOP": "GNOME", "DISPLAY": ":1",
		},
		"X11 session needs no help": {
			"XDG_SESSION_TYPE": "x11", "XDG_CURRENT_DESKTOP": "KDE", "DISPLAY": ":0",
		},
	}
	for name, vars := range cases {
		if preferX11OnKDEWayland(env(vars)) {
			t.Errorf("%s: should not force the X11 backend", name)
		}
	}
}

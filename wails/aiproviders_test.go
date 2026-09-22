package main

import (
	"strings"
	"testing"

	"draftline/internal/types"
)

func TestMigrateLocalEndpointBecomesAProvider(t *testing.T) {
	s, inherits, changed := migrateAIProviders(types.AppSettings{
		AIMode:          "local",
		AILocalEndpoint: "http://localhost:11434/v1",
		AILocalModel:    "llama3",
	})
	if !changed || inherits != "" {
		t.Fatalf("changed=%v inherits=%q", changed, inherits)
	}
	p, ok := findAIProvider(s.AIProviders, "local")
	if !ok {
		t.Fatal("local endpoint was not converted")
	}
	if p.Kind != "local" || p.BaseURL != "http://localhost:11434/v1" || p.Model != "llama3" {
		t.Fatalf("converted badly: %+v", p)
	}
	// "local" stays a valid mode because the migrated provider keeps that id.
	if s.AIMode != "local" {
		t.Fatalf("mode should still resolve, got %q", s.AIMode)
	}
	if s.AILocalEndpoint != "" || s.AILocalModel != "" {
		t.Fatal("legacy fields should be cleared once converted")
	}
}

func TestMigrateGeminiKeepsWorkingAsAConfiguredProvider(t *testing.T) {
	s, inherits, _ := migrateAIProviders(types.AppSettings{
		AIMode:       "api",
		AIProvider:   "gemini",
		AIModel:      "gemini-2.5-flash",
		AITaskRoutes: map[string]string{"expand": "api"},
	})
	if inherits != "gemini" {
		t.Fatalf("the stored key should follow the provider, got %q", inherits)
	}
	p, ok := findAIProvider(s.AIProviders, "gemini")
	if !ok || p.BaseURL != geminiCompatURL || p.Model != "gemini-2.5-flash" {
		t.Fatalf("converted badly: %+v", p)
	}
	if s.AIMode != "gemini" || s.AIProvider != "" {
		t.Fatalf("mode=%q provider=%q", s.AIMode, s.AIProvider)
	}
	if s.AITaskRoutes["expand"] != "gemini" {
		t.Fatalf("task route should follow, got %q", s.AITaskRoutes["expand"])
	}
}

func TestMigrateIsIdempotent(t *testing.T) {
	first, _, _ := migrateAIProviders(types.AppSettings{
		AIMode: "api", AIProvider: "grok", AIModel: "grok-4.3",
	})
	second, inherits, changed := migrateAIProviders(first)
	if changed || inherits != "" {
		t.Fatalf("second pass changed=%v inherits=%q", changed, inherits)
	}
	if len(second.AIProviders) != 1 {
		t.Fatalf("providers duplicated: %d", len(second.AIProviders))
	}
}

func TestMigrateLeavesAnUntouchedSetupAlone(t *testing.T) {
	_, inherits, changed := migrateAIProviders(types.AppSettings{
		AIMode: "api", AIProvider: "claude",
	})
	if changed || inherits != "" {
		t.Fatalf("claude setups need no migration: changed=%v inherits=%q", changed, inherits)
	}
}

// A saved key belongs to the address it was saved for.
//
// The provider form blanks the key field when you open a saved provider for
// editing, so pressing Test on it necessarily asks the backend to fill the key
// in. That is fine while the address is still the one the key was stored
// against, and must not happen once the address has been typed over: the reply
// would otherwise carry the writer's key to whatever server was named.
func TestAStoredKeyIsOnlyFilledInForItsOwnEndpoint(t *testing.T) {
	saved := "https://api.example.test/v1"
	app := &App{}
	app.setSettings(types.AppSettings{AIProviders: []types.AIProvider{{
		ID: "p1", Nickname: "Example", Kind: "cloud", BaseURL: saved, Model: "m", APIKey: "secret",
	}}})

	p, ok := findAIProvider(app.getSettings().AIProviders, "p1")
	if !ok {
		t.Fatal("the provider under test is not configured")
	}
	if got := app.providerKey(p); got != "secret" {
		t.Fatalf("the stored key should be readable for its own provider, got %q", got)
	}

	for _, tc := range []struct {
		name    string
		baseURL string
		want    bool
	}{
		{"its own address", saved, true},
		{"the same address with a trailing slash", saved + "/", true},
		{"the same address in another case", "https://API.EXAMPLE.TEST/v1", true},
		{"an address typed over it", "https://elsewhere.test/v1", false},
		{"a lookalike host", "https://api.example.test.evil.test/v1", false},
		{"no address at all", "", false},
	} {
		if got := sameEndpoint(saved, tc.baseURL); got != tc.want {
			t.Errorf("%s: a key saved for %q %s be sent to %q", tc.name, saved,
				map[bool]string{true: "should", false: "should not"}[tc.want], tc.baseURL)
		}
	}
}

// A cloud provider carries a key, so its address has to be one a key can
// safely travel to. A local provider is never sent one, so plain http stays
// available for Ollama on this machine or a box on the same network.
func TestACloudProviderNeedsASecureAddress(t *testing.T) {
	app := &App{}
	app.setSettings(types.AppSettings{})

	if _, err := app.SaveAIProvider(types.AIProvider{
		Nickname: "Insecure", Kind: "cloud", BaseURL: "http://api.example.test/v1", Model: "m",
	}); err == nil {
		t.Error("a cloud provider on plain http was accepted")
	}

	for _, url := range []string{"http://localhost:11434/v1", "http://192.168.1.50:11434/v1"} {
		if _, err := app.SaveAIProvider(types.AIProvider{
			Nickname: "Local", Kind: "local", BaseURL: url, Model: "m",
		}); err != nil {
			t.Errorf("a local provider at %s was refused: %v", url, err)
		}
	}

	if _, err := app.SaveAIProvider(types.AIProvider{
		Nickname: "Secure", Kind: "cloud", BaseURL: "https://api.example.test/v1", Model: "m",
	}); err != nil {
		t.Errorf("a cloud provider on https was refused: %v", err)
	}
}

// Testing an endpoint with a key in hand is the same promise as saving one.
func TestTestingAnEndpointWillNotSendAKeyInTheClear(t *testing.T) {
	app := &App{}
	app.setSettings(types.AppSettings{})
	res := app.TestAIEndpoint("http://api.example.test/v1", "secret")
	if res.Error == "" {
		t.Fatal("testing a plain http endpoint with a key was allowed")
	}
	if !strings.Contains(res.Error, "https") {
		t.Errorf("the refusal should say why: %q", res.Error)
	}
}

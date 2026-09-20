package main

import (
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

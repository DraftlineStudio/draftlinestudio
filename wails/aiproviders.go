package main

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"time"

	"draftline/internal/ai/providers"
	"draftline/internal/types"

	"github.com/zalando/go-keyring"
)

// Base URLs for the two providers Draftline used to build in. Writers who had
// either configured keep working; new ones add them like any other endpoint.
const (
	geminiCompatURL = "https://generativelanguage.googleapis.com/v1beta/openai"
	grokURL         = "https://api.x.ai/v1"
)

func providerKeyAccount(id string) string { return "ai_provider_" + id }

// providerKey reads a provider's key, preferring the keyring over the
// settings.json fallback used on machines without one.
func (a *App) providerKey(p types.AIProvider) string {
	if p.Kind != "cloud" {
		return ""
	}
	if v, err := keyring.Get(keyringService, providerKeyAccount(p.ID)); err == nil && v != "" {
		return v
	}
	return p.APIKey
}

// SetProviderKey stores a provider's key, falling back to settings.json when
// no keyring is available rather than losing it.
func (a *App) SetProviderKey(id, key string) error {
	key = strings.TrimSpace(key)
	if id == "" {
		return fmt.Errorf("provider id is empty")
	}
	if key == "" {
		return fmt.Errorf("API key is empty")
	}
	s := a.getSettings()
	idx := indexOfProvider(s.AIProviders, id)
	if idx < 0 {
		return fmt.Errorf("no such provider")
	}
	if err := keyring.Set(keyringService, providerKeyAccount(id), key); err != nil {
		log.Printf("keyring unavailable (%v); storing provider key in settings.json", err)
		s.AIProviders[idx].APIKey = key
	} else {
		s.AIProviders[idx].APIKey = ""
	}
	a.setSettings(s)
	return a.writeSettingsFile(s)
}

// SaveAIProvider adds or replaces a provider and returns its id.
func (a *App) SaveAIProvider(p types.AIProvider) (string, error) {
	p.Nickname = strings.TrimSpace(p.Nickname)
	p.BaseURL = strings.TrimSpace(p.BaseURL)
	p.Model = strings.TrimSpace(p.Model)
	if p.Kind != "cloud" && p.Kind != "local" {
		return "", fmt.Errorf("provider kind must be cloud or local")
	}
	if p.BaseURL == "" {
		return "", fmt.Errorf("an endpoint URL is required")
	}
	if p.Nickname == "" {
		return "", fmt.Errorf("a name is required")
	}
	s := a.getSettings()
	if p.ID == "" {
		p.ID = fmt.Sprintf("p%d", time.Now().UnixNano())
	}
	if idx := indexOfProvider(s.AIProviders, p.ID); idx >= 0 {
		p.APIKey = s.AIProviders[idx].APIKey // never round-trips through the frontend
		s.AIProviders[idx] = p
	} else {
		s.AIProviders = append(s.AIProviders, p)
	}
	a.setSettings(s)
	return p.ID, a.writeSettingsFile(s)
}

// DeleteAIProvider removes a provider, its stored key, and any task route or
// default still pointing at it.
func (a *App) DeleteAIProvider(id string) error {
	s := a.getSettings()
	idx := indexOfProvider(s.AIProviders, id)
	if idx < 0 {
		return nil
	}
	s.AIProviders = append(s.AIProviders[:idx], s.AIProviders[idx+1:]...)
	if s.AIMode == id {
		s.AIMode = "claudecode"
	}
	for task, mode := range s.AITaskRoutes {
		if mode == id {
			delete(s.AITaskRoutes, task)
		}
	}
	if err := keyring.Delete(keyringService, providerKeyAccount(id)); err != nil && !errors.Is(err, keyring.ErrNotFound) {
		log.Printf("could not remove provider key from keyring: %v", err)
	}
	a.setSettings(s)
	return a.writeSettingsFile(s)
}

// TestAIEndpoint checks an OpenAI-compatible endpoint answers on /models.
func (a *App) TestAIEndpoint(baseURL, apiKey string) types.AIRewriteResult {
	baseURL = strings.TrimSpace(baseURL)
	if baseURL == "" {
		return types.AIRewriteResult{Error: "No endpoint URL configured"}
	}
	req, err := http.NewRequest("GET", strings.TrimRight(baseURL, "/")+"/models", nil)
	if err != nil {
		return types.AIRewriteResult{Error: err.Error()}
	}
	if key := strings.TrimSpace(apiKey); key != "" {
		req.Header.Set("Authorization", "Bearer "+key)
	}
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return types.AIRewriteResult{Error: "Could not connect: " + err.Error()}
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode >= 400 {
		return types.AIRewriteResult{Error: fmt.Sprintf("Server returned %d", resp.StatusCode)}
	}
	return types.AIRewriteResult{Result: "Connected"}
}

func indexOfProvider(list []types.AIProvider, id string) int {
	for i, p := range list {
		if p.ID == id {
			return i
		}
	}
	return -1
}

// migrateAIProviders converts the legacy local endpoint and the removed Gemini
// and Grok built-ins into configured providers. It reports which provider
// should inherit the single legacy API key, and whether anything changed.
func migrateAIProviders(s types.AppSettings) (types.AppSettings, string, bool) {
	inherits := ""
	before := len(s.AIProviders)
	hadLocal := strings.TrimSpace(s.AILocalEndpoint) != ""
	add := func(p types.AIProvider) {
		if indexOfProvider(s.AIProviders, p.ID) < 0 {
			s.AIProviders = append(s.AIProviders, p)
		}
	}
	if strings.TrimSpace(s.AILocalEndpoint) != "" {
		add(types.AIProvider{
			ID: "local", Nickname: "Local", Kind: "local",
			BaseURL: s.AILocalEndpoint, Model: s.AILocalModel,
		})
		s.AILocalEndpoint, s.AILocalModel = "", ""
	}
	switch s.AIProvider {
	case "gemini":
		add(types.AIProvider{ID: "gemini", Nickname: "Gemini", Kind: "cloud", BaseURL: geminiCompatURL, Model: s.AIModel})
		s.AIProvider, inherits = "", "gemini"
		if s.AIMode == "api" {
			s.AIMode = "gemini"
		}
	case "grok":
		add(types.AIProvider{ID: "grok", Nickname: "Grok", Kind: "cloud", BaseURL: grokURL, Model: s.AIModel})
		s.AIProvider, inherits = "", "grok"
		if s.AIMode == "api" {
			s.AIMode = "grok"
		}
	}
	for task, mode := range s.AITaskRoutes {
		if mode == "api" && inherits != "" {
			s.AITaskRoutes[task] = inherits
		}
	}
	return s, inherits, hadLocal || inherits != "" || len(s.AIProviders) != before
}

// migrateAIProvidersOnce applies the migration at startup and writes it
// through, so every later read sees the converted shape.
func (a *App) migrateAIProvidersOnce(s types.AppSettings) types.AppSettings {
	migrated, inherits, changed := migrateAIProviders(s)
	if !changed {
		return s
	}
	if inherits != "" {
		if key := a.getAPIKey(); key != "" {
			if err := keyring.Set(keyringService, providerKeyAccount(inherits), key); err != nil {
				if idx := indexOfProvider(migrated.AIProviders, inherits); idx >= 0 {
					migrated.AIProviders[idx].APIKey = key
				}
			}
			_ = a.ClearAPIKey()
		}
	}
	if err := a.writeSettingsFile(migrated); err != nil {
		log.Printf("could not save migrated AI providers: %v", err)
	}
	return migrated
}

// chatModelFilter drops ids that cannot answer a chat request, so the model
// dropdown lists prose models rather than the whole catalogue.
var nonChatModel = regexp.MustCompile(`(?i)embed|whisper|tts|audio|speech|image|dall-e|moderation|rerank|transcribe|realtime|search|codex`)

// ListProviderModels asks a built-in provider what it serves, so the model
// dropdown shows what the account can actually use today.
func (a *App) ListProviderModels(provider string) []string {
	switch provider {
	case "claude":
		return sortedChatModels(providers.ListModels(claudeLadder.baseURL, a.getAPIKey(), true))
	case "openai":
		return sortedChatModels(providers.ListModels(openAILadder.baseURL, a.getAPIKey(), false))
	}
	return nil
}

// ListAIProviderModels does the same for an endpoint the writer configured.
// baseURL and apiKey come from the form so the list can be checked before the
// provider is saved.
func (a *App) ListAIProviderModels(baseURL, apiKey, id string) []string {
	if strings.TrimSpace(apiKey) == "" && id != "" {
		if p, ok := findAIProvider(a.getSettings().AIProviders, id); ok {
			apiKey = a.providerKey(p)
		}
	}
	return sortedChatModels(providers.ListModels(baseURL, apiKey, false))
}

func sortedChatModels(models map[string]bool) []string {
	out := make([]string, 0, len(models))
	for id := range models {
		if !nonChatModel.MatchString(id) {
			out = append(out, id)
		}
	}
	sort.Strings(out)
	return out
}

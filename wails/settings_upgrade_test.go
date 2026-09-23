package main

// The upgrade path: what a writer already had must survive the update.
//
// The settings directory helper and the settings snapshot were both reworked
// in the 02698-02725 run. Either could have dropped a field without anything
// failing, and the writer would only find out when a preference came back
// wrong after updating.

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"draftline/internal/types"
)

// A settings file written by the previous build must survive being read and
// written back by this one, field for field. This is the upgrade path: a
// writer updates, Draftline loads what they had, and saves it again.
func TestAnExistingSettingsFileSurvivesTheUpgrade(t *testing.T) {
	dir := t.TempDir()
	configRoot = func() (string, error) { return dir, nil }
	if err := os.MkdirAll(filepath.Join(dir, "draftline"), 0700); err != nil {
		t.Fatal(err)
	}

	// A fully-populated settings file, as 0.21.02688 would have written it.
	original := types.AppSettings{
		DefaultAuthor: "A. Author", DefaultPublisher: "A Press", DefaultCopyright: "legacy",
		DefaultSaveDir: `C:\Books`, DarkMode: true, ThemeMode: "auto",
		AutoThemeUseManual: true, AutoThemeDawn: "05:45", AutoThemeDusk: "20:15",
		ActivityAutoSaveEnabled: true, CustomDictionary: []string{"tesseract", "mother-in-law"},
		SpellCheckEnabled: true, GrammarCheckEnabled: true, CastEnabled: true,
		StoryBibleEnabled: true, AnalysisEnabled: true,
		ReadAloudEnabled: true, ReadAloudVoice: "af_heart", ReadAloudSpeed: 1.25,
		ReadAloudDevice: "native", ReadAloudThreads: "auto", ReadAloudVolume: 0.8, ReadAloudGlow: true,
		AnalysisCPUProfile: "balanced", CharactersLaneView: "heat",
		AIEnabled: true, AIMode: "codex",
		AITaskRoutes: map[string]string{"line_edit": "p1", "expand": "claudecode"},
		AIProviders: []types.AIProvider{
			{ID: "p1", Nickname: "My Ollama", Kind: "local", BaseURL: "http://localhost:11434/v1", Model: "llama3"},
			{ID: "p2", Nickname: "Work", Kind: "cloud", BaseURL: "https://api.example.com/v1", Model: "gpt-5"},
		},
		AIDebugLogging: false, AIModel: "claude-opus-5", ProseGuide: "keep it terse",
		BookFont: "Merriweather", EditorFontSize: "large", BookFontSize: 11,
		BookLineSpacing: "1.5", BookDropCaps: true, BookTrimSize: "6x9",
		SidebarPanelWidth: 420, SidebarActiveSection: "characters", UpdateCheckEnabled: true,
		PluginsEnabled: map[string]bool{"draftline.readaloud": true, "some.other": false},
		PluginSettings: map[string]map[string]any{"draftline.readaloud": {"tone": "warm"}},
	}

	app := &App{}
	raw, err := json.MarshalIndent(original, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(app.settingsPath(), raw, 0600); err != nil {
		t.Fatal(err)
	}

	// Read it the way startup does, then write it back the way a save does.
	loaded := app.loadSettingsFromDisk()
	app.setSettings(loaded)
	if err := app.writeSettingsFile(app.getSettings()); err != nil {
		t.Fatal(err)
	}
	reloaded := app.loadSettingsFromDisk()

	if !reflect.DeepEqual(loaded, reloaded) {
		t.Errorf("a save changed the settings:\n  before: %+v\n  after:  %+v", loaded, reloaded)
	}
	if !reflect.DeepEqual(original, reloaded) {
		t.Errorf("the upgrade changed what the writer had:\n  had: %+v\n  got: %+v", original, reloaded)
	}
}

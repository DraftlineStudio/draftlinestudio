package types

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestStoryFingerprintDoesNotPersistFormattedDiagnostic(t *testing.T) {
	encoded, err := json.Marshal(StoryFingerprint{DiagnosticReport: "quoted manuscript evidence"})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), "diagnostic_report") || strings.Contains(string(encoded), "quoted manuscript evidence") {
		t.Fatalf("formatted diagnostic leaked into persisted fingerprint JSON: %s", encoded)
	}
}

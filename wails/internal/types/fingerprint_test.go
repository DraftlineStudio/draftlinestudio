package types

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestStoryFingerprintDoesNotPersistFormattedDiagnostic(t *testing.T) {
	encoded, err := json.Marshal(StoryFingerprint{CorpusDiagnostic: "corpus quotations", DevelopmentDiagnostic: "development quotations", InspectionDiagnostic: "inspection quotations"})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), "corpus quotations") || strings.Contains(string(encoded), "development quotations") || strings.Contains(string(encoded), "inspection quotations") {
		t.Fatalf("formatted diagnostic leaked into persisted fingerprint JSON: %s", encoded)
	}
}

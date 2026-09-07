package main

import (
	"testing"

	"draftline/internal/types"
)

func TestReportSummaryPreservesUnavailableDiagnostic(t *testing.T) {
	const unavailable = "Fingerprint analysis has not run.\n"
	if got := reportSummary(unavailable); got != "Fingerprint analysis has not run." {
		t.Fatalf("reportSummary() = %q", got)
	}
}

func TestReportSummarySelectsQualityGateCounts(t *testing.T) {
	report := "FINGERPRINT CORPUS DIAGNOSTIC\nEngine: test (schema 5)\nEvidence atoms: 12\nAssertions: 10\nFingerprints: 10\nRelations: 2\nSame-event identities: 1\nState histories: 3\n\n1. hidden detail\n"
	want := "FINGERPRINT CORPUS DIAGNOSTIC\nEngine: test (schema 5)\nEvidence atoms: 12\nAssertions: 10\nFingerprints: 10\nRelations: 2\nSame-event identities: 1\nState histories: 3"
	if got := reportSummary(report); got != want {
		t.Fatalf("reportSummary() = %q, want %q", got, want)
	}
}

func TestSelectReportsKeepsThreeDiagnosticsDistinct(t *testing.T) {
	reports := types.FingerprintTextDiagnostics{Corpus: "corpus", Developments: "developments", Inspections: "inspections"}
	for kind, want := range map[string]string{"corpus": "corpus", "developments": "developments", "inspections": "inspections"} {
		got, ok := selectReports(reports, kind)
		if !ok || len(got) != 1 || got[0] != want {
			t.Fatalf("selectReports(%q) = %#v, %v", kind, got, ok)
		}
	}
}

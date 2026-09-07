package main

import "testing"

func TestReportSummaryPreservesUnavailableDiagnostic(t *testing.T) {
	const unavailable = "Narrative fingerprint analysis has not run.\n"
	if got := reportSummary(unavailable); got != "Narrative fingerprint analysis has not run." {
		t.Fatalf("reportSummary() = %q", got)
	}
}

func TestReportSummarySelectsQualityGateCounts(t *testing.T) {
	report := "NARRATIVE FINGERPRINT DIAGNOSTIC\nEngine: test (schema 4)\nEvidence atoms: 12\nAssertions: 10\nPromoted fingerprints: 2\nRetained as evidence only: 10\n\n1. hidden detail\n"
	want := "Engine: test (schema 4)\nEvidence atoms: 12\nAssertions: 10\nPromoted fingerprints: 2\nRetained as evidence only: 10"
	if got := reportSummary(report); got != want {
		t.Fatalf("reportSummary() = %q, want %q", got, want)
	}
}

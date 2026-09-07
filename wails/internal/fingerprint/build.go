// Package fingerprint builds Draftline's deterministic semantic story model.
package fingerprint

import (
	"sort"
	"time"

	"draftline/internal/types"
)

const (
	engine  = "draftline-manuscript-memory-v5"
	version = 5
)

// Build reconstructs all derived fingerprint data while preserving explicit
// author intent. Later passes enrich assertions, events, states and threads.
func Build(book *types.BookData, progress func(types.StoryAnalysisProgress)) *types.StoryFingerprint {
	prior := book.Analysis.Fingerprint
	result := &types.StoryFingerprint{
		Engine: engine, Version: version, LastAnalyzed: time.Now().UTC().Format(time.RFC3339),
		Contexts: []types.StoryContext{}, TemporalConstraints: []types.TemporalConstraint{},
		Assertions: []types.StoryAssertion{}, NarrativeFingerprints: []types.NarrativeFingerprint{}, NarrativeRelations: []types.NarrativeFingerprintRelation{},
		Fingerprints: []types.ManuscriptFingerprint{}, FingerprintRelations: []types.ManuscriptFingerprintRelation{}, EventIdentities: []types.ManuscriptEventIdentity{},
		StateHistories: []types.FingerprintStateHistory{}, NarrativeDevelopments: []types.NarrativeDevelopment{}, Inspections: []types.FingerprintInspection{},
		Events: []types.FingerprintEvent{}, States: []types.StoryStateInterval{},
		Threads: []types.StoryThread{}, Diagnostics: []types.FingerprintDiagnostic{},
	}
	if book.Analysis.Evidence == nil {
		return result
	}
	result.ContentHash = book.Analysis.Evidence.ContentHash
	if prior != nil {
		result.AuthorModel = prior.AuthorModel
	}
	if progress != nil {
		progress(types.StoryAnalysisProgress{Phase: "chronology", Message: "Resolving story time and reality contexts", Current: 0, Total: len(book.Analysis.Evidence.Records), Percent: 78})
	}
	records := eligibleEvidence(book.Analysis.Evidence.Records)
	reconcileStructureDecisions(result, records)
	contexts, contextByEvidence := inferContexts(records, result.AuthorModel.Contexts)
	result.Contexts = contexts
	applyEvidenceContextCorrections(result.AuthorModel.Corrections, contextByEvidence)
	result.TemporalConstraints = inferTemporalConstraints(records, contextByEvidence)
	points, diagnostics := solveTemporal(records, contextByEvidence, result.TemporalConstraints)
	result.Diagnostics = append(result.Diagnostics, diagnostics...)
	result.Assertions = buildAssertions(records, result.Contexts, contextByEvidence, points)
	applyAssertionCorrections(result.Assertions, result.AuthorModel.Corrections)
	result.Fingerprints, result.FingerprintRelations, result.EventIdentities = buildFingerprintCorpus(result.Assertions, records)
	result.NarrativeFingerprints, result.NarrativeRelations = promoteNarrativeFingerprints(result.Assertions, records)
	applyNarrativeFingerprintCorrections(result.NarrativeFingerprints, result.AuthorModel.Corrections)
	result.Events = legacyEventsFromNarrative(book, result.NarrativeFingerprints, records)
	applyEventCorrections(result.Events, result.AuthorModel.Corrections)
	// Continuity state remains evidence-complete even when a detail is not
	// promoted for narrative display. These support events are internal joins;
	// they are never exposed as narrative fingerprints or timeline nodes.
	supportEvents := consolidateEvents(seedEvents(book, records, contextByEvidence, points), records, result.Assertions, prior)
	result.States = buildStates(supportEvents, records, result.Assertions)
	result.Profiles = deriveProfiles(records, result.AuthorModel.Profiles)
	result.Voices = buildVoiceProfiles(book, result.AuthorModel.VoiceNotes)
	// Thread, structure and arc inference intentionally do not run in schema 4.
	// They may consume narrative fingerprints only after this semantic layer is
	// validated; evidence atoms must never leak into presentation again.
	result.Threads = []types.StoryThread{}
	result.Diagnostics = append(result.Diagnostics, buildDiagnostics(book, result, records, supportEvents)...)
	result.Diagnostics = append(result.Diagnostics, narrativeDiagnostics(result.NarrativeRelations, result.NarrativeFingerprints)...)
	reconcileCorrections(result)
	result.Diagnostics = append(result.Diagnostics, correctionDiagnostics(result.AuthorModel.Corrections)...)
	result.Diagnostics = append(result.Diagnostics, structureDecisionDiagnostics(result.AuthorModel.StructureDecisions)...)
	result.Structure = nil
	promotedEvidence := map[string]bool{}
	for _, fingerprint := range result.NarrativeFingerprints {
		for _, evidenceID := range fingerprint.EvidenceIDs {
			promotedEvidence[evidenceID] = true
		}
	}
	result.PromotionStats = types.NarrativePromotionStats{
		EvidenceAtoms: len(records), Assertions: len(result.Assertions), PromotedFingerprints: len(result.NarrativeFingerprints),
		RetainedAsEvidence: len(records) - len(promotedEvidence),
	}
	result.CorpusStats = types.FingerprintCorpusStats{
		EvidenceAtoms: len(records), Assertions: len(result.Assertions), Fingerprints: len(result.Fingerprints),
		Relations: len(result.FingerprintRelations), EventIdentities: len(result.EventIdentities),
	}
	result.CorpusDiagnostic = buildCorpusDiagnosticReport(result)
	result.DiagnosticReport = buildNarrativeDiagnosticReport(result)
	if progress != nil {
		progress(types.StoryAnalysisProgress{Phase: "chronology", Message: "Chronology model current", Current: len(records), Total: len(records), Percent: 82})
	}
	return result
}

func eligibleEvidence(records []types.EvidenceRecord) []types.EvidenceRecord {
	result := make([]types.EvidenceRecord, 0, len(records))
	for _, record := range records {
		if record.Status != "rejected" {
			result = append(result, record)
		}
	}
	sort.SliceStable(result, func(i, j int) bool {
		if result[i].ChapterIndex != result[j].ChapterIndex {
			return result[i].ChapterIndex < result[j].ChapterIndex
		}
		if result[i].ParagraphIndex != result[j].ParagraphIndex {
			return result[i].ParagraphIndex < result[j].ParagraphIndex
		}
		return result[i].StartOffset < result[j].StartOffset
	})
	return result
}

func seedEvents(book *types.BookData, records []types.EvidenceRecord, contexts map[string]string, points map[string]types.StoryTime) []types.FingerprintEvent {
	result := make([]types.FingerprintEvent, 0, len(records))
	for order, record := range records {
		result = append(result, types.FingerprintEvent{
			ID: stableID("event", record.ID), Summary: mechanicalSummary(record), EvidenceIDs: []string{record.ID},
			CharacterIDs: clone(record.CharacterIDs), CharacterNames: clone(record.CharacterNames), Kinds: []string{record.EvidenceType},
			ContextID: contexts[record.ID], StoryTime: points[record.ID], ChapterID: record.ChapterID,
			ChapterIndex: record.ChapterIndex, ChapterTitle: chapterTitle(book, record), ParagraphIndex: record.ParagraphIndex,
			StartOffset: record.StartOffset, NarrativeOrder: order, Importance: seedImportance(record), Confidence: record.Confidence, Status: record.Status,
		})
	}
	return result
}

func seedImportance(record types.EvidenceRecord) float64 {
	weight := map[string]float64{"discovery": .82, "introduction": .72, "transition": .68, "interaction": .62, "time_reference": .58, "state": .35}[record.EvidenceType]
	if weight == 0 {
		weight = .45
	}
	if record.Pinned || record.Source == "author" {
		return 1
	}
	if len(record.TimeExpressions) > 0 {
		weight += .08
	}
	if weight > 1 {
		weight = 1
	}
	return weight
}

func mechanicalSummary(record types.EvidenceRecord) string {
	if record.AuthorText != "" {
		return record.AuthorText
	}
	return record.Text
}

func chapterTitle(book *types.BookData, record types.EvidenceRecord) string {
	sections := [][]types.ChapterItem{book.FrontMatter, book.Body, book.BackMatter}
	for _, items := range sections {
		for _, chapter := range items {
			if chapter.ID != "" && chapter.ID == record.ChapterID {
				return chapter.Title
			}
		}
	}
	return "Chapter"
}

func clone(values []string) []string { return append([]string(nil), values...) }

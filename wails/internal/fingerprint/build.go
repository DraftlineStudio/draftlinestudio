// Package fingerprint builds Draftline's deterministic semantic story model.
package fingerprint

import (
	"sort"
	"time"

	"draftline/internal/types"
)

const (
	engine  = "draftline-story-fingerprint-v2"
	version = 2
)

// Build reconstructs all derived fingerprint data while preserving explicit
// author intent. Later passes enrich assertions, events, states and threads.
func Build(book *types.BookData, progress func(types.StoryAnalysisProgress)) *types.StoryFingerprint {
	prior := book.Analysis.Fingerprint
	result := &types.StoryFingerprint{
		Engine: engine, Version: version, LastAnalyzed: time.Now().UTC().Format(time.RFC3339),
		Contexts: []types.StoryContext{}, TemporalConstraints: []types.TemporalConstraint{},
		Assertions: []types.StoryAssertion{}, Events: []types.FingerprintEvent{}, States: []types.StoryStateInterval{},
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
	contexts, contextByEvidence := inferContexts(records, result.AuthorModel.Contexts)
	result.Contexts = contexts
	applyEvidenceContextCorrections(result.AuthorModel.Corrections, contextByEvidence)
	result.TemporalConstraints = inferTemporalConstraints(records, contextByEvidence)
	points, diagnostics := solveTemporal(records, contextByEvidence, result.TemporalConstraints)
	result.Diagnostics = append(result.Diagnostics, diagnostics...)
	result.Assertions = buildAssertions(records, contextByEvidence)
	seeded := seedEvents(book, records, contextByEvidence, points)
	result.Events = consolidateEvents(seeded, records, result.Assertions, prior)
	applyEventCorrections(result.Events, result.AuthorModel.Corrections)
	result.States = buildStates(result.Events, records, result.Assertions)
	result.Profiles = deriveProfiles(records, result.AuthorModel.Profiles)
	result.Voices = buildVoiceProfiles(book, result.AuthorModel.VoiceNotes)
	result.Threads = buildThreads(result.Events, records, result.Contexts)
	result.AuthorModel.Checkpoints, diagnostics = evaluateCheckpoints(result.AuthorModel.Checkpoints, result.Events, records)
	result.Diagnostics = append(result.Diagnostics, diagnostics...)
	result.Diagnostics = append(result.Diagnostics, buildDiagnostics(book, result, records)...)
	reconcileCorrections(result)
	result.Diagnostics = append(result.Diagnostics, correctionDiagnostics(result.AuthorModel.Corrections)...)
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

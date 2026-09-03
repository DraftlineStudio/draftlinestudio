package fingerprint

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math"
	"regexp"
	"sort"
	"strings"
	"time"

	"draftline/internal/types"
)

const structureEngine = "draftline-story-structure-v1"

var structuralDecisionRe = regexp.MustCompile(`(?i)\b(?:decid(?:ed|es|ing)|chos(?:e|en)|resolved|refused|agreed|committed|determined|must|will)\b`)

// buildStructure creates author-facing hierarchy without weakening or
// replacing the atomic fingerprint. All joins point back to EvidenceRecord.
func buildStructure(book *types.BookData, fp *types.StoryFingerprint, records []types.EvidenceRecord) *types.StoryStructure {
	s := &types.StoryStructure{ContentHash: fp.ContentHash, Engine: structureEngine, Version: 1, LastAnalyzed: time.Now().UTC().Format(time.RFC3339)}
	s.SignificantEvents = aggregateSignificantEvents(fp, records)
	s.Scenes = aggregateScenes(book, s.SignificantEvents)
	s.Sequences = aggregateSequences(s.Scenes, s.SignificantEvents)
	s.NarrativeThreads = aggregateNarrativeThreads(s.Sequences, s.Scenes, s.SignificantEvents)
	s.Arcs = aggregateArcs(s.NarrativeThreads, s.Sequences, s.Scenes, s.SignificantEvents)
	s.Cache = buildStructureCache(records, s)
	return s
}

func aggregateSignificantEvents(fp *types.StoryFingerprint, records []types.EvidenceRecord) []types.SignificantStoryEvent {
	recordByID := map[string]types.EvidenceRecord{}
	for _, r := range records {
		recordByID[r.ID] = r
	}
	stateByEvent, obligationByEvent := map[string][]string{}, map[string][]string{}
	contradicted := map[string]bool{}
	for _, state := range fp.States {
		stateByEvent[state.StartEventID] = appendUnique(stateByEvent[state.StartEventID], state.ID)
	}
	for _, thread := range fp.Threads {
		if thread.OpenedByEventID != "" {
			obligationByEvent[thread.OpenedByEventID] = appendUnique(obligationByEvent[thread.OpenedByEventID], thread.ID)
		}
		if thread.ResolvedByEventID != "" {
			obligationByEvent[thread.ResolvedByEventID] = appendUnique(obligationByEvent[thread.ResolvedByEventID], thread.ID)
		}
	}
	for _, diagnostic := range fp.Diagnostics {
		if diagnostic.Status != "" {
			continue
		}
		for _, eventID := range diagnostic.EventIDs {
			contradicted[eventID] = true
		}
	}
	termFrequency := map[string]int{}
	for _, e := range fp.Events {
		for _, t := range append(cloneTerms(e.Locations), e.Objects...) {
			termFrequency[termKey(t)]++
		}
	}

	result := []types.SignificantStoryEvent{}
	for _, event := range fp.Events {
		candidate := significantFromFingerprint(event, recordByID, stateByEvent[event.ID], obligationByEvent[event.ID], contradicted[event.ID])
		if len(result) > 0 && significantAffinity(result[len(result)-1], candidate) >= .52 {
			mergeSignificant(&result[len(result)-1], candidate)
			continue
		}
		result = append(result, candidate)
	}
	for i := range result {
		e := &result[i]
		e.ID = stableID("significant-event", strings.Join(e.EvidenceIDs, "|"))
		e.NarrativeOrder = i
		e.Summary = aggregateSummary(*e)
		for _, term := range append(cloneTerms(e.Locations), e.Objects...) {
			if n := termFrequency[termKey(term)] - 1; n > 0 {
				e.SalienceReasons.LaterReferences += math.Min(.16, float64(n)*.025)
			}
		}
		e.Salience = clamp01(sumSalience(e.SalienceReasons))
	}
	return result
}

func significantFromFingerprint(e types.FingerprintEvent, records map[string]types.EvidenceRecord, states, obligations []string, contradiction bool) types.SignificantStoryEvent {
	end := e.StartOffset
	section, sectionIndex := "body", e.ChapterIndex
	sourceParts := []string{}
	for _, id := range e.EvidenceIDs {
		if found, ok := records[id]; ok {
			sourceParts = append(sourceParts, found.Text)
			if found.EndOffset > end {
				end = found.EndOffset
			}
			section, sectionIndex = found.Section, found.SectionIndex
		}
	}
	reasons := types.SalienceBreakdown{Base: e.Importance * .5}
	for _, kind := range e.Kinds {
		switch kind {
		case "discovery", "knowledge_change":
			reasons.Discovery = math.Max(reasons.Discovery, .22)
		case "transition":
			reasons.Movement = math.Max(reasons.Movement, .18)
		case "interaction", "knowledge_transfer":
			reasons.Base += .08
		case "state":
			reasons.DescriptionOnly = .08
		}
	}
	if len(states) > 0 {
		reasons.StateChange = math.Min(.18, float64(len(states))*.06)
	}
	if len(obligations) > 0 {
		reasons.ThreadChange = .24
	}
	if structuralDecisionRe.MatchString(strings.Join(sourceParts, " ")) {
		reasons.Decision = .16
	}
	if contradiction {
		reasons.Contradiction = .2
	}
	if e.StoryTime.DayOffset != nil || e.StoryTime.Precision == "relative" {
		reasons.TemporalChange = .08
	}
	if e.Confidence < .6 {
		reasons.LowConfidence = .12
	}
	return types.SignificantStoryEvent{Summary: e.Summary, AuthorSummary: e.AuthorSummary, EvidenceIDs: clone(e.EvidenceIDs), FingerprintEventIDs: []string{e.ID}, AssertionIDs: clone(e.AssertionIDs), CharacterIDs: clone(e.CharacterIDs), CharacterNames: clone(e.CharacterNames), Locations: cloneTerms(e.Locations), Objects: cloneTerms(e.Objects), Kinds: clone(e.Kinds), StateIDs: clone(states), ObligationThreadIDs: clone(obligations), ContextID: e.ContextID, StoryTime: e.StoryTime, ChapterID: e.ChapterID, ChapterIndex: e.ChapterIndex, ChapterTitle: e.ChapterTitle, Section: section, SectionIndex: sectionIndex, ParagraphStart: e.ParagraphIndex, ParagraphEnd: e.ParagraphIndex, StartOffset: e.StartOffset, EndOffset: end, SalienceReasons: reasons, Salience: clamp01(sumSalience(reasons)), Confidence: e.Confidence}
}

func significantAffinity(a, b types.SignificantStoryEvent) float64 {
	if a.ChapterID != b.ChapterID || a.ContextID != b.ContextID || b.ParagraphStart-a.ParagraphEnd > 1 {
		return 0
	}
	if a.StoryTime.DayOffset != nil && b.StoryTime.DayOffset != nil && *a.StoryTime.DayOffset != *b.StoryTime.DayOffset {
		return 0
	}
	score := 0.0
	if a.ParagraphEnd == b.ParagraphStart {
		score += .34
	} else {
		score += .12
	}
	if stringOverlap(a.CharacterIDs, b.CharacterIDs) {
		score += .25
	}
	if termOverlap(a.Locations, b.Locations) {
		score += .18
	}
	if termOverlap(a.Objects, b.Objects) {
		score += .12
	}
	if complementaryKinds(a.Kinds, b.Kinds) {
		score += .15
	}
	// A hard movement or thread transition starts its own significant beat
	// unless the records otherwise describe the same participants/place.
	if contains(b.Kinds, "transition") && !stringOverlap(a.CharacterIDs, b.CharacterIDs) {
		score -= .25
	}
	if len(a.FingerprintEventIDs) >= 6 {
		score -= .3
	}
	return score
}

func complementaryKinds(a, b []string) bool {
	return (contains(a, "state") && (contains(b, "discovery") || contains(b, "transition"))) ||
		(contains(b, "state") && (contains(a, "discovery") || contains(a, "transition"))) ||
		stringOverlap(a, b)
}

func mergeSignificant(a *types.SignificantStoryEvent, b types.SignificantStoryEvent) {
	a.EvidenceIDs = appendUnique(a.EvidenceIDs, b.EvidenceIDs...)
	a.FingerprintEventIDs = appendUnique(a.FingerprintEventIDs, b.FingerprintEventIDs...)
	a.AssertionIDs = appendUnique(a.AssertionIDs, b.AssertionIDs...)
	a.CharacterIDs = appendUnique(a.CharacterIDs, b.CharacterIDs...)
	a.CharacterNames = appendUnique(a.CharacterNames, b.CharacterNames...)
	a.Kinds = appendUnique(a.Kinds, b.Kinds...)
	a.StateIDs = appendUnique(a.StateIDs, b.StateIDs...)
	a.ObligationThreadIDs = appendUnique(a.ObligationThreadIDs, b.ObligationThreadIDs...)
	a.Locations = appendUniqueTerms(a.Locations, b.Locations)
	a.Objects = appendUniqueTerms(a.Objects, b.Objects)
	a.ParagraphEnd = b.ParagraphEnd
	if b.EndOffset > a.EndOffset {
		a.EndOffset = b.EndOffset
	}
	if b.Confidence < a.Confidence {
		a.Confidence = b.Confidence
	}
	if b.StoryTime.Confidence > a.StoryTime.Confidence {
		a.StoryTime = b.StoryTime
	}
	a.SalienceReasons = mergeReasons(a.SalienceReasons, b.SalienceReasons)
	if a.AuthorSummary == "" && contains(b.Kinds, "discovery") && b.Summary != "" {
		a.Summary = b.Summary
	}
}

func aggregateSummary(e types.SignificantStoryEvent) string {
	if e.AuthorSummary != "" {
		return e.AuthorSummary
	}
	name := first(e.CharacterNames)
	place := firstTerm(e.Locations)
	if contains(e.Kinds, "transition") && name != "" {
		if place != "" {
			return name + " moves to " + place
		}
		return name + " changes location"
	}
	if contains(e.Kinds, "discovery") && name != "" {
		if obj := firstTerm(e.Objects); obj != "" {
			return name + " discovers " + obj
		}
		return name + " makes a discovery"
	}
	if strings.TrimSpace(e.Summary) != "" && len([]rune(e.Summary)) <= 120 {
		return strings.TrimSpace(e.Summary)
	}
	if len(e.FingerprintEventIDs) > 1 && name != "" {
		if place != "" {
			return name + " acts at " + place
		}
		return name + " advances the scene"
	}
	return "A significant story beat unfolds"
}

func aggregateScenes(book *types.BookData, events []types.SignificantStoryEvent) []types.SemanticScene {
	breaks := explicitBreakOffsets(book)
	result := []types.SemanticScene{}
	for _, e := range events {
		boundary, source, confidence := len(result) == 0, "manuscript-start", 1.0
		if !boundary {
			boundary, source, confidence = sceneBoundary(result[len(result)-1], e, breaks[e.ChapterIndex])
		}
		if boundary {
			result = append(result, sceneFromEvent(e, source, confidence))
			continue
		}
		mergeScene(&result[len(result)-1], e)
	}
	for i := range result {
		result[i].ID = stableID("semantic-scene", strings.Join(result[i].EvidenceIDs, "|"))
		result[i].Summary = sceneSummary(result[i], events)
	}
	return result
}

func explicitBreakOffsets(book *types.BookData) map[int][]int {
	result := map[int][]int{}
	if book.Analysis.Relationships == nil {
		return result
	}
	for _, scene := range book.Analysis.Relationships.Scenes {
		if scene.SceneType == "scene_break" {
			result[scene.ChapterIndex] = append(result[scene.ChapterIndex], scene.StartOffset)
		}
	}
	return result
}

func sceneBoundary(prev types.SemanticScene, e types.SignificantStoryEvent, breaks []int) (bool, string, float64) {
	if e.ChapterIndex != prev.ChapterEnd {
		return true, "chapter", .98
	}
	if e.ContextID != prev.ContextID {
		return true, "temporal-context", .94
	}
	for _, offset := range breaks {
		if offset > 0 && offset <= e.StartOffset && offset >= lastEventOffset(prev) {
			return true, "explicit-separator", .98
		}
	}
	if e.ParagraphStart-prev.ParagraphEnd > 4 {
		return true, "prose-gap", .7
	}
	if len(prev.Locations) > 0 && len(e.Locations) > 0 && !termOverlap(prev.Locations, e.Locations) && contains(e.Kinds, "transition") {
		return true, "location-transition", .84
	}
	return false, "continuity", .72
}

func sceneFromEvent(e types.SignificantStoryEvent, source string, confidence float64) types.SemanticScene {
	return types.SemanticScene{EventIDs: []string{e.ID}, EvidenceIDs: clone(e.EvidenceIDs), CharacterIDs: clone(e.CharacterIDs), CharacterNames: clone(e.CharacterNames), Locations: cloneTerms(e.Locations), ObjectiveTerms: cloneTerms(e.Objects), ContextID: e.ContextID, StoryTime: e.StoryTime, ChapterIDs: []string{e.ChapterID}, ChapterStart: e.ChapterIndex, ChapterEnd: e.ChapterIndex, NarrativeStart: e.NarrativeOrder, NarrativeEnd: e.NarrativeOrder, StartOffset: e.StartOffset, EndOffset: e.EndOffset, ParagraphEnd: e.ParagraphEnd, Salience: e.Salience, Confidence: e.Confidence, BoundarySource: source, BoundaryConfidence: confidence}
}

func mergeScene(s *types.SemanticScene, e types.SignificantStoryEvent) {
	s.EventIDs = append(s.EventIDs, e.ID)
	s.EvidenceIDs = appendUnique(s.EvidenceIDs, e.EvidenceIDs...)
	s.CharacterIDs = appendUnique(s.CharacterIDs, e.CharacterIDs...)
	s.CharacterNames = appendUnique(s.CharacterNames, e.CharacterNames...)
	s.Locations = appendUniqueTerms(s.Locations, e.Locations)
	s.ObjectiveTerms = appendUniqueTerms(s.ObjectiveTerms, e.Objects)
	s.ChapterIDs = appendUnique(s.ChapterIDs, e.ChapterID)
	s.ChapterEnd = e.ChapterIndex
	s.NarrativeEnd = e.NarrativeOrder
	s.EndOffset = e.EndOffset
	s.ParagraphEnd = e.ParagraphEnd
	s.Salience = math.Max(s.Salience, e.Salience)
	if e.Confidence < s.Confidence {
		s.Confidence = e.Confidence
	}
}

func aggregateSequences(scenes []types.SemanticScene, events []types.SignificantStoryEvent) []types.StorySequence {
	result := []types.StorySequence{}
	for _, scene := range scenes {
		if len(result) == 0 || sequenceAffinity(result[len(result)-1], scene) < .38 || len(result[len(result)-1].SceneIDs) >= 8 {
			result = append(result, sequenceFromScene(scene))
			continue
		}
		mergeSequence(&result[len(result)-1], scene)
	}
	for i := range result {
		q := &result[i]
		q.ID = stableID("story-sequence", strings.Join(q.EvidenceIDs, "|"))
		q.Summary = sequenceSummary(*q, scenes, events)
	}
	return result
}

func sequenceAffinity(q types.StorySequence, s types.SemanticScene) float64 {
	score := 0.0
	if stringOverlap(q.ContextIDs, []string{s.ContextID}) {
		score += .2
	}
	if stringOverlap(q.CharacterIDs, s.CharacterIDs) {
		score += .28
	}
	if termOverlap(q.Locations, s.Locations) {
		score += .22
	}
	if termOverlap(q.ObjectiveTerms, s.ObjectiveTerms) {
		score += .25
	}
	if s.NarrativeStart-q.NarrativeEnd == 1 {
		score += .08
	}
	return score
}
func sequenceFromScene(s types.SemanticScene) types.StorySequence {
	return types.StorySequence{SceneIDs: []string{s.ID}, EventIDs: clone(s.EventIDs), EvidenceIDs: clone(s.EvidenceIDs), CharacterIDs: clone(s.CharacterIDs), CharacterNames: clone(s.CharacterNames), ContextIDs: []string{s.ContextID}, Locations: cloneTerms(s.Locations), ObjectiveTerms: cloneTerms(s.ObjectiveTerms), NarrativeStart: s.NarrativeStart, NarrativeEnd: s.NarrativeEnd, Salience: s.Salience, Confidence: s.Confidence}
}
func mergeSequence(q *types.StorySequence, s types.SemanticScene) {
	q.SceneIDs = append(q.SceneIDs, s.ID)
	q.EventIDs = appendUnique(q.EventIDs, s.EventIDs...)
	q.EvidenceIDs = appendUnique(q.EvidenceIDs, s.EvidenceIDs...)
	q.CharacterIDs = appendUnique(q.CharacterIDs, s.CharacterIDs...)
	q.CharacterNames = appendUnique(q.CharacterNames, s.CharacterNames...)
	q.ContextIDs = appendUnique(q.ContextIDs, s.ContextID)
	q.Locations = appendUniqueTerms(q.Locations, s.Locations)
	q.ObjectiveTerms = appendUniqueTerms(q.ObjectiveTerms, s.ObjectiveTerms)
	q.NarrativeEnd = s.NarrativeEnd
	q.Salience = math.Max(q.Salience, s.Salience)
	if s.Confidence < q.Confidence {
		q.Confidence = s.Confidence
	}
}

func aggregateNarrativeThreads(sequences []types.StorySequence, scenes []types.SemanticScene, events []types.SignificantStoryEvent) []types.NarrativeThread {
	obligationsByEvent := map[string][]string{}
	for _, event := range events {
		obligationsByEvent[event.ID] = event.ObligationThreadIDs
	}
	result := []types.NarrativeThread{}
	for _, q := range sequences {
		matches := []int{}
		for i := range result {
			if threadAffinity(result[i], q) >= .34 {
				matches = append(matches, i)
			}
		}
		if len(matches) == 0 {
			result = append(result, threadFromSequence(q, obligationsByEvent))
			continue
		}
		for _, index := range matches {
			mergeNarrativeThread(&result[index], q)
		}
		if len(matches) > 1 && len(q.EventIDs) > 0 {
			for _, index := range matches {
				result[index].ConvergenceEventIDs = appendUnique(result[index].ConvergenceEventIDs, q.EventIDs[0])
			}
		}
	}
	for i := range result {
		t := &result[i]
		for _, eventID := range t.EventIDs {
			t.ObligationThreadIDs = appendUnique(t.ObligationThreadIDs, obligationsByEvent[eventID]...)
		}
		t.ID = stableID("narrative-thread", strings.Join(t.EvidenceIDs, "|"))
		t.Label = threadLabel(*t)
		if t.NarrativeEnd < lastNarrative(events)-3 {
			t.State = "dormant"
		}
	}
	// A later sequence matching only one side naturally separates the lanes;
	// the shared convergence event remains on both for projection coverage.
	return result
}

func threadFromSequence(q types.StorySequence, obligations map[string][]string) types.NarrativeThread {
	thread := types.NarrativeThread{SequenceIDs: []string{q.ID}, SceneIDs: clone(q.SceneIDs), EventIDs: clone(q.EventIDs), EvidenceIDs: clone(q.EvidenceIDs), CharacterIDs: clone(q.CharacterIDs), CharacterNames: clone(q.CharacterNames), ObjectiveTerms: termTexts(q.ObjectiveTerms), LocationTerms: termTexts(q.Locations), ContextIDs: clone(q.ContextIDs), NarrativeStart: q.NarrativeStart, NarrativeEnd: q.NarrativeEnd, State: "active", Salience: q.Salience, Confidence: q.Confidence}
	for _, eventID := range q.EventIDs {
		thread.ObligationThreadIDs = appendUnique(thread.ObligationThreadIDs, obligations[eventID]...)
	}
	return thread
}
func threadAffinity(t types.NarrativeThread, q types.StorySequence) float64 {
	score := 0.0
	if stringOverlap(t.CharacterIDs, q.CharacterIDs) {
		score += .28
	}
	if stringOverlap(t.ObjectiveTerms, termTexts(q.ObjectiveTerms)) {
		score += .36
	}
	if stringOverlap(t.LocationTerms, termTexts(q.Locations)) {
		score += .12
	}
	if stringOverlap(t.ContextIDs, q.ContextIDs) {
		score += .08
	}
	if q.NarrativeStart-t.NarrativeEnd <= 3 {
		score += .12
	}
	return score
}
func mergeNarrativeThread(t *types.NarrativeThread, q types.StorySequence) {
	t.SequenceIDs = append(t.SequenceIDs, q.ID)
	t.SceneIDs = appendUnique(t.SceneIDs, q.SceneIDs...)
	t.EventIDs = appendUnique(t.EventIDs, q.EventIDs...)
	t.EvidenceIDs = appendUnique(t.EvidenceIDs, q.EvidenceIDs...)
	t.CharacterIDs = appendUnique(t.CharacterIDs, q.CharacterIDs...)
	t.CharacterNames = appendUnique(t.CharacterNames, q.CharacterNames...)
	t.ObjectiveTerms = appendUnique(t.ObjectiveTerms, termTexts(q.ObjectiveTerms)...)
	t.LocationTerms = appendUnique(t.LocationTerms, termTexts(q.Locations)...)
	t.ContextIDs = appendUnique(t.ContextIDs, q.ContextIDs...)
	t.NarrativeEnd = q.NarrativeEnd
	t.Salience = math.Max(t.Salience, q.Salience)
	if q.Confidence < t.Confidence {
		t.Confidence = q.Confidence
	}
}

func aggregateArcs(threads []types.NarrativeThread, sequences []types.StorySequence, scenes []types.SemanticScene, events []types.SignificantStoryEvent) []types.StoryArc {
	if len(sequences) == 0 {
		return []types.StoryArc{}
	}
	// Conservative structural acts: contiguous sequence ranges. They are not
	// presented as literary diagnoses, only zoom-level containers.
	count := int(math.Ceil(float64(len(sequences)) / 6))
	if count < 1 {
		count = 1
	}
	if count > 5 {
		count = 5
	}
	size := int(math.Ceil(float64(len(sequences)) / float64(count)))
	result := []types.StoryArc{}
	for start := 0; start < len(sequences); start += size {
		end := start + size
		if end > len(sequences) {
			end = len(sequences)
		}
		a := types.StoryArc{Label: fmt.Sprintf("Story movement %d", len(result)+1), NarrativeStart: sequences[start].NarrativeStart, NarrativeEnd: sequences[end-1].NarrativeEnd, Confidence: .65}
		for _, q := range sequences[start:end] {
			a.SequenceIDs = append(a.SequenceIDs, q.ID)
			a.SceneIDs = appendUnique(a.SceneIDs, q.SceneIDs...)
			a.EventIDs = appendUnique(a.EventIDs, q.EventIDs...)
			a.EvidenceIDs = appendUnique(a.EvidenceIDs, q.EvidenceIDs...)
			a.Salience = math.Max(a.Salience, q.Salience)
		}
		for _, t := range threads {
			if rangesOverlap(a.NarrativeStart, a.NarrativeEnd, t.NarrativeStart, t.NarrativeEnd) {
				a.ThreadIDs = append(a.ThreadIDs, t.ID)
			}
		}
		a.ID = stableID("story-arc", strings.Join(a.EvidenceIDs, "|"))
		result = append(result, a)
	}
	return result
}

func buildStructureCache(records []types.EvidenceRecord, s *types.StoryStructure) types.StoryStructureCache {
	c := types.StoryStructureCache{EvidenceDependents: map[string][]string{}, AggregateParents: map[string][]string{}}
	grouped := map[string]*types.StoryStructureBlockCache{}
	blockByEvidence := map[string]*types.StoryStructureBlockCache{}
	blockText := map[string][]string{}
	for _, r := range records {
		k := fmt.Sprintf("%s\x00%d", r.ChapterID, r.ParagraphIndex)
		b := grouped[k]
		if b == nil {
			b = &types.StoryStructureBlockCache{ID: stableID("source-block", k), ChapterID: r.ChapterID, ChapterIndex: r.ChapterIndex, ParagraphIndex: r.ParagraphIndex}
			grouped[k] = b
		}
		b.EvidenceIDs = appendUnique(b.EvidenceIDs, r.ID)
		blockByEvidence[r.ID] = b
		blockText[k] = append(blockText[k], strings.ToLower(strings.Join(strings.Fields(r.Text), " ")))
	}
	for key, b := range grouped {
		h := sha256.Sum256([]byte(strings.Join(blockText[key], "\x00")))
		b.ContentHash = hex.EncodeToString(h[:10])
	}
	for _, e := range s.SignificantEvents {
		for _, id := range e.EvidenceIDs {
			c.EvidenceDependents[id] = appendUnique(c.EvidenceDependents[id], e.ID)
			if block := blockByEvidence[id]; block != nil {
				block.AggregateIDs = appendUnique(block.AggregateIDs, e.ID)
			}
		}
	}
	for _, scene := range s.Scenes {
		for _, id := range scene.EventIDs {
			c.AggregateParents[id] = appendUnique(c.AggregateParents[id], scene.ID)
		}
	}
	for _, q := range s.Sequences {
		for _, id := range q.SceneIDs {
			c.AggregateParents[id] = appendUnique(c.AggregateParents[id], q.ID)
		}
	}
	for _, t := range s.NarrativeThreads {
		for _, id := range t.SequenceIDs {
			c.AggregateParents[id] = appendUnique(c.AggregateParents[id], t.ID)
		}
	}
	for _, a := range s.Arcs {
		for _, id := range a.ThreadIDs {
			c.AggregateParents[id] = appendUnique(c.AggregateParents[id], a.ID)
		}
	}
	for _, b := range grouped {
		c.SourceBlocks = append(c.SourceBlocks, *b)
	}
	sort.Slice(c.SourceBlocks, func(i, j int) bool {
		if c.SourceBlocks[i].ChapterIndex != c.SourceBlocks[j].ChapterIndex {
			return c.SourceBlocks[i].ChapterIndex < c.SourceBlocks[j].ChapterIndex
		}
		return c.SourceBlocks[i].ParagraphIndex < c.SourceBlocks[j].ParagraphIndex
	})
	return c
}

func sceneSummary(s types.SemanticScene, events []types.SignificantStoryEvent) string {
	for _, e := range events {
		if e.ID == s.EventIDs[0] {
			if p := firstTerm(s.Locations); p != "" && first(s.CharacterNames) != "" {
				return first(s.CharacterNames) + " at " + p
			}
			return e.Summary
		}
	}
	return "Story scene"
}
func sequenceSummary(q types.StorySequence, _ []types.SemanticScene, events []types.SignificantStoryEvent) string {
	for _, e := range events {
		if contains(q.EventIDs, e.ID) {
			return e.Summary
		}
	}
	return "Story sequence"
}
func threadLabel(t types.NarrativeThread) string {
	if len(t.ObjectiveTerms) > 0 {
		return titleCase(t.ObjectiveTerms[0]) + " thread"
	}
	if len(t.CharacterNames) > 0 {
		return first(t.CharacterNames) + " thread"
	}
	return "Independent storyline"
}
func sumSalience(r types.SalienceBreakdown) float64 {
	return r.Base + r.StateChange + r.Movement + r.Decision + r.Discovery + r.TemporalChange + r.ThreadChange + r.LaterReferences + r.Contradiction - r.DescriptionOnly - r.LowConfidence
}
func mergeReasons(a, b types.SalienceBreakdown) types.SalienceBreakdown {
	return types.SalienceBreakdown{Base: math.Max(a.Base, b.Base), StateChange: math.Max(a.StateChange, b.StateChange), Movement: math.Max(a.Movement, b.Movement), Decision: math.Max(a.Decision, b.Decision), Discovery: math.Max(a.Discovery, b.Discovery), TemporalChange: math.Max(a.TemporalChange, b.TemporalChange), ThreadChange: math.Max(a.ThreadChange, b.ThreadChange), LaterReferences: math.Max(a.LaterReferences, b.LaterReferences), Contradiction: math.Max(a.Contradiction, b.Contradiction), DescriptionOnly: math.Min(a.DescriptionOnly, b.DescriptionOnly), LowConfidence: math.Max(a.LowConfidence, b.LowConfidence)}
}
func cloneTerms(v []types.EvidenceTerm) []types.EvidenceTerm {
	return append([]types.EvidenceTerm(nil), v...)
}
func termKey(t types.EvidenceTerm) string {
	return strings.ToLower(strings.TrimSpace(t.Label) + "\x00" + strings.TrimSpace(t.Text))
}
func termOverlap(a, b []types.EvidenceTerm) bool {
	set := map[string]bool{}
	for _, v := range a {
		set[termKey(v)] = true
	}
	for _, v := range b {
		if set[termKey(v)] {
			return true
		}
	}
	return false
}
func termTexts(v []types.EvidenceTerm) []string {
	r := []string{}
	for _, t := range v {
		if strings.TrimSpace(t.Text) != "" {
			r = appendUnique(r, strings.TrimSpace(t.Text))
		}
	}
	return r
}
func stringOverlap(a, b []string) bool {
	set := map[string]bool{}
	for _, v := range a {
		set[strings.ToLower(v)] = true
	}
	for _, v := range b {
		if set[strings.ToLower(v)] {
			return true
		}
	}
	return false
}
func sharedStrings(a, b []string) []string {
	r := []string{}
	for _, v := range a {
		if contains(b, v) {
			r = appendUnique(r, v)
		}
	}
	return r
}
func contains(v []string, x string) bool {
	for _, s := range v {
		if strings.EqualFold(s, x) {
			return true
		}
	}
	return false
}
func first(v []string) string {
	if len(v) > 0 {
		return v[0]
	}
	return ""
}
func firstTerm(v []types.EvidenceTerm) string {
	if len(v) > 0 {
		return strings.TrimSpace(v[0].Text)
	}
	return ""
}
func titleCase(v string) string {
	if v == "" {
		return v
	}
	return strings.ToUpper(v[:1]) + v[1:]
}
func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}
func lastEventOffset(s types.SemanticScene) int { return s.EndOffset }
func lastNarrative(v []types.SignificantStoryEvent) int {
	if len(v) == 0 {
		return 0
	}
	return v[len(v)-1].NarrativeOrder
}
func rangesOverlap(a, b, c, d int) bool { return a <= d && c <= b }

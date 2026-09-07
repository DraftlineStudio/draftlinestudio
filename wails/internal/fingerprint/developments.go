package fingerprint

// Narrative development synthesis (v5). Developments are story-level changes
// reconstructed from combinations of trusted frames, ledgers, and event
// identities at scene/chapter scope — no single sentence has to state a
// development outright. The synthesizer walks narrative units (scene-break
// delimited scenes, or the whole chapter when it has no breaks) carrying
// forward what is true or unresolved — open questions, goals, obstacles,
// firmly known facts — and emits a development whenever a unit's frames
// change that carried state. Every rule is deterministic, every summary is a
// template around verbatim source text, and every development records its
// supporting frames and evidence spans, before→after state, the goal or
// question it advances, and causal neighbors where the rules support them.

import (
	"fmt"
	"sort"
	"strings"

	"draftline/internal/types"
)

// ── carried story state ──────────────────────────────────────────────────────

type openQuestion struct {
	entity      string
	key         string
	words       map[string]bool
	detail      string
	devID       string
	frameID     string
	evidenceIDs []string
	unit        unitRef
}

type openGoal struct {
	entity  string
	key     string
	words   map[string]bool
	detail  string
	frameID string
	unit    unitRef
}

type developmentSynthesizer struct {
	frameByID map[string]types.NarrativeFrame
	// developments holds pointers so a rule can keep filling fields on an
	// emitted development after later emissions append to the list.
	developments []*types.NarrativeDevelopment
	byID         map[string]*types.NarrativeDevelopment

	openQuestions []*openQuestion
	openGoals     []*openGoal
	knownFacts    map[string]bool   // entity \x00 key
	injuryDevID   map[string]string // entity -> open injury obstacle
	lifeValue     map[string]string // entity \x00 scope -> dead|alive
	lifeFrameID   map[string]string // entity \x00 scope -> establishing frame
	countdownSeen map[string]countdownReading
	activeBefore  map[string]bool   // entity appeared in any earlier frame
	lastUnitDev   map[string]string // entity -> last development ID in the current unit
}

type countdownReading struct {
	value   float64
	phrase  string
	frameID string
}

func newDevelopmentSynthesizer(frames []types.NarrativeFrame) *developmentSynthesizer {
	byID := map[string]types.NarrativeFrame{}
	for _, frame := range frames {
		byID[frame.ID] = frame
	}
	return &developmentSynthesizer{
		frameByID: byID, byID: map[string]*types.NarrativeDevelopment{},
		knownFacts: map[string]bool{}, injuryDevID: map[string]string{},
		lifeValue: map[string]string{}, lifeFrameID: map[string]string{},
		countdownSeen: map[string]countdownReading{},
		activeBefore:  map[string]bool{}, lastUnitDev: map[string]string{},
	}
}

// emit creates a grounded development from one or more supporting frames.
func (s *developmentSynthesizer) emit(kind, summary string, unit unitRef, frames []types.NarrativeFrame, basis []string, confidence float64) *types.NarrativeDevelopment {
	if len(frames) == 0 {
		return nil
	}
	frameIDs := []string{}
	evidenceIDs := []string{}
	spans := []types.NarrativeEvidenceSpan{}
	entities := []types.NarrativeParticipant{}
	seenFrame := map[string]bool{}
	seenEvidence := map[string]bool{}
	seenEntity := map[string]bool{}
	order := frames[0].NarrativeOrder
	for _, frame := range frames {
		if seenFrame[frame.ID] {
			continue
		}
		seenFrame[frame.ID] = true
		frameIDs = append(frameIDs, frame.ID)
		for _, id := range frame.EvidenceIDs {
			if !seenEvidence[id] {
				seenEvidence[id] = true
				evidenceIDs = append(evidenceIDs, id)
			}
		}
		for _, span := range frame.EvidenceSpans {
			spans = append(spans, span)
		}
		for _, p := range frame.Participants {
			key := p.Role + "\x00" + p.EntityName
			if !seenEntity[key] {
				seenEntity[key] = true
				entities = append(entities, p)
			}
		}
		if frame.NarrativeOrder < order {
			order = frame.NarrativeOrder
		}
	}
	development := &types.NarrativeDevelopment{
		ID: stableID("development", append([]string{kind}, frameIDs...)...), Kind: kind,
		Summary: summary, Basis: basis, Entities: entities,
		FrameIDs: frameIDs, EvidenceIDs: evidenceIDs, EvidenceSpans: spans,
		ScopeID: frames[0].Scope.ID, ChapterIndex: unit.chapter, SceneIndex: unit.scene,
		NarrativeOrder: order, Confidence: confidence,
	}
	if existing, duplicate := s.byID[development.ID]; duplicate {
		return existing
	}
	s.byID[development.ID] = development
	s.developments = append(s.developments, development)
	for _, entity := range entities {
		if entity.Role == "subject" {
			s.lastUnitDev[entity.EntityName] = development.ID
		}
	}
	return development
}

func (s *developmentSynthesizer) linkPredecessor(development *types.NarrativeDevelopment, predecessorID string) {
	if development == nil || predecessorID == "" || predecessorID == development.ID {
		return
	}
	development.PredecessorID = predecessorID
	if predecessor, exists := s.byID[predecessorID]; exists && predecessor.SuccessorID == "" {
		predecessor.SuccessorID = development.ID
	}
}

// ── the synthesis walk ───────────────────────────────────────────────────────

func synthesizeDevelopments(
	book *types.BookData,
	frames []types.NarrativeFrame,
	ledgers []types.StateLedger,
	identities []types.NarrativeEventIdentity,
) []types.NarrativeDevelopment {
	s := newDevelopmentSynthesizer(frames)
	units := groupIntoUnits(book, frames)
	locator := newSceneLocator(book)
	for _, unit := range units {
		s.walkUnit(unit)
	}
	s.identityRules(identities, locator)
	s.ledgerRules(ledgers, locator)
	sort.SliceStable(s.developments, func(i, j int) bool {
		if s.developments[i].NarrativeOrder != s.developments[j].NarrativeOrder {
			return s.developments[i].NarrativeOrder < s.developments[j].NarrativeOrder
		}
		return s.developments[i].ID < s.developments[j].ID
	})
	result := make([]types.NarrativeDevelopment, len(s.developments))
	for index, development := range s.developments {
		result[index] = *development
	}
	return result
}

// learnedFact buffers a firm acquisition inside one unit so several small
// frames can aggregate into a single scene-level development. A fact may be
// tied to the open goal or open question its content overlaps; attributed
// marks claim-grade support (a character said it, narration did not).
type learnedFact struct {
	frame      types.NarrativeFrame
	goal       *openGoal
	question   *openQuestion
	attributed bool
}

func (s *developmentSynthesizer) walkUnit(current narrativeUnit) {
	unit := current.ref
	s.lastUnitDev = map[string]string{}
	learnedByEntity := map[string][]learnedFact{}
	entityOrder := []string{}
	for _, frame := range current.frames {
		subject := subjectName(frame)
		switch frame.Type {
		case types.FrameKnowledge, types.FrameBelief:
			if fact, isFact := s.knowledgeRules(unit, frame); isFact {
				if _, exists := learnedByEntity[subject]; !exists {
					entityOrder = append(entityOrder, subject)
				}
				learnedByEntity[subject] = append(learnedByEntity[subject], fact)
			}
		case types.FrameGoal:
			s.goalRule(unit, frame)
		case types.FrameDecision:
			s.decisionRule(unit, frame)
		case types.FrameObligation:
			s.obligationRule(unit, frame)
		case types.FrameInjury:
			s.injuryRule(unit, frame)
		case types.FrameLifeStatus:
			s.lifeStatusRule(unit, frame)
		case types.FramePossession, types.FrameAccess, types.FrameLocation:
			if fact, tied := s.goalAcquisitionRule(frame); tied {
				if _, exists := learnedByEntity[subject]; !exists {
					entityOrder = append(entityOrder, subject)
				}
				learnedByEntity[subject] = append(learnedByEntity[subject], fact)
			}
		case types.FrameEvent, types.FrameClaim:
			if fact, tied := s.stakeTieRule(frame); tied {
				if _, exists := learnedByEntity[subject]; !exists {
					entityOrder = append(entityOrder, subject)
				}
				learnedByEntity[subject] = append(learnedByEntity[subject], fact)
			}
		}
		s.countdownRule(unit, frame)
		for _, p := range frame.Participants {
			s.activeBefore[p.EntityName] = true
		}
	}
	for _, entity := range entityOrder {
		s.investigationRule(unit, entity, learnedByEntity[entity])
	}
}

// questionShaped reports whether a knowledge/belief frame states an open
// question rather than a firm fact: negation, mere belief, or speculation.
func questionShaped(frame types.NarrativeFrame) bool {
	if frame.Type == types.FrameBelief || frame.Epistemic == "belief" || frame.Epistemic == "speculation" {
		return true
	}
	return frame.Polarity == "negated" || frame.Value == "does_not_know" || frame.Value == "attempts_to_recall"
}

// firmFact reports whether a knowledge frame is narrator-grade acquisition.
func firmFact(frame types.NarrativeFrame) bool {
	return frame.Type == types.FrameKnowledge && frame.Epistemic == "narration" && frame.Polarity == "asserted"
}

func (s *developmentSynthesizer) knowledgeRules(unit unitRef, frame types.NarrativeFrame) (learnedFact, bool) {
	subject := subjectName(frame)
	if subject == "" || frame.Detail == "" {
		return learnedFact{}, false
	}
	key := qualifierKey(frame.Detail)
	words := contentWords(frame.Detail, subject)

	if questionShaped(frame) {
		s.questionRule(unit, frame, subject, key, words)
		return learnedFact{}, false
	}
	if !firmFact(frame) {
		return learnedFact{}, false
	}
	if s.knownFacts[subject+"\x00"+key] {
		return learnedFact{}, false // restating known fact — nothing new happened
	}

	// Resolution first: a firm fact that fully addresses an open question.
	for index, question := range s.openQuestions {
		if !covers(words, question.words) && key != question.key {
			continue
		}
		development := s.emit("mystery_resolved",
			fmt.Sprintf("Answered: %s now knows “%s”", subject, frame.Detail),
			unit, []types.NarrativeFrame{frame},
			[]string{"a firm acquisition fully addresses an open question"}, .85)
		if development != nil {
			development.Before = fmt.Sprintf("open question: “%s”", question.detail)
			development.After = fmt.Sprintf("known: “%s”", frame.Detail)
			development.Advances = fmt.Sprintf("question: “%s” (%s)", question.detail, question.entity)
			development.AdvancesFrameIDs = []string{question.frameID}
			s.linkPredecessor(development, question.devID)
		}
		s.openQuestions = append(s.openQuestions[:index], s.openQuestions[index+1:]...)
		s.knownFacts[subject+"\x00"+key] = true
		return learnedFact{}, false
	}

	// Narrowing: the fact meaningfully touches an open question without
	// closing it. The shared content words are recorded as the trace.
	var best *openQuestion
	bestShared := []string{}
	for _, question := range s.openQuestions {
		shared := overlapWords(words, question.words)
		if len(shared) >= 2 && len(shared) > len(bestShared) {
			best = question
			bestShared = shared
		}
	}
	if best != nil {
		development := s.emit("mystery_narrowed",
			fmt.Sprintf("The question of “%s” narrows: %s learns “%s”", best.detail, subject, frame.Detail),
			unit, []types.NarrativeFrame{frame},
			[]string{"a firm acquisition shares content with an open question: " + strings.Join(bestShared, ", ")}, .7)
		if development != nil {
			development.Advances = fmt.Sprintf("question: “%s” (%s)", best.detail, best.entity)
			development.AdvancesFrameIDs = []string{best.frameID}
			s.linkPredecessor(development, best.devID)
		}
		s.knownFacts[subject+"\x00"+key] = true
		return learnedFact{}, false
	}

	s.knownFacts[subject+"\x00"+key] = true
	// Not tied to a question — buffer for scene-level aggregation, tagged
	// with an open goal it advances when the content overlaps.
	fact := learnedFact{frame: frame}
	for _, goal := range s.openGoals {
		if goal.entity == subject && len(overlapWords(words, goal.words)) >= 2 {
			fact.goal = goal
			break
		}
	}
	// Only "learned"-type acquisitions read as new evidence appearing;
	// standing knowledge ("knew") is state, not a development.
	if frame.Value != "learned" && fact.goal == nil {
		return learnedFact{}, false
	}
	return fact, true
}

func (s *developmentSynthesizer) questionRule(unit unitRef, frame types.NarrativeFrame, subject, key string, words map[string]bool) {
	if s.knownFacts[subject+"\x00"+key] {
		return // already firmly known to this character — not a mystery
	}
	if len(words) < 2 {
		// "Hanlon didn't know." carries no content to ever resolve against —
		// pure anaphora opens nothing.
		return
	}
	for _, question := range s.openQuestions {
		if question.key == key {
			return // the same question stays one question
		}
		// Two frames cut from one sentence state one question, not two.
		for _, questionEvidence := range question.evidenceIDs {
			for _, frameEvidence := range frame.EvidenceIDs {
				if questionEvidence == frameEvidence {
					return
				}
			}
		}
	}
	for _, question := range s.openQuestions {
		// A belief that substantially overlaps an open question but states it
		// differently reframes that question.
		if frame.Type == types.FrameBelief {
			shared := overlapWords(words, question.words)
			if len(shared) >= 2 && !covers(words, question.words) {
				development := s.emit("mystery_reframed",
					fmt.Sprintf("The question shifts for %s: from “%s” to “%s”", subject, question.detail, frame.Detail),
					unit, []types.NarrativeFrame{frame},
					[]string{"a belief restates an open question with different content: " + strings.Join(shared, ", ")}, .65)
				if development != nil {
					development.Before = fmt.Sprintf("open question: “%s”", question.detail)
					development.After = fmt.Sprintf("open question: “%s”", frame.Detail)
					s.linkPredecessor(development, question.devID)
					question.key = key
					question.words = words
					question.detail = frame.Detail
					question.devID = development.ID
					question.frameID = frame.ID
				}
				return
			}
		}
	}
	template := "Unknown to %s: “%s”"
	basis := "a fact is negated or unknown for this character"
	if frame.Type == types.FrameBelief {
		template = "%s believes, without confirmation: “%s”"
		basis = "a belief or suspicion is stated without narrator confirmation"
	}
	development := s.emit("mystery_introduced",
		fmt.Sprintf(template, subject, frame.Detail),
		unit, []types.NarrativeFrame{frame}, []string{basis}, .7)
	if development == nil {
		return
	}
	development.Before = "not in question"
	development.After = fmt.Sprintf("open question: “%s”", frame.Detail)
	s.openQuestions = append(s.openQuestions, &openQuestion{
		entity: subject, key: key, words: words, detail: frame.Detail,
		devID: development.ID, frameID: frame.ID,
		evidenceIDs: clone(frame.EvidenceIDs), unit: unit,
	})
}

// goalAcquisitionRule ties a possession, access, or arrival to an open goal
// so obstacle-clearing acquisitions count as progress.
func (s *developmentSynthesizer) goalAcquisitionRule(frame types.NarrativeFrame) (learnedFact, bool) {
	subject := subjectName(frame)
	if subject == "" || frame.Detail == "" || frame.Polarity == "negated" {
		return learnedFact{}, false
	}
	if frame.Value == "relinquished" || frame.Value == "clear" || frame.Value == "revoked" {
		return learnedFact{}, false
	}
	words := contentWords(frame.Detail, subject)
	for _, goal := range s.openGoals {
		if goal.entity == subject && len(overlapWords(words, goal.words)) >= 2 {
			return learnedFact{frame: frame, goal: goal}, true
		}
	}
	return learnedFact{}, false
}

// stakeTieRule ties a witnessed event or an attributed claim to the active
// stake its content overlaps — the actor's open goal, or any open question.
// Untied events and claims stay out of the account entirely.
func (s *developmentSynthesizer) stakeTieRule(frame types.NarrativeFrame) (learnedFact, bool) {
	subject := subjectName(frame)
	if subject == "" || frame.Detail == "" || frame.Polarity == "negated" {
		return learnedFact{}, false
	}
	attributed := frame.Type == types.FrameClaim
	words := contentWords(frame.Detail, subject)
	for _, goal := range s.openGoals {
		if goal.entity == subject && len(overlapWords(words, goal.words)) >= 2 {
			return learnedFact{frame: frame, goal: goal, attributed: attributed}, true
		}
	}
	for _, question := range s.openQuestions {
		if len(overlapWords(words, question.words)) >= 2 {
			return learnedFact{frame: frame, question: question, attributed: attributed}, true
		}
	}
	return learnedFact{}, false
}

// investigationRule aggregates a unit's buffered acquisitions per entity:
// several individually small frames become one scene-level development.
func (s *developmentSynthesizer) investigationRule(unit unitRef, entity string, facts []learnedFact) {
	if entity == "" || len(facts) == 0 {
		return
	}
	// Stake-tied facts always count as progress; untied facts need scene-level
	// substance to rise above sentence scope.
	var goal *openGoal
	tied := []learnedFact{}
	questionTied := map[*openQuestion][]learnedFact{}
	questionOrder := []*openQuestion{}
	untied := []learnedFact{}
	for _, fact := range facts {
		switch {
		case fact.goal != nil:
			if goal == nil {
				goal = fact.goal
			}
			tied = append(tied, fact)
		case fact.question != nil:
			if _, exists := questionTied[fact.question]; !exists {
				questionOrder = append(questionOrder, fact.question)
			}
			questionTied[fact.question] = append(questionTied[fact.question], fact)
		default:
			untied = append(untied, fact)
		}
	}
	if goal != nil {
		frames := []types.NarrativeFrame{}
		details := []string{}
		confidence := .75
		basis := "acquisitions in this scene share content with the character's open goal"
		for _, fact := range tied {
			frames = append(frames, fact.frame)
			details = append(details, "“"+fact.frame.Detail+"”")
			if fact.attributed {
				confidence = .6
				basis += "; part of the support is an attributed claim, not narration"
			}
		}
		development := s.emit("investigation_progress",
			fmt.Sprintf("%s makes progress toward “%s”: %s", entity, goal.detail, strings.Join(details, "; ")),
			unit, frames,
			[]string{basis}, confidence)
		if development != nil {
			development.Advances = fmt.Sprintf("goal: “%s” (%s)", goal.detail, goal.entity)
			development.AdvancesFrameIDs = []string{goal.frameID}
		}
	}
	for _, question := range questionOrder {
		group := questionTied[question]
		frames := []types.NarrativeFrame{}
		details := []string{}
		confidence := .7
		basis := "events in this scene share content with an open question"
		for _, fact := range group {
			frames = append(frames, fact.frame)
			details = append(details, "“"+fact.frame.Detail+"”")
			if fact.attributed {
				confidence = .6
				basis = "an attributed claim speaks to an open question — said, not narrated"
			}
		}
		development := s.emit("investigation_progress",
			fmt.Sprintf("New material bears on “%s”: %s", question.detail, strings.Join(details, "; ")),
			unit, frames, []string{basis}, confidence)
		if development != nil {
			development.Advances = fmt.Sprintf("question: “%s” (%s)", question.detail, question.entity)
			development.AdvancesFrameIDs = []string{question.frameID}
			s.linkPredecessor(development, question.devID)
		}
	}
	// A lone but substantive firm acquisition still changed what the character
	// knows by scene close.
	if len(untied) == 1 {
		fact := untied[0]
		if len(contentWords(fact.frame.Detail, entity)) >= 4 {
			s.emit("investigation_progress",
				fmt.Sprintf("%s learns: “%s”", entity, fact.frame.Detail),
				unit, []types.NarrativeFrame{fact.frame},
				[]string{"a substantive firm acquisition changes what the character knows"}, .65)
		}
	}
	// Untied facts aggregate only across distinct sentences — two frames cut
	// from one sentence are one observation, not a pattern.
	distinctEvidence := map[string]bool{}
	for _, fact := range untied {
		for _, id := range fact.frame.EvidenceIDs {
			distinctEvidence[id] = true
		}
	}
	if len(untied) >= 2 && len(distinctEvidence) >= 2 {
		frames := []types.NarrativeFrame{}
		details := []string{}
		for _, fact := range untied {
			frames = append(frames, fact.frame)
			details = append(details, "“"+fact.frame.Detail+"”")
		}
		s.emit("investigation_progress",
			fmt.Sprintf("%s pieces together new information: %s", entity, strings.Join(details, "; ")),
			unit, frames,
			[]string{"several firm acquisitions by one character within one scene"}, .7)
	}
}

func (s *developmentSynthesizer) goalRule(unit unitRef, frame types.NarrativeFrame) {
	subject := subjectName(frame)
	if subject == "" || frame.Detail == "" || frame.Polarity == "negated" {
		return
	}
	key := qualifierKey(frame.Detail)
	words := contentWords(frame.Detail, subject)
	var latest *openGoal
	for _, goal := range s.openGoals {
		if goal.entity != subject {
			continue
		}
		if goal.key == key || len(overlapWords(words, goal.words)) >= 2 {
			return // the same pursuit restated is not a change of course
		}
		latest = goal
	}
	if latest == nil {
		development := s.emit("goal_established",
			fmt.Sprintf("%s sets a course: “%s”", subject, frame.Detail),
			unit, []types.NarrativeFrame{frame},
			[]string{"a canonical character's goal is first stated"}, .7)
		if development != nil {
			development.After = fmt.Sprintf("pursuing: “%s”", frame.Detail)
		}
	} else {
		development := s.emit("goal_change",
			fmt.Sprintf("%s changes course: now “%s”", subject, frame.Detail),
			unit, []types.NarrativeFrame{frame},
			[]string{"a new goal joins or replaces an earlier, unrelated one"}, .7)
		if development != nil {
			development.Before = fmt.Sprintf("pursuing: “%s”", latest.detail)
			development.After = fmt.Sprintf("pursuing: “%s”", frame.Detail)
			development.AdvancesFrameIDs = []string{latest.frameID}
		}
	}
	s.openGoals = append(s.openGoals, &openGoal{
		entity: subject, key: key, words: words, detail: frame.Detail,
		frameID: frame.ID, unit: unit,
	})
}

func (s *developmentSynthesizer) decisionRule(unit unitRef, frame types.NarrativeFrame) {
	subject := subjectName(frame)
	if subject == "" || frame.Detail == "" || frame.Epistemic != "narration" {
		return
	}
	template := "%s decides: “%s”"
	if frame.Polarity == "negated" {
		template = "%s refuses: “%s”"
	}
	predecessor := s.lastUnitDev[subject]
	development := s.emit("major_decision",
		fmt.Sprintf(template, subject, frame.Detail),
		unit, []types.NarrativeFrame{frame},
		[]string{"an explicit decision by a canonical character"}, .8)
	// A decision made in the same scene as a preceding development by the
	// same character reads as its consequence.
	s.linkPredecessor(development, predecessor)
}

func (s *developmentSynthesizer) obligationRule(unit unitRef, frame types.NarrativeFrame) {
	subject := subjectName(frame)
	if subject == "" || frame.Detail == "" || frame.Polarity == "negated" {
		return
	}
	development := s.emit("obstacle_introduced",
		fmt.Sprintf("%s is now bound: “%s”", subject, frame.Detail),
		unit, []types.NarrativeFrame{frame},
		[]string{"an obligation opens and constrains the character's options"}, .7)
	if development != nil {
		development.Before = "unbound"
		development.After = fmt.Sprintf("bound: “%s”", frame.Detail)
	}
}

func (s *developmentSynthesizer) injuryRule(unit unitRef, frame types.NarrativeFrame) {
	subject := subjectName(frame)
	if subject == "" || frame.Detail == "" {
		return
	}
	if frame.Polarity == "negated" {
		predecessorID := s.injuryDevID[subject]
		if predecessorID == "" {
			return // cannot claim recovery from an obstacle never established
		}
		development := s.emit("obstacle_overcome",
			fmt.Sprintf("%s recovers: “%s” no longer holds", subject, frame.Detail),
			unit, []types.NarrativeFrame{frame},
			[]string{"an established injury is negated"}, .7)
		if development != nil {
			development.Before = "injured"
			development.After = "recovered"
			s.linkPredecessor(development, predecessorID)
		}
		delete(s.injuryDevID, subject)
		return
	}
	development := s.emit("obstacle_introduced",
		fmt.Sprintf("%s is hurt: “%s”", subject, frame.Detail),
		unit, []types.NarrativeFrame{frame},
		[]string{"an injury constrains the character"}, .75)
	if development != nil {
		development.Before = "unharmed"
		development.After = fmt.Sprintf("injured: “%s”", frame.Detail)
		s.injuryDevID[subject] = development.ID
	}
}

func (s *developmentSynthesizer) lifeStatusRule(unit unitRef, frame types.NarrativeFrame) {
	subject := subjectName(frame)
	if subject == "" || frame.Value == "" || frame.Epistemic != "narration" {
		return
	}
	stateKey := subject + "\x00" + frame.Scope.ID
	previous := s.lifeValue[stateKey]
	switch frame.Value {
	case "dead":
		if previous == "dead" {
			return
		}
		if !s.activeBefore[subject] && previous == "" {
			// A character introduced already dead is backstory state, not a
			// change in the story's course.
			s.lifeValue[stateKey] = "dead"
			s.lifeFrameID[stateKey] = frame.ID
			return
		}
		development := s.emit("threat_escalation",
			fmt.Sprintf("%s is dead: “%s”", subject, frame.Detail),
			unit, []types.NarrativeFrame{frame},
			[]string{"narration establishes the death of an active character"}, .85)
		if development != nil {
			development.Before = "alive"
			development.After = "dead"
		}
	case "alive":
		// Alive after dead: a reversal within one scope, a cross-reality
		// revelation when the death belonged to another scope.
		if previous == "dead" {
			development := s.emit("revelation",
				fmt.Sprintf("%s is alive after being dead: “%s”", subject, frame.Detail),
				unit, []types.NarrativeFrame{frame},
				[]string{"life status flips within one narrative scope"}, .85)
			if development != nil {
				development.Before = "dead"
				development.After = "alive"
			}
		} else {
			// Deterministic scan: collect the subject's death records in
			// other scopes, sorted by state key.
			deadKeys := []string{}
			for key, value := range s.lifeValue {
				parts := strings.SplitN(key, "\x00", 2)
				if len(parts) == 2 && parts[0] == subject && parts[1] != frame.Scope.ID && value == "dead" {
					deadKeys = append(deadKeys, key)
				}
			}
			sort.Strings(deadKeys)
			if len(deadKeys) > 0 {
				supporting := []types.NarrativeFrame{frame}
				if dead, exists := s.frameByID[s.lifeFrameID[deadKeys[0]]]; exists {
					supporting = append(supporting, dead)
				}
				development := s.emit("revelation",
					fmt.Sprintf("Two realities disagree about %s: dead in one, alive in another", subject),
					unit, supporting,
					[]string{"life status diverges across narrative scopes"}, .75)
				if development != nil {
					development.Before = "dead (other scope)"
					development.After = "alive (this scope)"
					// The revelation happens where the later account appears.
					development.NarrativeOrder = frame.NarrativeOrder
				}
			}
		}
	}
	s.lifeValue[stateKey] = frame.Value
	s.lifeFrameID[stateKey] = frame.ID
}

// countdownRule reads deadline phrases from the frame's evidence quote; a
// count that shrinks within one scope is tightening time pressure.
func (s *developmentSynthesizer) countdownRule(unit unitRef, frame types.NarrativeFrame) {
	for _, span := range frame.EvidenceSpans {
		match := countdownRe.FindStringSubmatch(span.Quote)
		if match == nil {
			continue
		}
		value := countdownNumber(match[1])
		if value < 0 {
			continue
		}
		phrase := strings.TrimSpace(match[0])
		scopeKey := frame.Scope.ID + "\x00" + strings.TrimSuffix(strings.ToLower(match[2]), "s")
		previous, exists := s.countdownSeen[scopeKey]
		if exists && previous.frameID == frame.ID {
			continue
		}
		if exists && value < previous.value {
			supporting := []types.NarrativeFrame{frame}
			if earlier, found := s.frameByID[previous.frameID]; found {
				supporting = append([]types.NarrativeFrame{earlier}, supporting...)
			}
			development := s.emit("threat_escalation",
				fmt.Sprintf("Time pressure tightens: “%s” → “%s”", previous.phrase, phrase),
				unit, supporting,
				[]string{"a countdown shrinks within one narrative scope"}, .75)
			if development != nil {
				development.Before = fmt.Sprintf("“%s”", previous.phrase)
				development.After = fmt.Sprintf("“%s”", phrase)
				development.NarrativeOrder = frame.NarrativeOrder
				development.ChapterIndex = unit.chapter
				development.SceneIndex = unit.scene
			}
		}
		s.countdownSeen[scopeKey] = countdownReading{value: value, phrase: phrase, frameID: frame.ID}
	}
}

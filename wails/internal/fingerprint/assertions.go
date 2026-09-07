package fingerprint

import (
	"regexp"
	"sort"
	"strings"
	"unicode"

	"draftline/internal/types"
)

var (
	assertionNegationRe      = regexp.MustCompile(`(?i)\b(?:not|never|no longer|didn't|doesn't|isn't|wasn't|couldn't|wouldn't|cannot|can't)\b`)
	assertionUncertainRe     = regexp.MustCompile(`(?i)\b(?:maybe|perhaps|possibly|apparently|seemed|might|could have)\b`)
	beliefCueRe              = regexp.MustCompile(`(?i)\b(believed|believes|thought|thinks|suspected|suspects|theorized|guessed|assumed|inferred)\b`)
	claimCueRe               = regexp.MustCompile(`(?i)\b(said|says|told|claimed|claims|insisted|insists|reported|reports|testified|warned|explained|admitted|confessed|lied)\b`)
	discourseSaidRe          = regexp.MustCompile(`(?i)^\s*that said\s*,`)
	deceptionCueRe           = regexp.MustCompile(`(?i)\b(lied|deceived|fabricated|falsely claimed|pretended)\b`)
	corroborationCueRe       = regexp.MustCompile(`(?i)\b(confirmed|corroborated|proved|verified|demonstrated|established)\b`)
	narrativeCommitmentCueRe = regexp.MustCompile(`(?i)\b(promised|promises|swore|vowed|pledged|agreed\s+to|committed\s+(?:to|himself\s+to|herself\s+to|themselves\s+to))\b`)
	goalCueRe                = regexp.MustCompile(`(?i)\b(decided|decides|planned|plans|intended|intends|resolved|ordered|orders|determined\s+to|set out to|needed\s+to)\b`)
	conditionalGoalRe        = regexp.MustCompile(`(?i)\b(?:if|whether|in case)\b[^.!?]{0,100}\b(?:decided|decides|planned|plans|intended|intends|ordered|orders|determined\s+to|set out to|needed\s+to)\b`)
	relationshipCueRe        = regexp.MustCompile(`(?i)\b(forgave|trust(?:ed|s)?|distrust(?:ed|s)?|betrayed|allied|reconciled|befriended|married|divorced|abandoned|rejected|accepted)\b`)
	acquisitionCueRe         = regexp.MustCompile(`(?i)\b(obtained|acquired|received|picked up|took possession of|was given)\b`)
	custodyCueRe             = regexp.MustCompile(`(?i)\b(carried|carries|carrying)\b`)
	relinquishCueRe          = regexp.MustCompile(`(?i)\b(gave|handed|returned|lost|dropped|surrendered)\b`)
	nonPossessionObjectRe    = regexp.MustCompile(`(?i)^(?:it|his mind|her mind|their minds|any sense|the game|by \w+|control|interest|hope|patience|consciousness|track of|sight of|the ability|the chance)\b`)
	persistentChangeRe       = regexp.MustCompile(`(?i)\b(died|was killed|killed|was injured|was wounded|became|resigned|was fired|was promoted|disappeared|escaped|was captured|broke apart|broke down|burned down|exploded|was destroyed|closed permanently|opened permanently)\b`)
	attenuatedChangeRe       = regexp.MustCompile(`(?i)\b(?:nearly|almost|might have|could have|would have|about to|close to)\b[^.!?]{0,45}\b(?:died|killed|injured|wounded|destroyed|broke|burned|exploded|collapsed)\b`)
	strongDiscoveryRe        = regexp.MustCompile(`(?i)\b(discovered|uncovered|learned|learnt|realized|found out|determined|revealed)\b`)
	useCueRe                 = regexp.MustCompile(`(?i)\b(used|using|unlocked|activated|disabled|decoded|accessed|recognized|matched|recalled)\b`)
	completionCueRe          = regexp.MustCompile(`(?i)\b(fulfilled|kept (?:the |his |her |their )?promise|completed|finished|returned|delivered|handed|repaid|rescued|solved|answered)\b`)
	copulaPropositionRe      = regexp.MustCompile(`(?i)^\s*(.+?)\s+(is|are|was|were|has|have|had|will be|cannot be|isn't|wasn't)\s+(.+?)\s*$`)
	attributeReV2            = regexp.MustCompile(`(?i)\b(?:has|had|with)\s+(?:a\s+)?(blond|blonde|brown|black|red|gray|grey|white|blue|green|hazel)\s+(hair|eyes?)\b`)
	attributeChangeRe        = regexp.MustCompile(`(?i)\b(hair|eyes?|voice|name|rank|title|condition)\s+(?:had\s+)?(?:changed|turned|became|was now)\s+(?:to\s+)?([a-z][a-z -]{1,40})`)
	quoteRe                  = regexp.MustCompile(`["“]([^"”]+)["”]`)
	flashbackScopeRe         = regexp.MustCompile(`(?i)\b(?:flashback|years? earlier|months? earlier|days? earlier|back then)\b`)
	rememberedScopeRe        = regexp.MustCompile(`(?i)\b(?:remembered|recalled|memory of)\b`)
	visionScopeRe            = regexp.MustCompile(`(?i)\b(?:vision|vision of|saw in a vision)\b`)
	hypotheticalScopeRe      = regexp.MustCompile(`(?i)\b(?:if .* would|imagined|supposed that|hypothetically|might have been)\b`)
	storyWithinScopeRe       = regexp.MustCompile(`(?i)\b(?:in the story|the tale said|the letter read|the journal said|according to the diary)\b`)
	routineLogisticsRe       = regexp.MustCompile(`(?i)\b(?:car|truck|vehicle|coat|phone|keys?)\b[^.!?]{0,80}\b(?:parked|left|sitting|waiting|charging|stored)\b`)
)

// buildAssertions turns source-located evidence into conservative semantic
// assertions. Assertions are still not narrative fingerprints: mundane but
// continuity-useful statements are deliberately retained here.
func buildAssertions(records []types.EvidenceRecord, contexts []types.StoryContext, contextByEvidence map[string]string, points map[string]types.StoryTime) []types.StoryAssertion {
	byKey := map[string]int{}
	contextByID := map[string]types.StoryContext{}
	for _, context := range contexts {
		contextByID[context.ID] = context
	}
	result := make([]types.StoryAssertion, 0, len(records))
	for _, record := range records {
		context := contextByID[contextByEvidence[record.ID]]
		for _, assertion := range assertionsForRecord(record, context, points[record.ID]) {
			key := assertion.SemanticKey + "\x00" + assertion.EpistemicStatus + "\x00" + assertion.Scope.ID + "\x00" + assertion.Attribution.EntityID
			if index, exists := byKey[key]; exists {
				result[index].EvidenceIDs = appendUnique(result[index].EvidenceIDs, record.ID)
				if assertion.Confidence > result[index].Confidence {
					result[index].Confidence = assertion.Confidence
				}
				continue
			}
			assertion.ID = stableID("assertion", key)
			byKey[key] = len(result)
			result = append(result, assertion)
		}
	}
	sort.SliceStable(result, func(i, j int) bool {
		return evidenceOrder(result[i].EvidenceIDs, records) < evidenceOrder(result[j].EvidenceIDs, records)
	})
	return result
}

func assertionsForRecord(record types.EvidenceRecord, context types.StoryContext, point types.StoryTime) []types.StoryAssertion {
	scope := assertionScope(record, context)
	attribution := inferAttribution(record)
	primary := primaryAssertion(record, attribution, scope, point)
	result := []types.StoryAssertion{primary}
	for _, knowledge := range record.KnowledgeStates {
		for index, id := range knowledge.CharacterIDs {
			name := id
			if index < len(knowledge.CharacterNames) {
				name = knowledge.CharacterNames[index]
			}
			status := epistemicForKnowledge(knowledge.State)
			object := propositionAfterKnowledgeCue(record.Text, knowledge.Cue)
			if object == "" {
				object = strings.TrimSpace(record.Text)
			}
			item := types.StoryAssertion{
				EvidenceIDs: []string{record.ID}, Kind: "knowledge", Statement: knowledgeStatement(name, knowledge.State, object),
				SubjectID: id, Subject: name, Predicate: canonicalKnowledgePredicate(knowledge.State), Object: object,
				Posture: legacyPosture(status), Polarity: knowledgePolarity(knowledge.State), EpistemicStatus: status,
				Attribution: types.NarrativeAttribution{Kind: "narrator", Confidence: .72}, Scope: scope,
				Persistence: "persistent", Temporal: point, ContextID: scope.ID, Confidence: minFloat(record.Confidence, knowledge.Confidence), Status: "active",
			}
			item.SemanticKey = semanticAssertionKey(item)
			if item.SemanticKey != primary.SemanticKey {
				result = append(result, item)
			}
		}
	}
	return result
}

func primaryAssertion(record types.EvidenceRecord, attribution types.NarrativeAttribution, scope types.NarrativeRealityScope, point types.StoryTime) types.StoryAssertion {
	subjectID, subject := firstCharacter(record)
	statement := strings.TrimSpace(record.AuthorText)
	if statement == "" {
		statement = strings.TrimSpace(record.Text)
	}
	kind, predicate, object, persistence, change := classifyAssertionMeaning(record, subjectID, subject)
	if change != nil && persistentChangeRe.MatchString(record.Text) {
		if stateID, stateSubject := persistentChangeSubject(record); stateSubject != "" {
			subjectID, subject = stateID, stateSubject
			change.EntityID, change.EntityName = stateID, stateSubject
		}
	}
	if kind == "occurrence" {
		if propositionSubject, propositionPredicate, propositionObject := parseProposition(statement); propositionPredicate != "" {
			if !strings.EqualFold(propositionSubject, subject) {
				subjectID = ""
			}
			subject, predicate, object = propositionSubject, propositionPredicate, propositionObject
			kind, persistence = "state", "conditional"
		}
	}
	status := inferEpistemicStatus(record.Text, attribution, scope)
	if attribution.Kind == "character" {
		if proposition := attributedProposition(record.Text, attribution); proposition != "" {
			propSubject, propPredicate, propObject := parseProposition(proposition)
			kind = "claim"
			statement = attribution.EntityName + " " + attribution.Cue + " that " + proposition
			if propPredicate != "" {
				subjectID, subject = "", propSubject
				predicate, object = propPredicate, propObject
			} else {
				predicate, object = "claims", proposition
			}
			persistence = "conditional"
		}
	}
	if beliefCueRe.MatchString(record.Text) {
		status = "character_belief"
		kind = "knowledge"
		believerID, believer := firstCharacter(record)
		if proposition := clauseAfterMatch(record.Text, beliefCueRe); proposition != "" {
			statement = strings.TrimSpace(believer + " believes " + proposition)
			if propositionSubject, propositionPredicate, propositionObject := parseProposition(proposition); propositionPredicate != "" {
				subjectID, subject, predicate, object = "", propositionSubject, propositionPredicate, propositionObject
			} else {
				object = proposition
				predicate = "believes"
			}
		}
		persistence = "persistent"
		attribution = types.NarrativeAttribution{Kind: "character", EntityID: believerID, EntityName: believer, Cue: "believes", Confidence: .82}
	}
	polarity := "positive"
	if assertionNegationRe.MatchString(statement) {
		polarity = "negative"
	}
	if assertionUncertainRe.MatchString(statement) {
		polarity = "uncertain"
	}
	item := types.StoryAssertion{
		EvidenceIDs: []string{record.ID}, Kind: kind, Statement: statement, SubjectID: subjectID, Subject: subject,
		Predicate: predicate, Object: object, Posture: legacyPosture(status), Polarity: polarity,
		EpistemicStatus: status, Attribution: attribution, Scope: scope, StateChange: change,
		Persistence: persistence, Temporal: point, ContextID: scope.ID, Confidence: record.Confidence, Status: "active",
	}
	item.SemanticKey = semanticAssertionKey(item)
	return item
}

func classifyAssertionMeaning(record types.EvidenceRecord, subjectID, subject string) (kind, predicate, object, persistence string, change *types.NarrativeStateChange) {
	text := strings.TrimSpace(record.Text)
	switch {
	case narrativeCommitmentCueRe.MatchString(text):
		predicate = canonicalCue(narrativeCommitmentCueRe.FindString(text))
		object = clauseAfterMatch(text, narrativeCommitmentCueRe)
		return "commitment", predicate, object, "persistent", &types.NarrativeStateChange{EntityID: subjectID, EntityName: subject, StateKind: "commitment", New: object, Operation: "begin"}
	case goalCueRe.MatchString(text) && !conditionalGoalRe.MatchString(text):
		predicate = canonicalCue(goalCueRe.FindString(text))
		object = clauseAfterMatch(text, goalCueRe)
		return "goal", predicate, object, "persistent", &types.NarrativeStateChange{EntityID: subjectID, EntityName: subject, StateKind: "goal", New: object, Operation: "begin"}
	case relationshipCueRe.MatchString(text):
		predicate = canonicalCue(relationshipCueRe.FindString(text))
		object = relationObject(record, subject)
		return "relationship", predicate, object, "persistent", &types.NarrativeStateChange{EntityID: subjectID, EntityName: subject, StateKind: "relationship", New: strings.TrimSpace(predicate + " " + object), Operation: "change"}
	case attributeChangeRe.MatchString(text):
		match := attributeChangeRe.FindStringSubmatch(text)
		value := strings.TrimSpace(match[2])
		return "state", "changes_" + strings.ToLower(match[1]), value, "persistent", &types.NarrativeStateChange{EntityID: subjectID, EntityName: subject, StateKind: strings.ToLower(match[1]), New: value, Operation: "change"}
	case attributeReV2.MatchString(text):
		match := attributeReV2.FindStringSubmatch(text)
		value := strings.ToLower(match[1] + " " + match[2])
		return "state", "has_attribute", value, "persistent", &types.NarrativeStateChange{EntityID: subjectID, EntityName: subject, StateKind: "attribute", New: value, Operation: "establish"}
	case acquisitionCueRe.MatchString(text) && concretePossessionChange(record, acquisitionCueRe, "acquire"):
		object = clauseAfterMatch(text, acquisitionCueRe)
		entityID, entityName := possessionActorBeforeCue(record, acquisitionCueRe, subjectID, subject)
		return "state", "acquires", object, "conditional", &types.NarrativeStateChange{EntityID: entityID, EntityName: entityName, StateKind: "possession", New: object, Operation: "acquire"}
	case custodyCueRe.MatchString(text) && concretePossessionChange(record, custodyCueRe, "carry"):
		object = clauseAfterMatch(text, custodyCueRe)
		entityID, entityName := possessionActorBeforeCue(record, custodyCueRe, subjectID, subject)
		return "state", "carries", object, "conditional", &types.NarrativeStateChange{EntityID: entityID, EntityName: entityName, StateKind: "possession", New: object, Operation: "carry"}
	case relinquishCueRe.MatchString(text) && concretePossessionChange(record, relinquishCueRe, "relinquish"):
		object = clauseAfterMatch(text, relinquishCueRe)
		entityID, entityName := possessionActorBeforeCue(record, relinquishCueRe, subjectID, subject)
		return "state", "relinquishes", object, "conditional", &types.NarrativeStateChange{EntityID: entityID, EntityName: entityName, StateKind: "possession", Previous: object, New: "not in custody", Operation: "relinquish"}
	case persistentChangeRe.MatchString(text) && !attenuatedChangeRe.MatchString(text):
		predicate = canonicalCue(persistentChangeRe.FindString(text))
		object = clauseAfterMatch(text, persistentChangeRe)
		return "state", predicate, object, "persistent", &types.NarrativeStateChange{EntityID: subjectID, EntityName: subject, StateKind: persistentStateKind(predicate), New: persistentStateValue(predicate, object), Operation: "change"}
	case strongDiscoveryRe.MatchString(text):
		predicate = canonicalCue(strongDiscoveryRe.FindString(text))
		return "knowledge", predicate, clauseAfterMatch(text, strongDiscoveryRe), "persistent", nil
	default:
		predicate = strings.TrimSpace(record.Action)
		if predicate == "" {
			predicate = record.EvidenceType
		}
		_, object = assertionObject(record, subject)
		if object == "" {
			object = actionComplement(text, predicate)
		}
		return "occurrence", canonicalCue(predicate), object, occurrencePersistence(record), nil
	}
}

func concretePossessionChange(record types.EvidenceRecord, cue *regexp.Regexp, operation string) bool {
	match := cue.FindStringIndex(record.Text)
	if match == nil {
		return false
	}
	payload := strings.Join(strings.Fields(cleanClause(record.Text[match[1]:])), " ")
	if payload == "" || nonPossessionObjectRe.MatchString(payload) {
		return false
	}
	matchedCue := normalizeSemantic(record.Text[match[0]:match[1]])
	if operation == "relinquish" {
		switch matchedCue {
		case "gave", "handed":
			if regexp.MustCompile(`(?i)\b(?:gave|handed)\b[^.!?]{1,100}\bto\b`).MatchString(record.Text) {
				return true
			}
			afterCue := strings.TrimSpace(record.Text[match[1]:])
			for _, name := range record.CharacterNames {
				if strings.HasPrefix(strings.ToLower(afterCue), strings.ToLower(name)+" ") {
					return true
				}
			}
			return false
		case "returned":
			return len(semanticTerms(payload)) >= 2
		case "lost":
			return concretePossessionTerm(record) && len(semanticTerms(payload)) >= 1
		}
	}
	if operation == "carry" {
		if record.EvidenceType != "state" && record.EvidenceType != "interaction" {
			return false
		}
		_, name := possessionActorBeforeCue(record, cue, "", "")
		return name != "" && len(semanticTerms(payload)) >= 1
	}
	return len(semanticTerms(payload)) >= 1
}

func concretePossessionTerm(record types.EvidenceRecord) bool {
	for _, term := range record.NamedEntities {
		switch term.Label {
		case "PRODUCT", "WORK_OF_ART", "OBJECT":
			return true
		}
	}
	return regexp.MustCompile(`(?i)\blost\s+(?:a|an|the|his|her|their|my|our)\s+[a-z][a-z'-]*`).MatchString(record.Text)
}

func possessionActorBeforeCue(record types.EvidenceRecord, cue *regexp.Regexp, fallbackID, fallbackName string) (string, string) {
	match := cue.FindStringIndex(record.Text)
	if match == nil {
		return fallbackID, fallbackName
	}
	prefix := strings.ToLower(record.Text[:match[0]])
	bestID, bestName, bestPosition := "", "", -1
	for index, name := range record.CharacterNames {
		positions := allFoldIndexes(prefix, name)
		if len(positions) == 0 || positions[len(positions)-1] <= bestPosition {
			continue
		}
		bestPosition, bestName = positions[len(positions)-1], name
		if index < len(record.CharacterIDs) {
			bestID = record.CharacterIDs[index]
		}
	}
	if bestName != "" && match[0]-bestPosition <= 24 {
		return bestID, bestName
	}
	return fallbackID, fallbackName
}

func inferAttribution(record types.EvidenceRecord) types.NarrativeAttribution {
	text := record.Text
	match := claimCueRe.FindStringIndex(text)
	if match != nil && discourseSaidRe.MatchString(text) && normalizeSemantic(text[match[0]:match[1]]) == "said" {
		match = nil
	}
	if match == nil {
		if hasDialogueBoundary(text) || beginsSecondPersonAddress(text) {
			return types.NarrativeAttribution{Kind: "unknown", Cue: "unresolved dialogue attribution", Confidence: .35}
		}
		return types.NarrativeAttribution{Kind: "narrator", Confidence: .72}
	}
	cue := strings.ToLower(text[match[0]:match[1]])
	bestID, bestName, bestDistance := "", "", len(text)+1
	for index, name := range record.CharacterNames {
		for _, position := range allFoldIndexes(text, name) {
			distance := absInt(position - match[0])
			if distance < bestDistance {
				bestDistance, bestName = distance, name
				if index < len(record.CharacterIDs) {
					bestID = record.CharacterIDs[index]
				}
			}
		}
	}
	if bestName == "" && len(record.CharacterNames) > 0 {
		bestID, bestName = firstCharacter(record)
	}
	if bestName == "" {
		return types.NarrativeAttribution{Kind: "unknown", Cue: cue, Confidence: .4}
	}
	return types.NarrativeAttribution{Kind: "character", EntityID: bestID, EntityName: bestName, Cue: cue, Confidence: .82}
}

func inferEpistemicStatus(text string, attribution types.NarrativeAttribution, scope types.NarrativeRealityScope) string {
	switch {
	case deceptionCueRe.MatchString(text):
		return "deliberate_deception"
	case beliefCueRe.MatchString(text):
		return "character_belief"
	case attribution.Kind == "character":
		return "attributed_claim"
	case attribution.Kind == "unknown":
		return "attributed_claim"
	case assertionUncertainRe.MatchString(text):
		return "uncertain_interpretation"
	case corroborationCueRe.MatchString(text):
		return "externally_corroborated_fact"
	case scope.Kind == "dream" || scope.Kind == "vision" || scope.Kind == "hypothetical":
		return "observed_occurrence"
	default:
		return "world_state_fact"
	}
}

func assertionScope(record types.EvidenceRecord, context types.StoryContext) types.NarrativeRealityScope {
	kind, label, confidence := "current", "Current narrative reality", .9
	contextID := context.ID
	if contextID == "" {
		contextID = "context-primary"
	}
	switch context.Kind {
	case "past", "memory":
		kind, label, confidence = "flashback", nonEmpty(context.Label, "Earlier story time"), context.Confidence
	case "dream", "simulation", "unknown":
		kind, label, confidence = context.Kind, nonEmpty(context.Label, strings.Title(context.Kind)), context.Confidence
	}
	switch {
	case hypotheticalScopeRe.MatchString(record.Text):
		kind, label, confidence = "hypothetical", "Hypothetical", .86
	case storyWithinScopeRe.MatchString(record.Text):
		kind, label, confidence = "story_within_story", "Story within the story", .72
	case visionScopeRe.MatchString(record.Text):
		kind, label, confidence = "vision", "Vision", .78
	case dreamCueRe.MatchString(record.Text):
		kind, label, confidence = "dream", "Dream", .82
	case simulationCueRe.MatchString(record.Text):
		kind, label, confidence = "simulation", "Simulation or alternate reality", .82
	case rememberedScopeRe.MatchString(record.Text):
		kind, label, confidence = "remembered", "Remembered event", .76
	case flashbackScopeRe.MatchString(record.Text):
		kind, label, confidence = "flashback", "Earlier story time", .74
	}
	if kind != "current" {
		if context.ID == "" || (context.Kind == "primary" && kind != "current") {
			contextID = stableID("scope", record.ChapterID, kind)
		}
	}
	return types.NarrativeRealityScope{ID: contextID, Kind: kind, Label: label, ParentID: "context-primary", EvidenceIDs: []string{record.ID}, Confidence: confidence}
}

func attributedProposition(text string, attribution types.NarrativeAttribution) string {
	if quoted := quoteRe.FindStringSubmatch(text); len(quoted) == 2 {
		return cleanClause(quoted[1])
	}
	match := claimCueRe.FindStringIndex(text)
	if match == nil {
		return ""
	}
	return cleanClause(text[match[1]:])
}

func parseProposition(value string) (subject, predicate, object string) {
	match := copulaPropositionRe.FindStringSubmatch(cleanClause(value))
	if len(match) != 4 {
		return "", "", ""
	}
	return cleanClause(match[1]), canonicalCue(match[2]), cleanClause(match[3])
}

func semanticAssertionKey(item types.StoryAssertion) string {
	state := ""
	if item.StateChange != nil {
		state = strings.Join([]string{item.StateChange.StateKind, item.StateChange.Previous, item.StateChange.New, item.StateChange.Operation}, "|")
	}
	parts := []string{normalizeSemantic(item.Subject), normalizeSemantic(item.Predicate), normalizeSemantic(item.Object), item.Polarity, normalizeSemantic(state)}
	if item.Kind == "occurrence" || (item.StateChange == nil && len(semanticTerms(item.Object)) < 2) {
		parts = append(parts, normalizeSemantic(item.Statement))
	}
	return strings.Join(parts, "|")
}

func persistentChangeSubject(record types.EvidenceRecord) (string, string) {
	match := persistentChangeRe.FindStringIndex(record.Text)
	if match == nil {
		return "", ""
	}
	prefix := record.Text[:match[0]]
	bestID, bestName, bestPosition := "", "", -1
	for index, name := range record.CharacterNames {
		positions := allFoldIndexes(prefix, name)
		if len(positions) == 0 || positions[len(positions)-1] <= bestPosition {
			continue
		}
		bestPosition, bestName = positions[len(positions)-1], name
		if index < len(record.CharacterIDs) {
			bestID = record.CharacterIDs[index]
		}
	}
	if bestName != "" {
		return bestID, bestName
	}
	for _, separator := range []string{"\n", ".", ";", ":", ","} {
		if index := strings.LastIndex(prefix, separator); index >= 0 {
			prefix = prefix[index+len(separator):]
		}
	}
	words := strings.Fields(cleanClause(prefix))
	if len(words) > 6 {
		words = words[len(words)-6:]
	}
	for len(words) > 0 {
		last := strings.ToLower(strings.Trim(words[len(words)-1], "'’"))
		if last == "who" || last == "who'd" || last == "that" || last == "had" || last == "was" || last == "were" || last == "and" || last == "shot" {
			words = words[:len(words)-1]
			continue
		}
		break
	}
	return "", strings.Join(words, " ")
}

func hasDialogueBoundary(text string) bool {
	return strings.ContainsAny(text, "\"\u201c\u201d")
}

func beginsSecondPersonAddress(text string) bool {
	words := strings.Fields(normalizeSemantic(text))
	return len(words) > 0 && words[0] == "you"
}

func assertionObject(record types.EvidenceRecord, subject string) (string, string) {
	for _, term := range record.NamedEntities {
		if strings.EqualFold(term.Text, subject) {
			continue
		}
		return "", term.Text
	}
	if len(record.CharacterIDs) > 1 {
		name := record.CharacterIDs[1]
		if len(record.CharacterNames) > 1 {
			name = record.CharacterNames[1]
		}
		return record.CharacterIDs[1], name
	}
	return "", ""
}

func firstCharacter(record types.EvidenceRecord) (string, string) {
	if len(record.CharacterIDs) == 0 {
		return "", ""
	}
	name := record.CharacterIDs[0]
	if len(record.CharacterNames) > 0 {
		name = record.CharacterNames[0]
	}
	return record.CharacterIDs[0], name
}

func clauseAfterMatch(text string, expression *regexp.Regexp) string {
	match := expression.FindStringIndex(text)
	if match == nil {
		return ""
	}
	return cleanClause(text[match[1]:])
}

func actionComplement(text, action string) string {
	if action == "" {
		return ""
	}
	index := strings.Index(strings.ToLower(text), strings.ToLower(action))
	if index < 0 {
		return ""
	}
	return cleanClause(text[index+len(action):])
}

func propositionAfterKnowledgeCue(text, cue string) string {
	if cue == "" {
		return ""
	}
	index := strings.Index(strings.ToLower(text), strings.ToLower(cue))
	if index < 0 {
		return ""
	}
	return cleanClause(text[index+len(cue):])
}

func cleanClause(value string) string {
	value = strings.TrimSpace(value)
	value = strings.TrimLeft(value, " ,:;-–—")
	lower := strings.ToLower(value)
	for _, prefix := range []string{"that ", "to ", "how ", "why ", "whether "} {
		if strings.HasPrefix(lower, prefix) {
			value = strings.TrimSpace(value[len(prefix):])
			break
		}
	}
	return strings.TrimSpace(strings.Trim(value, " \t\r\n.,;:!?\"“”"))
}

func canonicalCue(value string) string {
	value = normalizeSemantic(value)
	switch value {
	case "promises", "promised", "swore", "vowed", "pledged", "committed":
		return "promises"
	case "decides", "decided", "resolved":
		return "decides"
	case "plans", "planned", "intended", "intends":
		return "plans"
	case "discovers", "discovered", "uncovered", "found out", "realized", "learned", "learnt", "determined":
		return "discovers"
	case "says", "said", "told", "claimed", "claims", "insisted", "insists", "reported", "reports", "testified", "warned", "explained", "admitted", "confessed", "lied":
		return "claims"
	case "is", "are", "was", "were", "will be", "isn t", "wasn t", "cannot be":
		return "is"
	default:
		return value
	}
}

func canonicalKnowledgePredicate(value string) string {
	switch value {
	case "learned":
		return "learns"
	case "shared":
		return "shares_knowledge"
	case "withheld":
		return "withholds_knowledge"
	default:
		return value
	}
}

func epistemicForKnowledge(value string) string {
	switch value {
	case "believes":
		return "character_belief"
	case "suspects":
		return "character_inference"
	case "does_not_believe", "does_not_suspect", "does_not_know", "attempts_to_recall":
		return "uncertain_interpretation"
	default:
		return "observed_occurrence"
	}
}

func knowledgeStatement(name, state, object string) string {
	verb := strings.ReplaceAll(state, "_", " ")
	return strings.TrimSpace(name + " " + verb + " " + object)
}

func knowledgePolarity(state string) string {
	if strings.HasPrefix(state, "does_not_") || state == "withheld" {
		return "negative"
	}
	return "positive"
}

func legacyPosture(status string) string {
	switch status {
	case "attributed_claim":
		return "claim"
	case "character_belief":
		return "belief"
	case "character_inference":
		return "suspicion"
	case "deliberate_deception":
		return "lie"
	default:
		return "fact"
	}
}

func relationObject(record types.EvidenceRecord, subject string) string {
	for _, name := range record.CharacterNames {
		if !strings.EqualFold(name, subject) {
			return name
		}
	}
	return clauseAfterMatch(record.Text, relationshipCueRe)
}

func persistentStateKind(predicate string) string {
	switch {
	case strings.Contains(predicate, "died") || strings.Contains(predicate, "killed"):
		return "life_status"
	case strings.Contains(predicate, "injur") || strings.Contains(predicate, "wound"):
		return "physical_condition"
	case strings.Contains(predicate, "captur") || strings.Contains(predicate, "escaped") || strings.Contains(predicate, "disappeared"):
		return "freedom_or_presence"
	case strings.Contains(predicate, "destroy") || strings.Contains(predicate, "broke") || strings.Contains(predicate, "burn") || strings.Contains(predicate, "explod"):
		return "world_object_condition"
	default:
		return "character_or_world_state"
	}
}

func persistentStateValue(predicate, object string) string {
	switch {
	case strings.Contains(predicate, "died") || strings.Contains(predicate, "killed"):
		return "dead"
	case strings.Contains(predicate, "injur") || strings.Contains(predicate, "wound"):
		return "injured"
	case strings.Contains(predicate, "captur"):
		return "captured"
	case strings.Contains(predicate, "escaped"):
		return "escaped"
	case strings.Contains(predicate, "disappeared"):
		return "missing"
	case strings.Contains(predicate, "destroy") || strings.Contains(predicate, "broke") || strings.Contains(predicate, "burn") || strings.Contains(predicate, "explod"):
		return "destroyed or inoperable"
	default:
		return nonEmpty(object, predicate)
	}
}

func occurrencePersistence(record types.EvidenceRecord) string {
	if record.EvidenceType == "state" || record.EvidenceType == "knowledge_state" {
		return "conditional"
	}
	return "transient"
}

func normalizeSemantic(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var b strings.Builder
	space := false
	for _, r := range value {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
			space = false
		} else if !space {
			b.WriteByte(' ')
			space = true
		}
	}
	return strings.Join(strings.Fields(b.String()), " ")
}

func allFoldIndexes(text, value string) []int {
	text, value = strings.ToLower(text), strings.ToLower(value)
	if value == "" {
		return nil
	}
	result := []int{}
	for offset := 0; offset < len(text); {
		index := strings.Index(text[offset:], value)
		if index < 0 {
			break
		}
		result = append(result, offset+index)
		offset += index + len(value)
	}
	return result
}

func evidenceOrder(ids []string, records []types.EvidenceRecord) int {
	for index, record := range records {
		for _, id := range ids {
			if record.ID == id {
				return index
			}
		}
	}
	return int(^uint(0) >> 1)
}

func minFloat(left, right float64) float64 {
	if left < right {
		return left
	}
	return right
}

func absInt(value int) int {
	if value < 0 {
		return -value
	}
	return value
}

func nonEmpty(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

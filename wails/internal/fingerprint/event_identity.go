package fingerprint

// Event identity resolution (v5). Frames are linked into likely same events
// from STRUCTURED properties only — frame class, subject, outcome/action
// key — never because two sentences merely share vocabulary. Every account
// is preserved; conflicting details become conflicting property values and
// mark the identity conflicted rather than being flattened.

import (
	"sort"
	"strings"

	"draftline/internal/types"
)

var anchorStopWords = map[string]bool{
	"the": true, "a": true, "an": true, "to": true, "at": true, "in": true,
	"on": true, "of": true, "and": true, "then": true, "his": true,
	"her": true, "their": true, "its": true, "into": true, "onto": true,
	"back": true, "up": true, "down": true, "out": true, "over": true,
	"toward": true, "towards": true, "forward": true, "away": true,
}

// Routine motion and posture verbs describe actions a character performs
// constantly; they can never anchor a same-event identity, no matter what
// follows them. Retellable events anchor on consequential verbs (found,
// killed, opened, broke…).
var routineActionVerbs = map[string]bool{
	"walked": true, "turned": true, "looked": true, "stood": true,
	"sat": true, "moved": true, "stepped": true, "ran": true,
	"nodded": true, "smiled": true, "stopped": true, "leaned": true,
	"waited": true, "watched": true, "listened": true, "breathed": true,
}

// contentHead returns the first n content words of a normalized phrase, or
// "" when fewer than n exist — an event without enough identifying content
// abstains from identity resolution entirely.
func contentHead(text string, n int) string {
	words := []string{}
	for _, word := range strings.Fields(qualifierKey(text)) {
		if anchorStopWords[word] {
			continue
		}
		words = append(words, word)
		if len(words) == n {
			break
		}
	}
	if len(words) < n {
		return ""
	}
	return strings.Join(words, " ")
}

type identityKey struct {
	class   string
	subject string
	anchor  string
}

// identityAnchor produces the structured anchor a frame must share to be
// considered the same underlying event as another frame of its class.
func identityAnchor(frame types.NarrativeFrame) (identityKey, bool) {
	subject := subjectName(frame)
	if subject == "" {
		return identityKey{}, false
	}
	switch frame.Type {
	case types.FrameLifeStatus:
		// Two accounts of the same person's death (or survival) describe one
		// underlying event, whatever scope tells each version.
		return identityKey{class: frame.Type, subject: subject, anchor: frame.Value}, true
	case types.FrameInjury:
		return identityKey{class: frame.Type, subject: subject, anchor: firstWords(qualifierKey(frame.Detail), 2)}, true
	case types.FrameTransfer:
		recipient := participantNamed(frame, "recipient")
		item := qualifierKey(participantNamed(frame, "item"))
		return identityKey{class: frame.Type, subject: subject, anchor: item + "→" + recipient}, true
	case types.FrameEvent:
		// Same actor + same action/object head. The head is the first two
		// CONTENT words of the typed detail — function words like "to" and
		// "the" identify nothing and must never cause a merge, and routine
		// motion verbs (wherever they sit in the head, "walked" or "started
		// walking") abstain from identity entirely.
		head := contentHead(frame.Detail, 2)
		if head == "" {
			return identityKey{}, false
		}
		for _, word := range strings.Fields(head) {
			if routineActionVerbs[word] || routineActionVerbs[strings.TrimSuffix(word, "ing")+"ed"] {
				return identityKey{}, false
			}
		}
		return identityKey{class: frame.Type, subject: subject, anchor: head}, true
	}
	return identityKey{}, false
}

func resolveEventIdentities(frames []types.NarrativeFrame) []types.NarrativeEventIdentity {
	groups := map[identityKey][]types.NarrativeFrame{}
	order := []identityKey{}
	for _, frame := range frames {
		key, ok := identityAnchor(frame)
		if !ok {
			continue
		}
		if _, exists := groups[key]; !exists {
			order = append(order, key)
		}
		groups[key] = append(groups[key], frame)
	}

	identities := []types.NarrativeEventIdentity{}
	for _, key := range order {
		members := groups[key]
		if len(members) < 2 {
			continue // an identity needs at least two accounts
		}
		identity := types.NarrativeEventIdentity{
			ID:           stableID("identity", key.class, key.subject, key.anchor),
			EventClass:   key.class,
			Participants: []types.NarrativeParticipant{participant("subject", key.subject)},
			Temporal:     members[0].Temporal,
			Status:       "consistent",
			Confidence:   .8,
		}
		scopeSeen := map[string]bool{}
		accountValues := map[string][]types.NarrativeFrame{}
		accountOrder := []string{}
		for _, member := range members {
			identity.FrameIDs = append(identity.FrameIDs, member.ID)
			identity.EvidenceIDs = append(identity.EvidenceIDs, member.EvidenceIDs...)
			if !scopeSeen[member.Scope.ID] {
				scopeSeen[member.Scope.ID] = true
				identity.ScopeIDs = append(identity.ScopeIDs, member.Scope.ID)
			}
			value := member.Detail
			if value == "" {
				value = member.Value
			}
			if _, exists := accountValues[value]; !exists {
				accountOrder = append(accountOrder, value)
			}
			accountValues[value] = append(accountValues[value], member)
		}
		property := types.EventIdentityProperty{Name: "account"}
		for _, value := range accountOrder {
			holders := accountValues[value]
			propertyValue := types.EventPropertyValue{Value: value, Epistemic: holders[0].Epistemic}
			for _, holder := range holders {
				propertyValue.FrameIDs = append(propertyValue.FrameIDs, holder.ID)
				propertyValue.EvidenceIDs = append(propertyValue.EvidenceIDs, holder.EvidenceIDs...)
			}
			property.Values = append(property.Values, propertyValue)
		}
		identity.Properties = []types.EventIdentityProperty{property}
		if len(property.Values) > 1 {
			identity.Status = "conflicted"
		}
		identities = append(identities, identity)
	}
	sort.SliceStable(identities, func(i, j int) bool { return identities[i].ID < identities[j].ID })
	return identities
}

package indexing

import (
	"math"
	"strings"

	"draftline/internal/entityresolution"
)

var entitySuffixKinds = map[string]string{
	"street": "place", "road": "place", "avenue": "place", "drive": "place",
	"lobby": "place", "mountains": "place", "tower": "place", "towers": "place",
	"ranch": "place", "city": "place", "river": "place", "falls": "place", "wacker": "place",
	"bank": "organization", "tribune": "organization", "press": "organization",
	"publishing": "organization", "works": "organization", "network": "organization",
	"department": "organization", "command": "organization", "fleetcom": "organization",
	"liaison": "organization", "telephone": "organization", "bar": "organization",
	"maps": "object", "accord": "object", "series": "object", "system": "object",
	"client": "object", "windows": "object", "atvs": "object", "leds": "object", "tvs": "object",
	"nodes": "object", "strike": "object",
}

var obviousNonActorWords = map[string]bool{
	"boom": true, "bang": true, "route": true, "streets": true, "footsteps": true,
	"system": true, "code": true, "windows": true, "transactions": true,
	"language": true, "paper": true, "smoke": true, "fire": true, "stone": true,
	"hours": true, "days": true, "words": true, "plan": true, "plans": true,
	"response": true, "guide": true, "official": true, "presidential": true, "access": true,
}

// ClassifyResolvedEntities separates identity resolution from Codex admission.
// The resolver answers "which mentions belong together"; this pass answers
// "what sort of story entity is this, and is the evidence strong enough to
// display automatically?" Rejected candidates remain in analysis for audit.
func ClassifyResolvedEntities(entities []entityresolution.Entity, mentions []entityresolution.Mention) {
	mentionByID := make(map[string]entityresolution.Mention, len(mentions))
	for _, mention := range mentions {
		mentionByID[mention.ID] = mention
	}

	for i := range entities {
		entity := &entities[i]
		personEvidence, nonPersonEvidence, strongEvidence := 0, 0, 0
		for _, id := range entity.MentionIDs {
			mention := mentionByID[id]
			if mention.PersonEvidence {
				personEvidence++
			}
			if mention.NonPersonEvidence {
				nonPersonEvidence++
			}
			if mention.StrongPersonEvidence {
				strongEvidence++
			}
		}

		kind := lexicalEntityKind(entity.Canonical)
		if kind == "" && (len(entity.Titles) > 0 || personEvidence > nonPersonEvidence) {
			kind = "person"
		}
		if kind == "" {
			kind = "unknown"
		}

		score := math.Min(0.35, float64(len(entity.MentionIDs))*0.04)
		if len(strings.Fields(entity.Canonical)) >= 2 {
			score += 0.10
		}
		if personEvidence > 0 {
			score += 0.25
		}
		if strongEvidence > 0 {
			score += 0.20
		}
		if len(entity.Titles) > 0 {
			score += 0.20
		}
		if nonPersonEvidence > personEvidence {
			score -= 0.30
		}

		status := "review"
		if kind == "person" && score >= 0.55 && (len(entity.MentionIDs) >= 2 || len(entity.Titles) > 0) {
			status = "accepted"
		}
		lowerCanonical := strings.ToLower(strings.TrimSpace(entity.Canonical))
		if kind == "object" && personEvidence == 0 && (obviousNonActorWords[lowerCanonical] || score < 0.25) {
			status = "rejected"
		}
		if score < 0 {
			score = 0
		}
		if score > 1 {
			score = 1
		}

		entity.Kind = kind
		entity.DetectionStatus = status
		entity.DetectionScore = score
	}
}

func lexicalEntityKind(name string) string {
	words := strings.Fields(strings.ToLower(name))
	if len(words) == 0 {
		return "unknown"
	}
	if kind := entitySuffixKinds[words[len(words)-1]]; kind != "" {
		return kind
	}
	if obviousNonActorWords[strings.Join(words, " ")] {
		return "object"
	}
	return ""
}

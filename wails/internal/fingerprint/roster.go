package fingerprint

// Codex identity for participants: the accepted people of the character
// codex, keyed by case-folded name, so a frame's participant can carry the
// entity ID the Planner and the codex use. Names are the durable key; the
// IDs are positional and churn across re-indexing, which is why the roster
// itself matches by name and the ID is stamped on afterwards.

import (
	"strings"

	"draftline/internal/types"
)

// entityID returns the codex entity ID of an accepted person by exact,
// case-folded canonical name or alias; "" when the name is not an accepted
// person.
func (r *rosterMatcher) entityID(name string) string {
	if r == nil {
		return ""
	}
	return r.ids[strings.ToLower(strings.TrimSpace(name))]
}

// AcceptedPeopleIDs collects the codex's accepted people: manual characters,
// and auto-detected ones the writer accepted. Rejected characters and
// non-person entities never get an ID here. Canonical names win over
// aliases when two people share a name form.
func AcceptedPeopleIDs(book *types.BookData) map[string]string {
	ids := map[string]string{}
	aliases := map[string]string{}
	for _, character := range book.StoryBible.Characters {
		if character.ID == "" || character.Name == "" || character.DetectionStatus == "rejected" {
			continue
		}
		if character.EntityKind != "" && character.EntityKind != "person" {
			continue
		}
		if character.IsAutoDetected && character.DetectionStatus != "accepted" {
			continue
		}
		key := strings.ToLower(strings.TrimSpace(character.Name))
		if _, taken := ids[key]; !taken {
			ids[key] = character.ID
		}
		for _, alias := range character.Aliases {
			key := strings.ToLower(strings.TrimSpace(alias))
			if key == "" {
				continue
			}
			if _, taken := aliases[key]; !taken {
				aliases[key] = character.ID
			}
		}
	}
	for key, id := range aliases {
		if _, taken := ids[key]; !taken {
			ids[key] = id
		}
	}
	return ids
}

// bindParticipantIDs fills every participant's EntityID from the codex's
// accepted people, by exact case-folded name. Names that are not an
// accepted person keep an empty ID; nothing else about the frame changes.
func bindParticipantIDs(roster *rosterMatcher, frames []types.NarrativeFrame) {
	for f := range frames {
		for p := range frames[f].Participants {
			if frames[f].Participants[p].EntityID == "" {
				frames[f].Participants[p].EntityID = roster.entityID(frames[f].Participants[p].EntityName)
			}
		}
	}
}

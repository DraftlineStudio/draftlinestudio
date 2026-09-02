package fingerprint

import (
	"strings"

	"draftline/internal/types"
)

var profileVocabulary = map[string][]string{
	"police-procedural":           {"detective", "interrogation", "suspect", "officer", "precinct", "warrant", "evidence", "internal affairs"},
	"speculative-science-fiction": {"simulation", "spacecraft", "reactor", "universe", "quantum", "computer", "artificial intelligence", "wormhole"},
	"military-science-fiction":    {"captain", "fleet", "torpedo", "railgun", "battle cruiser", "command", "warship"},
	"mystery-thriller":            {"mystery", "conspiracy", "investigate", "discovered", "secret", "missing", "murder", "tunnel"},
	"romance":                     {"kissed", "romance", "lover", "attraction", "wedding", "heartbreak"},
}

func deriveProfiles(records []types.EvidenceRecord, authored []string) []types.StoryProfile {
	result := []types.StoryProfile{}
	seen := map[string]bool{}
	for _, id := range authored {
		id = strings.TrimSpace(strings.ToLower(id))
		if id != "" && !seen[id] {
			result = append(result, types.StoryProfile{ID: id, Label: profileLabel(id), Confidence: 1, Source: "author"})
			seen[id] = true
		}
	}
	corpus := strings.Builder{}
	for _, record := range records {
		corpus.WriteString(" ")
		corpus.WriteString(strings.ToLower(record.Text))
	}
	text := corpus.String()
	for id, terms := range profileVocabulary {
		if seen[id] {
			continue
		}
		hits := 0
		for _, term := range terms {
			hits += strings.Count(text, term)
		}
		if hits >= 2 {
			confidence := .55 + float64(hits)*.04
			if confidence > .92 {
				confidence = .92
			}
			result = append(result, types.StoryProfile{ID: id, Label: profileLabel(id), Confidence: confidence, Source: "inferred"})
		}
	}
	return result
}

func profileLabel(id string) string {
	words := strings.Split(id, "-")
	for index := range words {
		if words[index] != "" {
			words[index] = strings.ToUpper(words[index][:1]) + words[index][1:]
		}
	}
	return strings.Join(words, " ")
}

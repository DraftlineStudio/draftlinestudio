package entityresolution

import "strings"

func buildReferenceAnchors(mentions []Mention) map[string][]*Mention {
	anchors := map[string][]*Mention{}
	for i := range mentions {
		mention := &mentions[i]
		if len(mention.Tokens) < 2 {
			continue
		}
		for _, token := range []string{mention.Tokens[0], mention.Tokens[len(mention.Tokens)-1]} {
			key := strings.ToLower(token)
			anchors[key] = append(anchors[key], mention)
		}
	}
	return anchors
}

func (reg *registry) referenceCandidates(m *Mention, candidates []*regEntity) []*regEntity {
	if len(m.Tokens) != 1 {
		return candidates
	}
	anchors := reg.referenceAnchors[strings.ToLower(m.Tokens[0])]
	names := map[string]bool{}
	sentenceNames, chapterNames := map[string]bool{}, map[string]bool{}
	groups := TitleGroups(m.Titles)
	for _, anchor := range anchors {
		if !GroupsCompatible(groups, TitleGroups(anchor.Titles)) {
			continue
		}
		name := strings.ToLower(strings.Join(anchor.Tokens, " "))
		names[name] = true
		if anchor.Chapter == m.Chapter {
			chapterNames[name] = true
			if m.SentenceID != "" && anchor.SentenceID == m.SentenceID {
				sentenceNames[name] = true
			}
		}
	}
	if len(names) <= 1 && len(candidates) <= 1 {
		if candidates == nil {
			return []*regEntity{}
		}
		return candidates
	}
	if len(names) > 1 {
		local := sentenceNames
		if len(local) == 0 {
			local = chapterNames
		}
		if len(local) != 1 {
			return nil
		}
		filtered := []*regEntity{}
		for _, candidate := range candidates {
			for name := range local {
				if candidate.variantSet[name] {
					filtered = append(filtered, candidate)
					break
				}
			}
		}
		candidates = filtered
	}
	if len(candidates) <= 1 {
		if len(candidates) == 0 {
			return nil
		}
		return candidates
	}
	for _, sameSentence := range []bool{true, false} {
		local := []*regEntity{}
		for _, candidate := range candidates {
			found := false
			for _, mention := range candidate.mentions {
				if mention.Chapter == m.Chapter && (!sameSentence || (m.SentenceID != "" && mention.SentenceID == m.SentenceID)) {
					found = true
					break
				}
			}
			if found {
				local = append(local, candidate)
			}
		}
		if len(local) == 1 {
			return local
		}
		if len(local) > 1 {
			return nil
		}
	}
	return nil
}

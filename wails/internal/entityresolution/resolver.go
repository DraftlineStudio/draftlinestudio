package entityresolution

import (
	"fmt"
	"sort"
	"strings"
)

// Phase 2 of the character pipeline: entity resolution.
//
// Mentions resolve in document order against a registry of entities:
//
//  1. Exact full-name match (after honorific stripping) → existing entity.
//  2. Subset/superset token match — a single-token mention matches an entity
//     containing that component; a multi-token mention absorbs existing
//     entities whose names are components of it (longest form becomes
//     canonical, shorter forms become aliases). Recency disambiguates when
//     several entities match.
//  3. Nickname match (same surname, known nickname pair).
//  4. Likely-typo match on the name head.
//  5. Otherwise a new entity is created.
//
// SeparatedPairs (from manual splits) are always respected: a mention never
// joins an entity it is separated from, and separated entities never merge.

// Resolver performs entity resolution on mentions.
type Resolver struct {
	Honorifics     map[string]bool // Configurable honorific set
	SeparatedPairs []SeparatedPair // Pairs that should not be merged
}

// NewResolver creates a new resolver with default settings.
func NewResolver() *Resolver {
	return &Resolver{
		Honorifics:     DefaultHonorifics,
		SeparatedPairs: nil,
	}
}

// regEntity is a registry entry being built during resolution.
type regEntity struct {
	ordinal    int
	mentions   []*Mention
	variants   [][]string      // distinct token lists seen for this entity
	variantSet map[string]bool // lowercase joined variants
	titles     []string
	groups     map[string]bool // title groups (see TitleGroups)
	lastSeen   int             // sequence number of the most recent mention
}

// registry holds resolution state.
type registry struct {
	resolver  *Resolver
	entities  []*regEntity
	separated map[string]bool // "mentionID|mentionID" pairs that must not merge
	// separatedMentions holds every mention ID involved in any separation —
	// the fast path skips pair scans entirely for uninvolved mentions.
	separatedMentions map[string]bool
	nextID            int
}

// ResolveEntities resolves mentions into entities in document order.
func (r *Resolver) ResolveEntities(mentions []Mention) *ResolvedEntities {
	if len(mentions) == 0 {
		return &ResolvedEntities{
			Entities:       []Entity{},
			SeparatedPairs: r.SeparatedPairs,
		}
	}

	for i := range mentions {
		r.parseMention(&mentions[i])
	}

	// Document order: by chapter, then offset. Stable for determinism.
	order := make([]int, len(mentions))
	for i := range order {
		order[i] = i
	}
	sort.SliceStable(order, func(a, b int) bool {
		ma, mb := &mentions[order[a]], &mentions[order[b]]
		if ma.Chapter != mb.Chapter {
			return ma.Chapter < mb.Chapter
		}
		return ma.CharOffset < mb.CharOffset
	})

	separatedMentions := make(map[string]bool, len(r.SeparatedPairs)*2)
	for _, pair := range r.SeparatedPairs {
		separatedMentions[pair.MentionID1] = true
		separatedMentions[pair.MentionID2] = true
	}

	reg := &registry{
		resolver:          r,
		separated:         r.buildSeparationIndex(),
		separatedMentions: separatedMentions,
		nextID:            1,
	}

	for seq, idx := range order {
		m := &mentions[idx]
		if len(m.Tokens) == 0 {
			continue
		}
		reg.resolve(m, seq)
	}

	return &ResolvedEntities{
		Entities:       reg.toEntities(),
		SeparatedPairs: r.SeparatedPairs,
	}
}

// parseMention extracts normalized head, titles, and tokens from a mention.
func (r *Resolver) parseMention(m *Mention) {
	head, titles := StripHonorifics(m.Text, r.Honorifics)
	m.NormalizedHead = head
	m.Titles = titles
	m.Tokens = TokenizeName(head)
}

// buildSeparationIndex creates a set of separated mention ID pairs.
func (r *Resolver) buildSeparationIndex() map[string]bool {
	index := make(map[string]bool)
	for _, pair := range r.SeparatedPairs {
		index[pair.MentionID1+"|"+pair.MentionID2] = true
		index[pair.MentionID2+"|"+pair.MentionID1] = true
	}
	return index
}

// resolve assigns a mention to a registry entity, merging or creating
// entities as required.
func (reg *registry) resolve(m *Mention, seq int) {
	candidates := reg.match(m)

	if len(candidates) == 0 {
		e := &regEntity{
			ordinal:    reg.nextID,
			variantSet: map[string]bool{},
			groups:     map[string]bool{},
			lastSeen:   seq,
		}
		reg.nextID++
		reg.entities = append(reg.entities, e)
		reg.assign(e, m, seq)
		return
	}

	// A multi-token mention absorbs every candidate whose name is a
	// component of it ("Daniel" + "Hanlon" merge into "Daniel Hanlon"),
	// unless the candidates are separated or title-incompatible.
	target := candidates[0]
	if len(m.Tokens) >= 2 {
		for _, c := range candidates[1:] {
			if reg.entitiesSeparated(target, c) || !GroupsCompatible(target.groups, c.groups) {
				continue
			}
			reg.merge(target, c)
		}
	}
	reg.assign(target, m, seq)
}

// match returns candidate entities for a mention, best tier first and most
// recently seen first within a tier. Separated and title-incompatible
// entities are excluded (Dr. Chen never matches Staff Sgt. Chen).
func (reg *registry) match(m *Mention) []*regEntity {
	full := strings.ToLower(strings.Join(m.Tokens, " "))
	mentionGroups := TitleGroups(m.Titles)

	var tiers [4][]*regEntity
	for _, e := range reg.entities {
		if reg.mentionSeparated(m, e) {
			continue
		}
		if !GroupsCompatible(mentionGroups, e.groups) {
			continue
		}
		switch {
		case e.variantSet[full]:
			tiers[0] = append(tiers[0], e)
		case reg.subsetMatch(m.Tokens, e):
			tiers[1] = append(tiers[1], e)
		case reg.nicknameMatch(m.Tokens, e):
			tiers[2] = append(tiers[2], e)
		case reg.typoMatch(m.Tokens, e):
			tiers[3] = append(tiers[3], e)
		}
	}

	for _, tier := range tiers {
		if len(tier) > 0 {
			// Recency disambiguation: most recently mentioned first.
			sort.SliceStable(tier, func(a, b int) bool {
				return tier[a].lastSeen > tier[b].lastSeen
			})
			return tier
		}
	}
	return nil
}

// subsetMatch reports whether mention tokens and any entity variant are in a
// strict subset/superset relation (component match).
func (reg *registry) subsetMatch(tokens []string, e *regEntity) bool {
	for _, v := range e.variants {
		if isSubset(tokens, v) || isSubset(v, tokens) {
			return true
		}
	}
	return false
}

// nicknameMatch reports whether the mention shares a surname with a variant
// and the first names are a known nickname pair.
func (reg *registry) nicknameMatch(tokens []string, e *regEntity) bool {
	if len(tokens) < 2 {
		return false
	}
	for _, v := range e.variants {
		if len(v) < 2 {
			continue
		}
		if strings.EqualFold(tokens[len(tokens)-1], v[len(v)-1]) && IsNickname(tokens[0], v[0]) {
			return true
		}
	}
	return false
}

// typoMatch reports whether the mention head is a likely typo of a variant head.
func (reg *registry) typoMatch(tokens []string, e *regEntity) bool {
	head := tokens[len(tokens)-1]
	for _, v := range e.variants {
		vHead := v[len(v)-1]
		if !strings.EqualFold(head, vHead) && IsLikelyTypo(head, vHead) {
			return true
		}
	}
	return false
}

// isSubset checks if tokens A are a strict subset of tokens B (case-insensitive).
func isSubset(tokensA, tokensB []string) bool {
	if len(tokensA) >= len(tokensB) {
		return false
	}
	for _, tA := range tokensA {
		found := false
		for _, tB := range tokensB {
			if strings.EqualFold(tA, tB) {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

// mentionSeparated reports whether a mention is separated from any mention
// already assigned to the entity.
func (reg *registry) mentionSeparated(m *Mention, e *regEntity) bool {
	if !reg.separatedMentions[m.ID] {
		return false
	}
	for _, existing := range e.mentions {
		if reg.separated[m.ID+"|"+existing.ID] {
			return true
		}
	}
	return false
}

// entitiesSeparated reports whether any mention pair across two entities is separated.
func (reg *registry) entitiesSeparated(a, b *regEntity) bool {
	if len(reg.separated) == 0 {
		return false
	}
	for _, ma := range a.mentions {
		if !reg.separatedMentions[ma.ID] {
			continue
		}
		for _, mb := range b.mentions {
			if reg.separated[ma.ID+"|"+mb.ID] {
				return true
			}
		}
	}
	return false
}

// assign adds a mention to an entity, recording its name variant and titles.
// Single-token REFERENCE mentions ("Marcus" resolving to Marcus Webb) are
// counted but never become match variants — otherwise every other
// "Marcus <Surname>" would merge into this entity through the bare token.
func (reg *registry) assign(e *regEntity, m *Mention, seq int) {
	e.mentions = append(e.mentions, m)
	e.lastSeen = seq

	if len(e.variants) == 0 || len(m.Tokens) >= 2 {
		full := strings.ToLower(strings.Join(m.Tokens, " "))
		if !e.variantSet[full] {
			e.variantSet[full] = true
			e.variants = append(e.variants, m.Tokens)
		}
	}

	for _, t := range m.Titles {
		found := false
		normalized := strings.TrimSuffix(t, ".")
		for _, existing := range e.titles {
			if strings.EqualFold(strings.TrimSuffix(existing, "."), normalized) {
				found = true
				break
			}
		}
		if !found {
			e.titles = append(e.titles, t)
		}
	}
	for g := range TitleGroups(m.Titles) {
		e.groups[g] = true
	}
}

// merge absorbs entity b into entity a and removes b from the registry.
func (reg *registry) merge(a, b *regEntity) {
	a.mentions = append(a.mentions, b.mentions...)
	for g := range b.groups {
		a.groups[g] = true
	}
	for _, v := range b.variants {
		full := strings.ToLower(strings.Join(v, " "))
		if !a.variantSet[full] {
			a.variantSet[full] = true
			a.variants = append(a.variants, v)
		}
	}
	for _, t := range b.titles {
		found := false
		normalized := strings.TrimSuffix(t, ".")
		for _, existing := range a.titles {
			if strings.EqualFold(strings.TrimSuffix(existing, "."), normalized) {
				found = true
				break
			}
		}
		if !found {
			a.titles = append(a.titles, t)
		}
	}
	if b.lastSeen > a.lastSeen {
		a.lastSeen = b.lastSeen
	}

	for i, e := range reg.entities {
		if e == b {
			reg.entities = append(reg.entities[:i], reg.entities[i+1:]...)
			break
		}
	}
}

// toEntities converts registry entries to the final Entity format,
// sorted by mention count (most mentioned first) with deterministic IDs.
func (reg *registry) toEntities() []Entity {
	entities := make([]Entity, 0, len(reg.entities))

	for _, e := range reg.entities {
		if len(e.mentions) == 0 {
			continue
		}

		canonical := pickCanonical(e)
		aliases := buildAliases(e, canonical)

		mentionIDs := make([]string, len(e.mentions))
		for i, m := range e.mentions {
			mentionIDs[i] = m.ID
		}

		entities = append(entities, Entity{
			ID:         fmt.Sprintf("entity-%d", e.ordinal),
			Canonical:  canonical,
			Aliases:    aliases,
			MentionIDs: mentionIDs,
			Confidence: calculateConfidence(e),
			Titles:     e.titles,
		})
	}

	sort.SliceStable(entities, func(i, j int) bool {
		return len(entities[i].MentionIDs) > len(entities[j].MentionIDs)
	})

	return entities
}

// pickCanonical selects the most complete name variant, preserving the
// original casing of the first mention that used it.
func pickCanonical(e *regEntity) string {
	best := e.variants[0]
	for _, v := range e.variants[1:] {
		if len(v) > len(best) {
			best = v
		} else if len(v) == len(best) && len(strings.Join(v, " ")) > len(strings.Join(best, " ")) {
			best = v
		}
	}
	return strings.Join(best, " ")
}

// buildAliases constructs the alias list: all other name variants and the
// raw mention texts (which naturally include titled forms like "Dr. Chen").
// Nothing is fabricated — every alias occurred in the manuscript.
func buildAliases(e *regEntity, canonical string) []string {
	seen := map[string]bool{strings.ToLower(canonical): true}
	aliases := []string{}

	add := func(s string) {
		lower := strings.ToLower(s)
		if s != "" && !seen[lower] {
			seen[lower] = true
			aliases = append(aliases, s)
		}
	}

	for _, v := range e.variants {
		add(strings.Join(v, " "))
	}
	for _, m := range e.mentions {
		add(m.Text)
	}

	return aliases
}

// calculateConfidence estimates confidence from cluster characteristics:
// more mentions raise it, high variant diversity lowers it.
func calculateConfidence(e *regEntity) float64 {
	mentionBonus := float64(len(e.mentions)) * 0.02
	if mentionBonus > 0.2 {
		mentionBonus = 0.2
	}
	variantPenalty := float64(len(e.variants)-1) * 0.05
	if variantPenalty > 0.3 {
		variantPenalty = 0.3
	}
	confidence := 0.8 + mentionBonus - variantPenalty
	if confidence < 0.5 {
		confidence = 0.5
	}
	if confidence > 1.0 {
		confidence = 1.0
	}
	return confidence
}

// SplitEntity splits an entity by moving specified mentions to a new entity.
// Returns the modified entity list and adds SeparatedPairs to prevent re-merging.
func (r *Resolver) SplitEntity(entities []Entity, entityID string, mentionIDs []string) ([]Entity, error) {
	entityIdx := -1
	for i, e := range entities {
		if e.ID == entityID {
			entityIdx = i
			break
		}
	}
	if entityIdx == -1 {
		return entities, fmt.Errorf("entity %s not found", entityID)
	}

	entity := &entities[entityIdx]

	mentionSet := make(map[string]bool)
	for _, id := range entity.MentionIDs {
		mentionSet[id] = true
	}
	for _, id := range mentionIDs {
		if !mentionSet[id] {
			return entities, fmt.Errorf("mention %s does not belong to entity %s", id, entityID)
		}
	}

	if len(mentionIDs) >= len(entity.MentionIDs) {
		return entities, fmt.Errorf("cannot split all mentions from entity")
	}

	splitSet := make(map[string]bool)
	for _, id := range mentionIDs {
		splitSet[id] = true
	}

	remainingIDs := []string{}
	for _, id := range entity.MentionIDs {
		if !splitSet[id] {
			remainingIDs = append(remainingIDs, id)
		}
	}
	entity.MentionIDs = remainingIDs

	// The canonical name and aliases are recomputed by the caller, which has
	// access to the mention records.
	newEntity := Entity{
		ID:         fmt.Sprintf("%s-split-%d", entityID, len(entities)),
		Canonical:  "",
		Aliases:    []string{},
		MentionIDs: mentionIDs,
		Confidence: 0.7,
		Titles:     []string{},
	}

	// Separate each split mention from each remaining mention so
	// re-resolution keeps them apart.
	for _, splitID := range mentionIDs {
		for _, remainID := range remainingIDs {
			r.SeparatedPairs = append(r.SeparatedPairs, SeparatedPair{
				MentionID1: splitID,
				MentionID2: remainID,
				Reason:     "manual split",
			})
		}
	}

	entities = append(entities, newEntity)
	return entities, nil
}

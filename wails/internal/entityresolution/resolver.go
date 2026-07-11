package entityresolution

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

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

// ResolveEntities clusters mentions into entities using rule-based matching.
// The algorithm:
// 1. Parse all mentions, strip honorifics, tokenize names
// 2. Initial clustering by exact normalized head match (respecting separations)
// 3. Merge clusters using subset/superset, nickname, and typo rules
// 4. Respect SeparatedPairs - don't merge those
// 5. Pick canonical name (most complete form)
func (r *Resolver) ResolveEntities(mentions []Mention) *ResolvedEntities {
	if len(mentions) == 0 {
		return &ResolvedEntities{
			Entities:       []Entity{},
			SeparatedPairs: r.SeparatedPairs,
		}
	}

	// Step 1: Parse and normalize all mentions
	for i := range mentions {
		r.parseMention(&mentions[i])
	}

	// Step 2: Build separation index for O(1) lookup
	separated := r.buildSeparationIndex()

	// Step 3: Initial clustering by exact head match (respecting separations)
	clusters := r.initialClusters(mentions, separated)

	// Step 4: Merge clusters iteratively until no more merges
	clusters = r.mergeClusters(clusters, separated)

	// Step 5: Convert clusters to entities
	entities := r.clustersToEntities(clusters)

	return &ResolvedEntities{
		Entities:       entities,
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
		// Store both orderings for O(1) lookup
		key1 := pair.MentionID1 + "|" + pair.MentionID2
		key2 := pair.MentionID2 + "|" + pair.MentionID1
		index[key1] = true
		index[key2] = true
	}
	return index
}

// areSeparated checks if any mention in cluster A is separated from any in B.
func (r *Resolver) areSeparated(clusterA, clusterB *Cluster, separated map[string]bool) bool {
	for _, mA := range clusterA.Mentions {
		for _, mB := range clusterB.Mentions {
			key := mA.ID + "|" + mB.ID
			if separated[key] {
				return true
			}
		}
	}
	return false
}

// isMentionSeparatedFromCluster checks if a mention is separated from any mention in a cluster.
func (r *Resolver) isMentionSeparatedFromCluster(m *Mention, c *Cluster, separated map[string]bool) bool {
	for _, existing := range c.Mentions {
		key := m.ID + "|" + existing.ID
		if separated[key] {
			return true
		}
	}
	return false
}

// initialClusters groups mentions by exact normalized head match (case-insensitive).
// Respects separated pairs - a mention won't be added to a cluster if it's separated
// from any existing mention in that cluster.
func (r *Resolver) initialClusters(mentions []Mention, separated map[string]bool) []*Cluster {
	// headMap can have multiple clusters per head if separations exist
	headMap := make(map[string][]*Cluster) // normalized head (lowercase) -> list of clusters

	for i := range mentions {
		m := &mentions[i]
		head := GetNameHead(m.Tokens)
		if head == "" {
			continue
		}

		headLower := strings.ToLower(head)

		// Find a cluster we can add to (not separated)
		added := false
		if clusters, exists := headMap[headLower]; exists {
			for _, cluster := range clusters {
				if !r.isMentionSeparatedFromCluster(m, cluster, separated) {
					cluster.Mentions = append(cluster.Mentions, m)
					r.addVariantsToCluster(cluster, m)
					added = true
					break
				}
			}
		}

		// If no compatible cluster found, create a new one
		if !added {
			cluster := &Cluster{
				Mentions:       []*Mention{m},
				HeadVariants:   []string{head},
				FullVariants:   []string{m.NormalizedHead},
				Titles:         append([]string{}, m.Titles...),
				Representative: m,
			}
			headMap[headLower] = append(headMap[headLower], cluster)
		}
	}

	// Convert map to flat slice
	clusters := make([]*Cluster, 0)
	for _, clusterList := range headMap {
		clusters = append(clusters, clusterList...)
	}

	return clusters
}

// addVariantsToCluster adds new name variants from a mention to a cluster.
func (r *Resolver) addVariantsToCluster(c *Cluster, m *Mention) {
	// Add head variant if not already present
	head := GetNameHead(m.Tokens)
	found := false
	for _, v := range c.HeadVariants {
		if strings.EqualFold(v, head) {
			found = true
			break
		}
	}
	if !found {
		c.HeadVariants = append(c.HeadVariants, head)
	}

	// Add full variant
	found = false
	for _, v := range c.FullVariants {
		if strings.EqualFold(v, m.NormalizedHead) {
			found = true
			break
		}
	}
	if !found {
		c.FullVariants = append(c.FullVariants, m.NormalizedHead)
	}

	// Add titles
	for _, t := range m.Titles {
		found = false
		for _, existing := range c.Titles {
			if strings.EqualFold(existing, t) {
				found = true
				break
			}
		}
		if !found {
			c.Titles = append(c.Titles, t)
		}
	}

	// Update representative if this mention has more tokens (more complete name)
	if len(m.Tokens) > len(c.Representative.Tokens) {
		c.Representative = m
	}
}

// mergeClusters iteratively merges clusters based on matching rules.
func (r *Resolver) mergeClusters(clusters []*Cluster, separated map[string]bool) []*Cluster {
	for {
		// Find best merge candidate
		best := r.findBestMerge(clusters, separated)
		if best == nil {
			break
		}

		// Perform merge: merge B into A, remove B
		r.doMerge(clusters[best.ClusterA], clusters[best.ClusterB])

		// Remove cluster B by swapping with last and truncating
		last := len(clusters) - 1
		clusters[best.ClusterB] = clusters[last]
		clusters = clusters[:last]
	}

	return clusters
}

// findBestMerge finds the highest-confidence merge candidate.
func (r *Resolver) findBestMerge(clusters []*Cluster, separated map[string]bool) *MergeCandidate {
	var best *MergeCandidate

	for i := 0; i < len(clusters); i++ {
		for j := i + 1; j < len(clusters); j++ {
			// Skip if any mentions are separated
			if r.areSeparated(clusters[i], clusters[j], separated) {
				continue
			}

			confidence, reason := r.shouldMerge(clusters[i], clusters[j])
			if confidence > 0 {
				if best == nil || confidence > best.Confidence {
					best = &MergeCandidate{
						ClusterA:   i,
						ClusterB:   j,
						Confidence: confidence,
						Reason:     reason,
					}
				}
			}
		}
	}

	return best
}

// shouldMerge determines if two clusters should be merged and with what confidence.
// Returns 0 if they should not be merged.
func (r *Resolver) shouldMerge(a, b *Cluster) (confidence float64, reason string) {
	// Rule 1: Subset/superset matching on tokens
	// "Ruiz" is a subset of "Carlos Ruiz"
	// "Daniel" is a subset of "Daniel Hanlon"
	for _, fullA := range a.FullVariants {
		tokensA := TokenizeName(fullA)
		for _, fullB := range b.FullVariants {
			tokensB := TokenizeName(fullB)

			if isSubset(tokensA, tokensB) || isSubset(tokensB, tokensA) {
				return 0.9, "subset"
			}
		}
	}

	// Rule 1b: First name matching
	// If one cluster has only single-token names (first names like "Daniel")
	// and another cluster has multi-token names where the first token matches,
	// merge them. E.g., "Daniel" should merge with cluster containing "Daniel Hanlon"
	for _, fullA := range a.FullVariants {
		tokensA := TokenizeName(fullA)
		if len(tokensA) != 1 {
			continue // Only check single-token variants
		}
		singleName := tokensA[0]
		for _, fullB := range b.FullVariants {
			tokensB := TokenizeName(fullB)
			if len(tokensB) >= 2 {
				// Check if single name matches first token (first name)
				if strings.EqualFold(singleName, tokensB[0]) {
					return 0.85, "firstname"
				}
				// Check if single name matches last token (surname)
				if strings.EqualFold(singleName, tokensB[len(tokensB)-1]) {
					return 0.85, "surname"
				}
			}
		}
	}
	// Check the reverse direction
	for _, fullB := range b.FullVariants {
		tokensB := TokenizeName(fullB)
		if len(tokensB) != 1 {
			continue
		}
		singleName := tokensB[0]
		for _, fullA := range a.FullVariants {
			tokensA := TokenizeName(fullA)
			if len(tokensA) >= 2 {
				if strings.EqualFold(singleName, tokensA[0]) {
					return 0.85, "firstname"
				}
				if strings.EqualFold(singleName, tokensA[len(tokensA)-1]) {
					return 0.85, "surname"
				}
			}
		}
	}

	// Rule 2: Same head with nickname match on first name
	// "Carlos Ruiz" matches "Charlie Ruiz" if Carlos↔Charlie
	for _, fullA := range a.FullVariants {
		tokensA := TokenizeName(fullA)
		if len(tokensA) < 2 {
			continue
		}
		for _, fullB := range b.FullVariants {
			tokensB := TokenizeName(fullB)
			if len(tokensB) < 2 {
				continue
			}

			// Same head (surname)?
			headA := GetNameHead(tokensA)
			headB := GetNameHead(tokensB)
			if !strings.EqualFold(headA, headB) {
				continue
			}

			// Check if first names are nicknames
			firstA := tokensA[0]
			firstB := tokensB[0]
			if IsNickname(firstA, firstB) {
				return 0.85, "nickname"
			}
		}
	}

	// Rule 3: Typo detection (Levenshtein ≤ 1)
	// "Ruiz" matches "Riuz" (typo)
	for _, headA := range a.HeadVariants {
		for _, headB := range b.HeadVariants {
			if IsLikelyTypo(headA, headB) && !strings.EqualFold(headA, headB) {
				return 0.7, "typo"
			}
		}
	}

	return 0, ""
}

// isSubset checks if tokens A are a subset of tokens B (case-insensitive).
// A single token is always considered a subset of multi-token names
// if that token matches any token in the superset.
func isSubset(tokensA, tokensB []string) bool {
	if len(tokensA) >= len(tokensB) {
		return false
	}

	// For each token in A, check if it exists in B
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

// doMerge merges cluster B into cluster A.
func (r *Resolver) doMerge(a, b *Cluster) {
	// Merge mentions
	a.Mentions = append(a.Mentions, b.Mentions...)

	// Merge head variants
	for _, h := range b.HeadVariants {
		found := false
		for _, existing := range a.HeadVariants {
			if strings.EqualFold(existing, h) {
				found = true
				break
			}
		}
		if !found {
			a.HeadVariants = append(a.HeadVariants, h)
		}
	}

	// Merge full variants
	for _, f := range b.FullVariants {
		found := false
		for _, existing := range a.FullVariants {
			if strings.EqualFold(existing, f) {
				found = true
				break
			}
		}
		if !found {
			a.FullVariants = append(a.FullVariants, f)
		}
	}

	// Merge titles
	for _, t := range b.Titles {
		found := false
		for _, existing := range a.Titles {
			if strings.EqualFold(existing, t) {
				found = true
				break
			}
		}
		if !found {
			a.Titles = append(a.Titles, t)
		}
	}

	// Update representative if B has a more complete name
	if len(b.Representative.Tokens) > len(a.Representative.Tokens) {
		a.Representative = b.Representative
	}
}

// clustersToEntities converts clusters to the final Entity format.
func (r *Resolver) clustersToEntities(clusters []*Cluster) []Entity {
	entities := make([]Entity, 0, len(clusters))

	for _, c := range clusters {
		if len(c.Mentions) == 0 {
			continue
		}

		// Pick canonical name - the longest full variant
		canonical := r.pickCanonical(c)

		// Build aliases (all variants except canonical, including titled versions)
		aliases := r.buildAliases(c, canonical)

		// Collect mention IDs
		mentionIDs := make([]string, len(c.Mentions))
		for i, m := range c.Mentions {
			mentionIDs[i] = m.ID
		}

		// Calculate average confidence (for now, set based on diversity)
		confidence := r.calculateConfidence(c)

		entity := Entity{
			ID:         fmt.Sprintf("entity-%d", time.Now().UnixNano()),
			Canonical:  canonical,
			Aliases:    aliases,
			MentionIDs: mentionIDs,
			Confidence: confidence,
			Titles:     c.Titles,
		}
		entities = append(entities, entity)
	}

	// Sort entities by mention count (most mentioned first)
	sort.Slice(entities, func(i, j int) bool {
		return len(entities[i].MentionIDs) > len(entities[j].MentionIDs)
	})

	return entities
}

// pickCanonical selects the canonical name for a cluster.
// Prefers the longest, most complete variant.
func (r *Resolver) pickCanonical(c *Cluster) string {
	if len(c.FullVariants) == 0 {
		return c.Representative.NormalizedHead
	}

	// Pick the variant with the most tokens
	best := c.FullVariants[0]
	bestTokens := len(TokenizeName(best))

	for _, v := range c.FullVariants[1:] {
		tokens := len(TokenizeName(v))
		if tokens > bestTokens {
			best = v
			bestTokens = tokens
		} else if tokens == bestTokens && len(v) > len(best) {
			// Same token count, prefer longer string
			best = v
		}
	}

	return best
}

// buildAliases constructs the alias list for an entity.
func (r *Resolver) buildAliases(c *Cluster, canonical string) []string {
	seen := make(map[string]bool)
	seen[strings.ToLower(canonical)] = true

	aliases := []string{}

	// Add all full variants (except canonical)
	for _, v := range c.FullVariants {
		lower := strings.ToLower(v)
		if !seen[lower] {
			aliases = append(aliases, v)
			seen[lower] = true
		}
	}

	// Add titled versions
	for _, title := range c.Titles {
		for _, v := range c.HeadVariants {
			titled := title + " " + v
			lower := strings.ToLower(titled)
			if !seen[lower] {
				aliases = append(aliases, titled)
				seen[lower] = true
			}
		}
	}

	// Add raw mention texts that differ from variants
	for _, m := range c.Mentions {
		lower := strings.ToLower(m.Text)
		if !seen[lower] {
			aliases = append(aliases, m.Text)
			seen[lower] = true
		}
	}

	return aliases
}

// calculateConfidence estimates confidence based on cluster characteristics.
func (r *Resolver) calculateConfidence(c *Cluster) float64 {
	// Base confidence: more mentions = higher confidence
	mentionBonus := float64(len(c.Mentions)) * 0.02
	if mentionBonus > 0.2 {
		mentionBonus = 0.2
	}

	// Variant diversity penalty: more variants = lower confidence
	// (might indicate over-merging)
	variantPenalty := float64(len(c.FullVariants)-1) * 0.05
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
// Returns the modified entity list and adds a SeparatedPair to prevent re-merging.
func (r *Resolver) SplitEntity(entities []Entity, entityID string, mentionIDs []string) ([]Entity, error) {
	// Find the entity
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

	// Validate that all mentionIDs belong to this entity
	mentionSet := make(map[string]bool)
	for _, id := range entity.MentionIDs {
		mentionSet[id] = true
	}
	for _, id := range mentionIDs {
		if !mentionSet[id] {
			return entities, fmt.Errorf("mention %s does not belong to entity %s", id, entityID)
		}
	}

	// Don't allow splitting all mentions
	if len(mentionIDs) >= len(entity.MentionIDs) {
		return entities, fmt.Errorf("cannot split all mentions from entity")
	}

	// Remove the specified mentions from the original entity
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

	// Create a new entity for the split mentions
	// The canonical name and aliases will need to be recomputed by caller
	// since we don't have access to the original mentions here
	newEntity := Entity{
		ID:         fmt.Sprintf("entity-%d", time.Now().UnixNano()),
		Canonical:  "", // To be filled in by caller
		Aliases:    []string{},
		MentionIDs: mentionIDs,
		Confidence: 0.7, // Lower confidence for manually split entities
		Titles:     []string{},
	}

	// Add separated pairs to prevent re-merging
	// Each split mention is separated from each remaining mention
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

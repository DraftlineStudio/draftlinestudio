package narrative

import (
	"strings"
	"unicode/utf8"

	"github.com/jdkato/prose/v3"
)

// This is a bounded grammar over Prose tokens, not a dependency parser. Lexical
// entries specify verb argument requirements; names and location noun phrases
// are inputs, not compiled sentence templates or a list of room names.
type actorBinding struct {
	id          string
	start, end  int
	antecedents []Anchor
}

type locationBinding struct {
	place, relation string
	start, end      int
}

type presenceClause struct {
	actors, companions []actorBinding
	location           *locationBinding
	predicate          int
	predicateEnd       int
	kind               string
	alone              bool
	complete           bool
}

type clauseReader struct {
	text       string
	tokens     []prose.Token
	roster     []nameMatch
	antecedent *actorBinding
}

func (r clauseReader) word(i int) string {
	if i < 0 || i >= len(r.tokens) {
		return ""
	}
	return strings.ToLower(r.tokens[i].Text)
}

func (r clauseReader) actor(i int) (actorBinding, int, bool) {
	if i >= len(r.tokens) {
		return actorBinding{}, i, false
	}
	start := r.tokens[i].Start
	for _, name := range r.roster {
		nameRunes := utf8.RuneCountInString(name.name)
		for j := i; j < len(r.tokens); j++ {
			end := r.tokens[j].End()
			candidate := r.text[start:end]
			if strings.EqualFold(candidate, name.name) {
				// A person's possessions are not that person's grammatical body.
				if r.word(j+1) == "'s" || r.word(j+1) == "’s" {
					return actorBinding{}, i, false
				}
				return actorBinding{id: name.id, start: start, end: end}, j + 1, true
			}
			if utf8.RuneCountInString(candidate) > nameRunes {
				break
			} // Unicode case folding can change byte lengths.
		}
	}
	if (r.word(i) == "he" || r.word(i) == "she") && r.antecedent != nil {
		return actorBinding{id: r.antecedent.id, start: start, end: r.tokens[i].End(), antecedents: append([]Anchor(nil), r.antecedent.antecedents...)}, i + 1, true
	}
	return actorBinding{}, i, false
}

func (r clauseReader) actors(i int) ([]actorBinding, int, bool) {
	first, next, ok := r.actor(i)
	if !ok {
		return nil, i, false
	}
	actors := []actorBinding{first}
	for r.word(next) == "and" {
		actor, end, found := r.actor(next + 1)
		if !found {
			break
		}
		actors = append(actors, actor)
		next = end
	}
	return actors, next, true
}

func nounTag(tag string) bool      { return strings.HasPrefix(tag, "NN") }
func adjectiveTag(tag string) bool { return strings.HasPrefix(tag, "JJ") }

// A descriptive comma continuation must end in a common noun and then a
// phrase boundary. In particular, "room, angry Tomas waited" and "room,
// angry strangers waited" are new clauses, not longer location names. This
// permits noun/adjective POS ambiguity before the comma without a word list.
func (r clauseReader) descriptiveContinuation(i int) bool {
	if i >= len(r.tokens) || !adjectiveTag(r.tokens[i].Tag) {
		return false
	}
	for i < len(r.tokens) && (adjectiveTag(r.tokens[i].Tag) || r.tokens[i].Tag == "CD") {
		if _, _, named := r.actor(i); named {
			return false
		}
		i++
	}
	start := i
	for i < len(r.tokens) && (r.tokens[i].Tag == "NN" || r.tokens[i].Tag == "NNS") {
		if _, _, named := r.actor(i); named {
			return false
		}
		i++
	}
	if i == start {
		return false
	}
	switch r.word(i) {
	case "", ",", ".", "!", "?", "of":
		return true
	}
	return spatialPreposition(r.word(i))
}

// These heads express manner/mental condition, not physical location. The
// exclusion is deliberately explicit; POS alone cannot establish concreteness.
// This is not claimed to exhaust metaphor or abstract spatial language.
var nonSpatialHeads = map[string]bool{
	"silence": true, "trouble": true, "love": true, "pain": true, "panic": true,
	"fear": true, "shock": true, "disbelief": true, "mood": true, "thought": true,
	"thoughts": true, "spirit": true, "agreement": true, "debt": true, "danger": true,
	"despair": true,
}

func (r clauseReader) nounPhrase(i int) (int, bool) {
	start := i
	if r.word(i) == "the" || r.word(i) == "a" || r.word(i) == "an" {
		i++
	}
	head := ""
	for i < len(r.tokens) {
		t := r.tokens[i]
		if (r.word(i) == "'s" || r.word(i) == "’s") && head != "" {
			head = ""
			i++
			continue
		}
		if nounTag(t.Tag) {
			head = strings.ToLower(t.Text)
			i++
			continue
		}
		if adjectiveTag(t.Tag) || t.Tag == "CD" {
			i++
			continue
		}
		// Degree adverbs modify an adjective, not an arbitrary later clause.
		if t.Tag == "RB" && i+1 < len(r.tokens) && adjectiveTag(r.tokens[i+1].Tag) {
			i++
			continue
		}
		// Prose can tag an attributive modifier as NN. Accept a descriptive
		// continuation only with a bounded lookahead, not any comma + noun.
		if t.Text == "," && i > start && (adjectiveTag(r.tokens[i-1].Tag) || r.tokens[i-1].Tag == "NN") && r.descriptiveContinuation(i+1) {
			i++
			continue
		}
		break
	}
	if head == "" || nonSpatialHeads[head] {
		return start, false
	}
	return i, true
}

func spatialPreposition(word string) bool {
	switch word {
	case "in", "inside", "at", "on", "above", "below", "outside", "under", "beneath", "behind", "near":
		return true
	}
	return false
}

func (r clauseReader) location(i int, direct bool) (*locationBinding, int, bool) {
	start := i
	relation := "in"
	if !direct {
		relation = r.word(i)
		if !spatialPreposition(relation) {
			return nil, start, false
		}
		i++
	}
	nounStart := i
	end, ok := r.nounPhrase(i)
	if !ok {
		return nil, start, false
	}
	// Additional PP qualifiers belong to the location only when followed by a
	// noun phrase; they cannot swallow a new finite clause or a roster companion.
	for spatialPreposition(r.word(end)) || r.word(end) == "of" {
		if _, _, person := r.actor(end + 1); person {
			break
		}
		next, found := r.nounPhrase(end + 1)
		if !found {
			break
		}
		end = next
	}
	if nounStart >= end {
		return nil, start, false
	}
	text := r.text[r.tokens[nounStart].Start:r.tokens[end-1].End()]
	place := canonicalPlace(text)
	// Relative positions must not collapse into occupancy of their reference
	// location: above a pass is not inside that pass.
	if relation != "in" && relation != "inside" && relation != "at" && relation != "on" {
		place = relation + " " + text
	}
	return &locationBinding{place: place, relation: relation, start: r.tokens[start].Start, end: r.tokens[end-1].End()}, end, true
}

func canonicalPlace(text string) string {
	text = strings.Join(strings.Fields(text), " ")
	for _, article := range []string{"the ", "a ", "an "} {
		if strings.HasPrefix(strings.ToLower(text), article) {
			return text[len(article):]
		}
	}
	return text
}

func (r clauseReader) parse() presenceClause {
	c := presenceClause{predicate: -1}
	if len(r.tokens) == 0 {
		return c
	}
	i := 0
	// Fronted spatial adjuncts bind to the following main-clause subject, not
	// the first entity anywhere in the sentence. No punctuation-less scan.
	front := 0
	if r.word(0) == "hidden" || r.word(0) == "standing" || r.word(0) == "sitting" {
		front = 1
	}
	if loc, end, ok := r.location(front, false); ok && r.word(end) == "," {
		c.location = loc
		i = end + 1
	}
	resultativeStart := i
	resultative := r.word(i) == "that" && r.word(i+1) == "left"
	if resultative {
		i += 2
	}
	if r.word(i) == "only" {
		c.alone = true
		i++
	}
	actors, next, ok := r.actors(i)
	if !ok {
		return presenceClause{predicate: -1}
	}
	c.actors = actors
	i = next
	if r.word(i) == "himself" || r.word(i) == "herself" || r.word(i) == "themselves" {
		i++
	}
	if resultative {
		if r.word(i) != "alone" {
			return presenceClause{predicate: -1}
		}
		c.predicate = resultativeStart
		c.predicateEnd = resultativeStart + 2
		c.kind = "alone"
		c.alone = true
	} else {
		c.predicate = i
		c.predicateEnd = i + 1
		switch r.word(i) {
		case "is", "are", "was", "were", "sat", "sit", "sits", "stood", "stand", "stands", "waited", "wait", "waits", "remained", "remain", "remains", "stayed", "stay", "stays":
			c.kind = "present"
		case "entered", "enter", "enters":
			c.kind = "present"
			loc, end, found := r.location(i+1, true)
			if !found {
				return presenceClause{predicate: -1}
			}
			c.location = loc
			i = end - 1
		case "left", "leave", "leaves", "exited", "exit", "exits":
			c.kind = "departed"
			loc, end, found := r.location(i+1, true)
			if !found {
				return presenceClause{predicate: -1}
			}
			c.location = loc
			c.location.relation = "from"
			i = end - 1
		case "departed", "depart", "departs":
			c.predicateEnd = i + 2
			if r.word(i+1) != "from" {
				return presenceClause{predicate: -1}
			}
			c.kind = "departed"
			loc, end, found := r.location(i+2, true)
			if !found {
				return presenceClause{predicate: -1}
			}
			c.location = loc
			c.location.relation = "from"
			i = end - 1
		case "walked", "went", "ran", "slipped", "moved":
			c.predicateEnd = i + 3
			if r.word(i+1) != "out" || r.word(i+2) != "of" {
				return presenceClause{predicate: -1}
			}
			c.kind = "departed"
			loc, end, found := r.location(i+3, true)
			if !found {
				return presenceClause{predicate: -1}
			}
			c.location = loc
			c.location.relation = "from"
			i = end - 1
		case "arrived":
			c.kind = "present"
		default:
			// A finite main predicate with a fronted spatial adjunct establishes
			// presence but is not promoted into an event/development.
			if c.location == nil || i >= len(r.tokens) || !strings.HasPrefix(r.tokens[i].Tag, "VB") {
				return presenceClause{predicate: -1}
			}
			c.kind = "present"
			return c // the main verb's complement is not another location argument
		}
		i++
	}
	exception, negatedAlone, unresolvedCompanion := false, false, false
	for i < len(r.tokens) {
		switch r.word(i) {
		case ".", "!", "?":
			i++
		case ",":
			i++
		case "alone":
			c.alone = true
			i++
		case "not":
			if r.word(i+1) != "alone" {
				return presenceClause{predicate: -1}
			}
			negatedAlone = true
			i += 2
		case "except":
			exception = true
			i++
			if r.word(i) == "for" {
				i++
			}
			companions, end, found := r.actors(i)
			if !found {
				unresolvedCompanion = true
				i = len(r.tokens)
				break
			}
			c.companions = append(c.companions, companions...)
			i = end
		case "with", "beside", "alongside":
			companions, end, found := r.actors(i + 1)
			if !found {
				// Thoughts are not a physical companion in the resultative idiom.
				if !(c.kind == "alone" && (r.word(i+1) == "his" || r.word(i+1) == "her" || r.word(i+1) == "their") && r.word(i+2) == "thoughts") {
					unresolvedCompanion = true
				}
				i = len(r.tokens)
				break
			}
			c.companions = append(c.companions, companions...)
			i = end
		default:
			if loc, end, found := r.location(i, false); found {
				if c.location != nil {
					return presenceClause{predicate: -1}
				} // ambiguous attachment, not last-place-wins
				c.location = loc
				i = end
				continue
			}
			// An unparsed continuation is retained in coverage, not mined for
			// verbs or participants belonging to someone else's clause.
			c.alone = c.alone && !exception && !negatedAlone && !unresolvedCompanion
			return c
		}
	}
	c.alone = c.alone && !exception && !negatedAlone && !unresolvedCompanion
	c.complete = !unresolvedCompanion
	return c
}

// Only a complete, intransitive named-subject sentence supplies a local pronoun
// candidate. A companion, object, quoted turn, intervening unknown sentence or
// scene boundary invalidates it. No gender is guessed from names.
func (r clauseReader) nextAntecedent(d Document, block int, base int) *actorBinding {
	actors, i, ok := r.actors(0)
	if !ok || len(actors) != 1 || len(actors[0].antecedents) > 0 || i >= len(r.tokens) {
		return nil
	}
	if !strings.HasPrefix(r.tokens[i].Tag, "VB") {
		return nil
	}
	i++
	for i < len(r.tokens) && (r.tokens[i].Tag == "RB" || r.word(i) == "." || r.word(i) == "!" || r.word(i) == "?") {
		i++
	}
	if i != len(r.tokens) {
		return nil
	}
	a := actors[0]
	a.antecedents = []Anchor{d.Anchor(block, base, base+len(r.text))}
	return &a
}

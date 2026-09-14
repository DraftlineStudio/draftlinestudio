package narrative

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/jdkato/prose/v3"
)

// Only an initial, balanced, sentence-terminal dialogue turn may be skipped.
// Do not mine quotations, attribution complements, or unmatched multi-paragraph
// dialogue for current action. All offsets still address the original block.
func movementNarrationStart(text string) (int, bool) {
	start := len(text) - len(strings.TrimLeftFunc(text, unicode.IsSpace))
	if start == len(text) {
		return start, false
	}
	open, size := utf8.DecodeRuneInString(text[start:])
	if open != '“' && open != '"' {
		return start, !strings.ContainsAny(text, "“”\"")
	}
	close := '”'
	if open == '"' {
		close = '"'
	}
	i := start + size
	for i < len(text) {
		ch, n := utf8.DecodeRuneInString(text[i:])
		if ch == close {
			quoted := strings.TrimSpace(text[start+size : i])
			if !strings.HasSuffix(quoted, ".") && !strings.HasSuffix(quoted, "!") && !strings.HasSuffix(quoted, "?") {
				return 0, false
			}
			end := i + n
			end += len(text[end:]) - len(strings.TrimLeftFunc(text[end:], unicode.IsSpace))
			return end, end < len(text) && !strings.ContainsAny(text[end:], "“”\"")
		}
		if ch == '“' || ch == '”' || ch == '"' {
			return 0, false
		}
		i += n
	}
	return 0, false
}

// These are traversal apertures, not a vocabulary of story locations. A route
// remains a route; e.g. a door is not the room that the participant departed.
func aperture(word string) bool {
	switch word {
	case "door", "doorway", "gate", "gateway", "hatch", "window", "exit":
		return true
	}
	return false
}

func balancedMovementQuotes(text string) bool {
	var closing rune
	for _, ch := range text {
		if ch != '“' && ch != '”' && ch != '"' {
			continue
		}
		if closing != 0 {
			if ch != closing {
				return false
			}
			closing = 0
		} else {
			if ch == '”' {
				return false
			}
			closing = '"'
			if ch == '“' {
				closing = '”'
			}
		}
	}
	return closing == 0
}

func extractMovements(d Document, people []Person, observations []Observation) ([]MovementEvidence, error) {
	var result []MovementEvidence
	roster := names(people)
	scene, uncertainQuotation := "", false
	for bi, b := range d.Blocks {
		if b.Scene != scene {
			scene, uncertainQuotation = b.Scene, false
		}
		// Multi-paragraph/unbalanced quotation is not safely resolved by the
		// local splitter. Abstain through this scene rather than treating its
		// next unquoted-looking paragraph as independent current narration.
		if !balancedMovementQuotes(b.Text) {
			uncertainQuotation = true
		}
		if uncertainQuotation {
			continue
		}
		if b.Mode != "current_narration" {
			continue
		}
		start, ok := movementNarrationStart(b.Text)
		if !ok {
			continue
		}
		segmented, err := prose.NewDocument(b.Text[start:], prose.WithExtraction(false), prose.WithTagging(false), prose.WithTokenization(false))
		if err != nil {
			return nil, err
		}
		for _, sentence := range segmented.Sentences() {
			if unsafeContext.MatchString(sentence.Text) {
				continue
			}
			tagged, err := prose.NewDocument(sentence.Text, prose.WithExtraction(false), prose.WithSegmentation(false))
			if err != nil {
				return nil, err
			}
			r := clauseReader{text: sentence.Text, tokens: tagged.Tokens(), roster: roster}
			if alternative, found := r.alternativeMovement(d, bi, start+sentence.Start); found {
				alternative.ID = id(d.Revision, b.ID, fmt.Sprint(alternative.Source.Start), alternative.Rule)
				result = append(result, alternative)
				continue
			}
			leader, verb, ok := r.actor(0)
			if !ok {
				continue
			}
			switch r.word(verb) {
			case "led", "guided", "escorted", "herded":
			default:
				continue
			}
			base := start + sentence.Start
			anchor := func(a, z int) Anchor { return d.Anchor(bi, base+a, base+z) }
			m := MovementEvidence{Scene: b.Scene, Source: anchor(0, len(sentence.Text)), Rule: "group-motion-clause-v1"}
			m.Participants = append(m.Participants, MovementParticipant{Subject: leader.id, Role: "leader", Source: anchor(leader.start, leader.end), IdentityStatus: "explicit"})
			i := verb + 1
			followers, end, named := r.actors(i)
			if named {
				for _, follower := range followers {
					m.Participants = append(m.Participants, MovementParticipant{Subject: follower.id, Role: "follower", Source: anchor(follower.start, follower.end), IdentityStatus: "explicit"})
				}
				i = end
			} else if r.word(i) == "them" || r.word(i) == "him" || r.word(i) == "her" {
				m.Participants = append(m.Participants, MovementParticipant{Role: "follower", Source: anchor(r.tokens[i].Start, r.tokens[i].End()), IdentityStatus: "unresolved"})
				i++
			} else {
				continue
			}
			if r.word(i) != "out" {
				continue
			}
			particle := i
			i++
			var explicitOrigin *MovementOrigin
			if r.word(i) == "of" {
				loc, next, found := r.location(i+1, true)
				if !found {
					continue
				}
				ls := anchor(loc.start, loc.end)
				explicitOrigin = &MovementOrigin{Place: loc.place, Status: "explicit", Source: &ls}
				i = next
			} else {
				if r.word(i) == "through" {
					i++
				}
				routeStart := i
				next, found := r.nounPhrase(i)
				if !found || !aperture(r.word(next-1)) {
					continue
				}
				rs := anchor(r.tokens[routeStart].Start, r.tokens[next-1].End())
				m.RouteSource = &rs
				i = next
			}
			// Unknown bare complements could reverse the apparent construction.
			// A comma-separated continuation is retained but not interpreted.
			if r.word(i) != "" && r.word(i) != "," && r.word(i) != "." && r.word(i) != "!" && r.word(i) != "?" {
				continue
			}
			m.Complete = r.word(i) != "," && (i == len(r.tokens) || i+1 == len(r.tokens))
			particleEnd := particle + 1
			if explicitOrigin != nil || r.word(particle+1) == "through" {
				particleEnd++
			}
			m.PredicateSources = []Anchor{anchor(r.tokens[verb].Start, r.tokens[verb].End()), anchor(r.tokens[particle].Start, r.tokens[particleEnd-1].End())}
			for pi := range m.Participants {
				p := &m.Participants[pi]
				if explicitOrigin != nil {
					copyOrigin := *explicitOrigin
					p.Origin = &copyOrigin
				} else if p.Subject != "" {
					for _, o := range observations {
						if supportsMovementOrigin(d, o, m, *p) {
							if p.Origin != nil {
								p.Origin = nil
								break
							} // ambiguous evidence, not last-wins
							p.Origin = &MovementOrigin{Place: o.Place, Status: "inferred_adjacent_occupancy", Premise: o.ID}
						}
					}
				}
			}
			m.ID = id(d.Revision, b.ID, fmt.Sprint(m.Source.Start), m.Rule)
			result = append(result, m)
		}
	}
	return result, nil
}

// Adjacency is a conservative inference guard, not proof of continuity. Only
// whitespace may separate the explicit occupancy and movement, regardless of
// paragraph formatting. No unknown clause, quoted turn or scene may intervene.
func supportsMovementOrigin(d Document, o Observation, m MovementEvidence, p MovementParticipant) bool {
	if p.Subject == "" || o.Subject != p.Subject || o.Kind != "present" || o.Mode != "current_narration" || o.Scene != m.Scene || o.BindingStatus != "explicit" || !o.ClauseComplete || o.PlaceSource == nil || o.Place == "" {
		return false
	}
	if o.SpatialRelation != "in" && o.SpatialRelation != "inside" {
		return false
	}
	from, err := d.Position(o.Source)
	if err != nil {
		return false
	}
	to, err := d.Position(m.Source)
	if err != nil || from > to {
		return false
	}
	if from == to {
		return o.Source.End <= m.Source.Start && strings.TrimSpace(d.Blocks[from].Text[o.Source.End:m.Source.Start]) == ""
	}
	if strings.TrimSpace(d.Blocks[from].Text[o.Source.End:]) != "" || strings.TrimSpace(d.Blocks[to].Text[:m.Source.Start]) != "" {
		return false
	}
	for i := from; i <= to; i++ {
		if d.Blocks[i].Scene != m.Scene || d.Blocks[i].Mode != "current_narration" {
			return false
		}
		if i > from && i < to && strings.TrimSpace(d.Blocks[i].Text) != "" {
			return false
		}
	}
	return true
}

// Structural/source validation is not semantic proof. Candidates deliberately
// remain outside the accepted observation stream used to close inspections.
func validateMovements(d Document, e Extraction) error {
	observations := map[string]Observation{}
	for _, o := range e.Observations {
		observations[o.ID] = o
	}
	seen := map[string]bool{}
	for _, m := range e.Movements {
		bi, err := d.Position(m.Source)
		if err != nil {
			return err
		}
		if m.ID == "" || seen[m.ID] || m.Scene != d.Blocks[bi].Scene || d.Blocks[bi].Mode != "current_narration" || m.Rule == "" || len(m.Participants) < 2 || len(m.PredicateSources) != 2 {
			return fmt.Errorf("invalid movement candidate")
		}
		seen[m.ID] = true
		field := func(a Anchor) error {
			if _, err := d.Position(a); err != nil {
				return err
			}
			if a.BlockID != m.Source.BlockID || a.Start < m.Source.Start || a.End > m.Source.End {
				return fmt.Errorf("movement field outside clause")
			}
			return nil
		}
		for _, a := range m.PredicateSources {
			if err := field(a); err != nil {
				return err
			}
		}
		if m.RouteSource != nil {
			if err := field(*m.RouteSource); err != nil {
				return err
			}
		}
		for pi, p := range m.Participants {
			if err := field(p.Source); err != nil {
				return err
			}
			if pi == 0 && p.Role != "leader" || pi > 0 && p.Role != "follower" {
				return fmt.Errorf("invalid movement participant role")
			}
			if p.IdentityStatus == "explicit" {
				if p.Subject == "" {
					return fmt.Errorf("explicit participant without identity")
				}
			} else if p.IdentityStatus == "unexpressed" {
				if p.Subject != "" || p.Role != "leader" || m.Rule != "group-motion-passive-v2" || p.Origin != nil {
					return fmt.Errorf("invalid unexpressed passive agent")
				}
			} else if p.IdentityStatus != "unresolved" || p.Subject != "" || pi == 0 {
				return fmt.Errorf("invalid unresolved participant")
			}
			if p.Origin == nil {
				if m.RouteSource == nil && !(m.Rule == "group-motion-passive-v2" && p.Role == "leader") {
					return fmt.Errorf("movement without route or origin")
				}
				continue
			}
			o := p.Origin
			switch o.Status {
			case "explicit":
				if o.Source == nil || o.Premise != "" || o.Place != canonicalPlace(o.Source.Quote) {
					return fmt.Errorf("invalid explicit movement origin")
				}
				if err := field(*o.Source); err != nil {
					return err
				}
			case "inferred_adjacent_occupancy":
				prior, exists := observations[o.Premise]
				if !exists || o.Source != nil || m.RouteSource == nil || o.Place != prior.Place || !supportsMovementOrigin(d, prior, m, p) {
					return fmt.Errorf("unsupported movement origin premise")
				}
			default:
				return fmt.Errorf("unknown movement origin status")
			}
		}
	}
	return nil
}

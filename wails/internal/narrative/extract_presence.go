package narrative

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/jdkato/prose/v3"
)

// Discourse interpretation is still conservative: explicit non-current cues
// and quoted paragraphs abstain. This is not a general flashback classifier.
var unsafeContext = regexp.MustCompile(`(?i)\b(?:remembered|recalled|dreamed|dreaming|imagined|would|might|could|if|once|earlier|ago|yesterday|previously)\b`)

type nameMatch struct{ id, name string }

func names(people []Person) []nameMatch {
	owners := map[string]string{}
	for _, p := range people {
		for _, name := range p.Names {
			key := strings.ToLower(strings.TrimSpace(name))
			if key == "" || p.ID == "" {
				continue
			}
			if owner, exists := owners[key]; exists && owner != p.ID {
				owners[key] = ""
			} else if !exists {
				owners[key] = p.ID
			}
		}
	}
	result := []nameMatch{}
	for name, owner := range owners {
		if owner != "" {
			result = append(result, nameMatch{owner, name})
		}
	}
	sort.Slice(result, func(i, j int) bool {
		if len(result[i].name) != len(result[j].name) {
			return len(result[i].name) > len(result[j].name)
		}
		return result[i].name < result[j].name
	})
	return result
}

func ExtractPresence(d Document, people []Person) (Extraction, error) {
	if err := d.Validate(); err != nil {
		return Extraction{}, err
	}
	result := Extraction{Observations: []Observation{}, UnparsedBlocks: []string{}}
	roster := names(people)
	var previous *actorBinding
	scene := ""
	seen := map[string]bool{}
	for bi, b := range d.Blocks {
		// Local pronoun binding never crosses a paragraph. Multiple named
		// candidates anywhere in this paragraph also prohibit the shortcut.
		previous = nil
		paragraphActors := map[string]bool{}
		if b.Scene != scene {
			previous = nil
			scene = b.Scene
		}
		if b.Mode != "current_narration" || strings.ContainsAny(b.Text, "“”\"") {
			result.UnparsedBlocks = append(result.UnparsedBlocks, b.ID)
			previous = nil
			continue
		}
		segmented, err := prose.NewDocument(b.Text, prose.WithExtraction(false), prose.WithTagging(false), prose.WithTokenization(false))
		if err != nil {
			return Extraction{}, err
		}
		fullyParsed := true
		for _, sentence := range segmented.Sentences() {
			text := sentence.Text
			base := sentence.Start
			if unsafeContext.MatchString(text) {
				fullyParsed = false
				previous = nil
				continue
			}
			tagged, err := prose.NewDocument(text, prose.WithExtraction(false), prose.WithSegmentation(false))
			if err != nil {
				return Extraction{}, err
			}
			r := clauseReader{text: text, tokens: tagged.Tokens(), roster: roster, antecedent: previous}
			for tokenIndex := range r.tokens {
				if actor, _, ok := (clauseReader{text: text, tokens: r.tokens, roster: roster}).actor(tokenIndex); ok {
					paragraphActors[actor.id] = true
				}
			}
			if len(paragraphActors) > 1 {
				r.antecedent = nil
			}
			c := r.parse()
			previous = r.nextAntecedent(d, bi, base)
			if c.predicate < 0 || c.predicate >= len(r.tokens) || c.predicateEnd <= c.predicate || c.predicateEnd > len(r.tokens) {
				fullyParsed = false
				continue
			}
			before := len(result.Observations)
			add := func(actor actorBinding, kind string) {
				if c.location == nil && kind != "alone" {
					return
				}
				predicate := r.tokens[c.predicate]
				ps := d.Anchor(bi, base+predicate.Start, base+r.tokens[c.predicateEnd-1].End())
				o := Observation{Kind: kind, Subject: actor.id, Scene: b.Scene, Mode: "current_narration", Source: d.Anchor(bi, base, base+len(text)), SubjectSource: d.Anchor(bi, base+actor.start, base+actor.end), PredicateSource: &ps, Antecedents: actor.antecedents, Rule: "clause-presence-v2"}
				if c.location != nil {
					o.ClauseComplete = c.complete
					o.Place = c.location.place
					o.SpatialRelation = c.location.relation
					ls := d.Anchor(bi, base+c.location.start, base+c.location.end)
					o.PlaceSource = &ls
				}
				if len(actor.antecedents) > 0 {
					o.Rule += "/local-single-subject-continuation"
					o.BindingStatus = "inferred_local_subject"
				} else {
					o.BindingStatus = "explicit"
				}
				o.ID = id(d.Revision, b.ID, fmt.Sprint(o.Source.Start), o.Subject, kind, o.Place)
				if !seen[o.ID] {
					seen[o.ID] = true
					result.Observations = append(result.Observations, o)
				}
			}
			if c.kind != "alone" {
				for _, actor := range c.actors {
					add(actor, c.kind)
				}
			}
			if c.kind == "present" {
				for _, actor := range c.companions {
					add(actor, "present")
				}
			}
			if c.alone && len(c.actors) == 1 {
				add(c.actors[0], "alone")
			}
			if !c.complete || len(result.Observations) == before {
				fullyParsed = false
			}
		}
		if !fullyParsed {
			result.UnparsedBlocks = append(result.UnparsedBlocks, b.ID)
		}
	}
	movements, err := extractMovements(d, people, result.Observations)
	if err != nil {
		return Extraction{}, err
	}
	result.Movements = movements
	return result, nil
}

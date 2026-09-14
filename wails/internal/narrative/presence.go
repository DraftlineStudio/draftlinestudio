package narrative

import (
	"fmt"
	"sort"
	"strings"
)

func normalizePlace(s string) string {
	s = strings.ToLower(strings.Join(strings.Fields(s), " "))
	for _, prefix := range []string{"the ", "an ", "a "} {
		s = strings.TrimPrefix(s, prefix)
	}
	return s
}

// InspectPresence is intentionally a review-only, open-world inspection: even
// with no unparsed blocks, lack of an extracted departure is not proof nobody
// left off-screen. Each warning includes both witnesses and search coverage.
func InspectPresence(d Document, extraction Extraction) ([]Inspection, error) {
	if err := d.Validate(); err != nil {
		return nil, err
	}
	obs := append([]Observation(nil), extraction.Observations...)
	positions := map[string]int{}
	for _, o := range obs {
		p, err := d.Position(o.Source)
		if err != nil {
			return nil, err
		}
		if _, err = d.Position(o.SubjectSource); err != nil {
			return nil, err
		}
		if o.SubjectSource.BlockID != o.Source.BlockID || o.SubjectSource.Start < o.Source.Start || o.SubjectSource.End > o.Source.End {
			return nil, fmt.Errorf("subject span outside proposition")
		}
		for _, field := range []*Anchor{o.PredicateSource, o.PlaceSource} {
			if field == nil {
				continue
			}
			if _, err := d.Position(*field); err != nil {
				return nil, err
			}
			if field.BlockID != o.Source.BlockID || field.Start < o.Source.Start || field.End > o.Source.End {
				return nil, fmt.Errorf("field span outside proposition")
			}
		}
		for _, antecedent := range o.Antecedents {
			pos, err := d.Position(antecedent)
			if err != nil {
				return nil, err
			}
			if pos != p || antecedent.End > o.Source.Start {
				return nil, fmt.Errorf("nonlocal or future pronoun antecedent")
			}
		}
		if o.BindingStatus == "inferred_local_subject" && len(o.Antecedents) == 0 {
			return nil, fmt.Errorf("inferred subject has no antecedent evidence")
		}
		if _, duplicate := positions[o.ID]; duplicate {
			return nil, fmt.Errorf("duplicate observation %s", o.ID)
		}
		if o.ID == "" || o.Subject == "" || o.Rule == "" || o.Scene != d.Blocks[p].Scene {
			return nil, fmt.Errorf("invalid observation %q", o.ID)
		}
		positions[o.ID] = p
	}
	if err := validateMovements(d, extraction); err != nil {
		return nil, err
	}
	sort.SliceStable(obs, func(i, j int) bool {
		if positions[obs[i].ID] != positions[obs[j].ID] {
			return positions[obs[i].ID] < positions[obs[j].ID]
		}
		if obs[i].Source.Start != obs[j].Source.Start {
			return obs[i].Source.Start < obs[j].Source.Start
		}
		// Co-presence premises must be processed before the same-clause
		// exclusivity constraint, irrespective of input batch order.
		if (obs[i].Kind == "alone") != (obs[j].Kind == "alone") {
			return obs[j].Kind == "alone"
		}
		return obs[i].ID < obs[j].ID
	})
	active := map[string]Observation{}
	result := []Inspection{}
	scene := ""
	for _, o := range obs {
		if o.Scene != scene {
			active = map[string]Observation{}
			scene = o.Scene
		}
		if o.Mode != "current_narration" {
			continue
		}
		place := normalizePlace(o.Place)
		if prior, ok := active[o.Subject]; ok && (place == "" || place == "room" && strings.HasSuffix(prior.Place, "room")) {
			place = prior.Place
		}
		switch o.Kind {
		case "present":
			if place != "" {
				o.Place = place
				active[o.Subject] = o
			}
		case "departed":
			if prior, ok := active[o.Subject]; ok && prior.Place == place {
				delete(active, o.Subject)
			}
		case "alone":
			if place == "" {
				continue
			}
			keys := []string{}
			for subject := range active {
				keys = append(keys, subject)
			}
			sort.Strings(keys)
			for _, subject := range keys {
				prior := active[subject]
				if subject == o.Subject || prior.Place != place {
					continue
				}
				gaps := []string{}
				for _, blockID := range extraction.UnparsedBlocks {
					for index, b := range d.Blocks {
						if b.ID == blockID && index >= positions[prior.ID] && index <= positions[o.ID] {
							gaps = append(gaps, blockID)
							break
						}
					}
				}
				kind, detail := "unclosed_presence", "Earlier co-presence and later solitude have no identified connecting departure. Review for an omitted transition, off-screen departure, or extraction gap."
				if prior.Source == o.Source {
					kind = "conflicting_copresence"
					detail = "The same passage establishes another person here while describing the subject as alone. Review the wording or extraction."
				}
				result = append(result, Inspection{Kind: kind, Subject: subject, Place: place, Detail: detail, Premises: []string{prior.ID, o.ID}, Sources: []Anchor{prior.Source, o.Source}, SearchFrom: prior.Source, SearchThrough: o.Source, UnparsedBlocks: gaps, Coverage: "bounded_grammar_not_proof_of_absence"})
				// The newer explicit occupancy replaces the prior hypothesis;
				// do not repeat the same warning at every subsequent alone cue.
				delete(active, subject)
			}
		}
	}
	return result, nil
}

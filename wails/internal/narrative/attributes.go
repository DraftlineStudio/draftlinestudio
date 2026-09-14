package narrative

import "regexp"

// AttributeClaim is an evidence index, not canon. Surface agreement does not
// identify the same object across scenes; differing state values are not errors.
type AttributeClaim struct {
	SubjectSurface  string `json:"subject_surface"`
	Property        string `json:"property"`
	Value           string `json:"value"`
	PreviousValue   string `json:"previous_value,omitempty"`
	Mode            string `json:"mode"`
	IdentityStatus  string `json:"identity_status"`
	TemporalStatus  string `json:"temporal_status"`
	Source          Anchor `json:"source"`
	SubjectSource   Anchor `json:"subject_source"`
	ValueSource     Anchor `json:"value_source"`
	ContextSource   Anchor `json:"context_source"`
	AssertionStatus string `json:"assertion_status"`
}

var attributiveColor = regexp.MustCompile(`(?i)\b(red|green|blue|amber|yellow|white|black|gray|grey) (LED|light|indicator|keypad|hair|eyes|logo)\b`)
var predicativeColor = regexp.MustCompile(`(?i)\b((?:status )?(?:LED|light|indicator|keypad|hair|eyes|logo)) (?:was|is|burned|glowed|blinked|turned) (red|green|blue|amber|yellow|white|black|gray|grey)\b`)
var colorTransition = regexp.MustCompile(`(?i)\b((?:status )?(?:light|indicator|LED|keypad))(?: above the door)? changed from (red|green|blue|amber|yellow|white|black|gray|grey) to (red|green|blue|amber|yellow|white|black|gray|grey)\b(?:, then (red|green|blue|amber|yellow|white|black|gray|grey) to (red|green|blue|amber|yellow|white|black|gray|grey)\b)?`)
var statedAge = regexp.MustCompile(`(?i)\b([\pL][\pL’'-]*) (?:is|was) ([0-9]{1,3}|(?:twenty|thirty|forty|fifty|sixty|seventy|eighty|ninety)(?:[- ](?:one|two|three|four|five|six|seven|eight|nine))?) years old\b`)
var firstPersonAge = regexp.MustCompile(`(?i)\b(I)['’]m ((?:twenty|thirty|forty|fifty|sixty|seventy|eighty|ninety)(?:[- ](?:one|two|three|four|five|six|seven|eight|nine))?)\b`)
var birthYear = regexp.MustCompile(`(?i)\b([\pL][\pL’'-]*) was born in ([0-9]{4})\b`)

func extractAttributes(d Document) []AttributeClaim {
	result := []AttributeClaim{}
	for bi, b := range d.Blocks {
		if b.Mode != "current_narration" {
			continue
		}
		for _, span := range plotSpans(b.Text) {
			text := b.Text[span.start:span.end]
			add := func(m []int, subject, value int, property, previous, temporal string) {
				a := func(i, j int) Anchor { return d.Anchor(bi, span.start+i, span.start+j) }
				result = append(result, AttributeClaim{SubjectSurface: text[m[subject]:m[subject+1]], Property: property, Value: text[m[value]:m[value+1]], PreviousValue: previous, Mode: span.mode, IdentityStatus: "surface_only_unresolved_identity", TemporalStatus: temporal, Source: a(m[0], m[1]), SubjectSource: a(m[subject], m[subject+1]), ValueSource: a(m[value], m[value+1])})
				result[len(result)-1].ContextSource = a(0, len(text))
				result[len(result)-1].AssertionStatus = "mention_only_requires_context_interpretation"
			}
			for _, m := range attributiveColor.FindAllStringSubmatchIndex(text, -1) {
				add(m, 4, 2, "color", "", "state_or_description_not_invariant")
			}
			for _, m := range predicativeColor.FindAllStringSubmatchIndex(text, -1) {
				add(m, 2, 4, "color", "", "state_or_description_not_invariant")
			}
			for _, m := range colorTransition.FindAllStringSubmatchIndex(text, -1) {
				add(m, 2, 6, "color", text[m[4]:m[5]], "explicit_state_transition")
				if m[8] >= 0 && text[m[8]:m[9]] == text[m[6]:m[7]] {
					add(m, 2, 10, "color", text[m[8]:m[9]], "explicit_state_transition")
				}
			}
			for _, m := range statedAge.FindAllStringSubmatchIndex(text, -1) {
				add(m, 2, 4, "age", "", "reference_time_unresolved")
			}
			if span.mode == "unattributed_report" {
				for _, m := range firstPersonAge.FindAllStringSubmatchIndex(text, -1) {
					add(m, 2, 4, "age", "", "speaker_and_reference_time_unresolved")
				}
			}
			for _, m := range birthYear.FindAllStringSubmatchIndex(text, -1) {
				add(m, 2, 4, "birth_year", "", "stated_year_not_verified")
			}
		}
	}
	return result
}

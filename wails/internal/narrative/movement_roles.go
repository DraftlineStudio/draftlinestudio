package narrative

// Alternative grammatical voices bind semantic roles independently of word
// order. They remain bounded candidates, not a general dependency parser.
func traversalVerb(word string) bool {
	switch word {
	case "led", "guided", "escorted", "herded":
		return true
	}
	return false
}

func movementBoundary(word string) bool {
	switch word {
	case "", ",", ".", "!", "?":
		return true
	}
	return false
}

func (r clauseReader) alternativeMovement(d Document, bi, base int) (MovementEvidence, bool) {
	anchor := func(a, z int) Anchor { return d.Anchor(bi, base+a, base+z) }
	tokenAnchor := func(a, z int) Anchor { return anchor(r.tokens[a].Start, r.tokens[z-1].End()) }
	m := MovementEvidence{Scene: d.Blocks[bi].Scene, Source: anchor(0, len(r.text))}
	actors, i, named := r.actors(0)
	var followers []MovementParticipant
	if named {
		for _, a := range actors {
			followers = append(followers, MovementParticipant{Subject: a.id, Role: "follower", Source: anchor(a.start, a.end), IdentityStatus: "explicit"})
		}
	} else if r.word(0) == "they" || r.word(0) == "he" || r.word(0) == "she" {
		followers = []MovementParticipant{{Role: "follower", Source: tokenAnchor(0, 1), IdentityStatus: "unresolved"}}
		i = 1
	} else {
		return m, false
	}
	verb := i
	passive := r.word(i) == "was" || r.word(i) == "were" || r.word(i) == "is" || r.word(i) == "are"
	if passive {
		i++
	}
	// Reflexive permission is a traversal candidate; wanting/planning to be
	// led is not. Do not guess gender from a supplied name.
	if r.word(i) == "let" && (r.word(i+1) == "herself" || r.word(i+1) == "himself" || r.word(i+1) == "themselves") && r.word(i+2) == "be" {
		if len(followers) > 1 && r.word(i+1) != "themselves" {
			return m, false
		}
		if !named && (r.word(0) == "they" && r.word(i+1) != "themselves" || r.word(0) == "he" && r.word(i+1) != "himself" || r.word(0) == "she" && r.word(i+1) != "herself") {
			return m, false
		}
		passive = true
		i += 3
	}
	if passive {
		if !traversalVerb(r.word(i)) {
			return m, false
		}
		participle := i
		i++
		particle := i
		if r.word(i) == "from" {
			i++
		} else if r.word(i) == "out" && r.word(i+1) == "of" {
			i += 2
		} else {
			return m, false
		}
		particleEnd := i
		loc, end, found := r.location(i, true)
		if !found {
			return m, false
		}
		i = end
		// No participant mention exists for an omitted passive agent. Its
		// source is the predicate licensing that unexpressed role, not a name.
		leader := MovementParticipant{Role: "leader", IdentityStatus: "unexpressed", Source: tokenAnchor(participle, participle+1)}
		if r.word(i) == "by" {
			a, next, ok := r.actor(i + 1)
			if !ok {
				return m, false
			}
			leader = MovementParticipant{Role: "leader", Subject: a.id, IdentityStatus: "explicit", Source: anchor(a.start, a.end)}
			i = next
		}
		if !movementBoundary(r.word(i)) {
			return m, false
		}
		m.Rule = "group-motion-passive-v2"
		m.PredicateSources = []Anchor{tokenAnchor(verb, participle+1), tokenAnchor(particle, particleEnd)}
		m.Participants = append([]MovementParticipant{leader}, followers...)
		for pi := 1; pi < len(m.Participants); pi++ {
			ls := anchor(loc.start, loc.end)
			m.Participants[pi].Origin = &MovementOrigin{Place: loc.place, Status: "explicit", Source: &ls}
		}
		// The origin phrase modifies the transported follower. It does not
		// independently locate the passive agent (which may guide remotely).
		m.Complete = r.word(i) != "," && (i == len(r.tokens) || i+1 == len(r.tokens))
		return m, true
	}
	// A postposed agent attaches only to an explicit physical departure, not
	// to any earlier noun or another finite clause in the paragraph.
	switch r.word(i) {
	case "ran", "walked", "marched", "fled":
	default:
		return m, false
	}
	i++
	particle := i
	if r.word(i) != "from" {
		return m, false
	}
	loc, end, found := r.location(i+1, true)
	if !found {
		return m, false
	}
	i = end
	if r.word(i) != "," {
		return m, false
	}
	i++
	// Optional quantified formation adjunct: CD + spatial PP. No finite
	// predicates or named participants may be skipped to find a later agent.
	if i < len(r.tokens) && r.tokens[i].Tag == "CD" {
		_, next, ok := r.location(i+1, false)
		if !ok || r.word(next) != "," {
			return m, false
		}
		i = next + 1
	}
	agentVerb := i
	if !traversalVerb(r.word(i)) || r.word(i+1) != "by" {
		return m, false
	}
	leader, next, ok := r.actor(i + 2)
	if !ok || !movementBoundary(r.word(next)) {
		return m, false
	}
	m.Rule = "group-motion-postposed-v2"
	m.PredicateSources = []Anchor{tokenAnchor(verb, particle+1), tokenAnchor(agentVerb, agentVerb+2)}
	m.Participants = append([]MovementParticipant{{Subject: leader.id, Role: "leader", IdentityStatus: "explicit", Source: anchor(leader.start, leader.end)}}, followers...)
	for pi := range m.Participants {
		ls := anchor(loc.start, loc.end)
		m.Participants[pi].Origin = &MovementOrigin{Place: loc.place, Status: "explicit", Source: &ls}
	}
	m.Complete = r.word(next) != "," && (next == len(r.tokens) || next+1 == len(r.tokens))
	return m, true
}

package dialogue

import (
	"sort"
	"strings"
)

// ── Roster matching ──────────────────────────────────────────────────────────

type matcher struct {
	key     string
	aliases []string // longest first; matched case-sensitively
}

func buildMatchers(roster []RosterEntry) []matcher {
	matchers := make([]matcher, 0, len(roster))
	for _, entry := range roster {
		seen := map[string]bool{}
		var aliases []string
		for _, alias := range append([]string{entry.Name}, entry.Aliases...) {
			alias = strings.TrimSpace(alias)
			if len([]rune(alias)) > 1 && !seen[alias] {
				seen[alias] = true
				aliases = append(aliases, alias)
			}
		}
		sort.Slice(aliases, func(a, b int) bool { return len(aliases[a]) > len(aliases[b]) })
		matchers = append(matchers, matcher{key: entry.Key, aliases: aliases})
	}
	return matchers
}

type rosterHit struct {
	key    string
	index  int
	length int
}

// The frontend matcher uses JS \W, which is ASCII: letters, digits and
// underscore are word characters, everything else is a boundary.
func isWordByte(b byte) bool {
	return b == '_' || (b >= '0' && b <= '9') || (b >= 'A' && b <= 'Z') || (b >= 'a' && b <= 'z')
}

func findRosterHits(text string, matchers []matcher) []rosterHit {
	var hits []rosterHit
	for _, m := range matchers {
		for _, alias := range m.aliases {
			from := 0
			for {
				rel := strings.Index(text[from:], alias)
				if rel < 0 {
					break
				}
				index := from + rel
				from = index + 1
				if index > 0 && isWordByte(text[index-1]) {
					continue
				}
				if end := index + len(alias); end < len(text) && isWordByte(text[end]) {
					continue
				}
				// Longest-alias-first within a matcher: skip shorter aliases
				// landing inside an already-recorded hit for the same
				// character.
				covered := false
				for _, h := range hits {
					if h.key == m.key && index >= h.index && index < h.index+h.length {
						covered = true
						break
					}
				}
				if !covered {
					hits = append(hits, rosterHit{key: m.key, index: index, length: len(alias)})
				}
			}
		}
	}
	sort.Slice(hits, func(a, b int) bool { return hits[a].index < hits[b].index })
	return hits
}

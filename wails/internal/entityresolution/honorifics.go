package entityresolution

import (
	"regexp"
	"strings"
)

// DefaultHonorifics is the default set of honorifics/titles to strip from names.
// These are stored lowercase for case-insensitive matching.
var DefaultHonorifics = map[string]bool{
	// Common titles
	"mr":        true,
	"mrs":       true,
	"ms":        true,
	"miss":      true,
	"mister":    true,
	"missus":    true,
	"madam":     true,
	"madame":    true,
	"dr":        true,
	"prof":      true,
	"professor": true,
	// Military ranks (including compound-rank modifiers: Staff Sgt., Master Sgt.)
	"private":   true,
	"corporal":  true,
	"cpl":       true,
	"sergeant":  true,
	"sgt":       true,
	"staff":     true,
	"gunnery":   true,
	"lance":     true,
	"petty":     true,
	"warrant":   true,
	"master":    true,
	"lieutenant": true,
	"lt":        true,
	"captain":   true,
	"capt":      true,
	"major":     true,
	"maj":       true,
	"colonel":   true,
	"col":       true,
	"general":   true,
	"gen":       true,
	"admiral":   true,
	"adm":       true,
	"commander": true,
	"cmdr":      true,
	"airman":    true,
	"seaman":    true,
	"ensign":    true,
	"cadet":     true,
	"marshal":   true,
	"commissioner": true,
	// Law enforcement
	"officer":   true,
	"detective": true,
	"det":       true,
	"inspector": true,
	"agent":     true,
	"sheriff":   true,
	"deputy":    true,
	"constable": true,
	"chief":     true,
	// Nobility/formal
	"lord":      true,
	"lady":      true,
	"sir":       true,
	"dame":      true,
	"king":      true,
	"queen":     true,
	"prince":    true,
	"princess":  true,
	"duke":      true,
	"duchess":   true,
	"baron":     true,
	"baroness":  true,
	"count":     true,
	"countess":  true,
	"earl":      true,
	// Political
	"president":  true,
	"senator":    true,
	"governor":   true,
	"mayor":      true,
	"councilman": true,
	"councilwoman": true,
	"ambassador": true,
	// Religious
	"father":    true,
	"mother":    true,
	"sister":    true,
	"brother":   true,
	"reverend":  true,
	"rev":       true,
	"pastor":    true,
	"bishop":    true,
	"cardinal":  true,
	"pope":      true,
	"rabbi":     true,
	"imam":      true,
	// Family (when used as titles before names)
	"uncle":     true,
	"aunt":      true,
	"grandpa":   true,
	"grandma":   true,
	"grandmother": true,
	"grandfather": true,
	// Medical
	"nurse":     true,
	"doctor":    true,
}

// honorificAbbrevs are ABBREVIATED honorifics ("Dr.", "Sgt."). Only these may
// be followed by a period inside a name span — a full-word honorific ending
// in a period ("...saluted the general.") is a sentence boundary, never part
// of the next name.
var honorificAbbrevs = map[string]bool{
	"mr": true, "mrs": true, "ms": true, "dr": true, "prof": true,
	"sgt": true, "cpl": true, "pvt": true, "lt": true, "capt": true,
	"maj": true, "col": true, "gen": true, "adm": true, "cmdr": true,
	"det": true, "insp": true, "supt": true, "rev": true, "fr": true,
	"st": true,
}

// IsHonorificAbbrev reports whether a word is an abbreviated honorific.
func IsHonorificAbbrev(word string) bool {
	return honorificAbbrevs[strings.ToLower(strings.TrimSuffix(word, "."))]
}

// honorificGroup assigns professional/gender groups to honorifics. Two names
// whose titles fall in DIFFERENT groups refer to different people
// (Dr. Chen is never Staff Sgt. Chen). Titles with no group are neutral.
var honorificGroup = map[string]string{
	// Academic / medical professionals
	"dr": "academic", "doctor": "academic", "prof": "academic", "professor": "academic",
	"nurse": "medical",
	// Uniformed services (military and police share ranks like Sgt/Captain)
	"private": "service", "corporal": "service", "cpl": "service",
	"sergeant": "service", "sgt": "service", "staff": "service",
	"gunnery": "service", "lance": "service", "petty": "service",
	"warrant": "service", "lieutenant": "service", "lt": "service",
	"captain": "service", "capt": "service", "major": "service", "maj": "service",
	"colonel": "service", "col": "service", "general": "service", "gen": "service",
	"admiral": "service", "adm": "service", "commander": "service", "cmdr": "service",
	"airman": "service", "seaman": "service", "ensign": "service", "cadet": "service",
	"officer": "service", "detective": "service", "det": "service",
	"inspector": "service", "insp": "service", "supt": "service",
	"sheriff": "service", "deputy": "service", "constable": "service",
	"agent": "service", "marshal": "service", "commissioner": "service",
	// Religious
	"reverend": "religious", "rev": "religious", "pastor": "religious",
	"bishop": "religious", "cardinal": "religious", "pope": "religious",
	"rabbi": "religious", "imam": "religious",
	// Royalty / nobility
	"king": "royalty", "queen": "royalty", "prince": "royalty", "princess": "royalty",
	"duke": "royalty", "duchess": "royalty", "baron": "royalty", "baroness": "royalty",
	"count": "royalty", "countess": "royalty", "earl": "royalty",
	// Political office
	"president": "political", "senator": "political", "governor": "political",
	"mayor": "political", "councilman": "political", "councilwoman": "political",
	"ambassador": "political",
	// Gendered civilian address
	"mr": "male", "mister": "male",
	"mrs": "female", "ms": "female", "miss": "female", "missus": "female",
	"madam": "female", "madame": "female",
}

// TitleGroups returns the set of groups the given titles belong to.
func TitleGroups(titles []string) map[string]bool {
	groups := map[string]bool{}
	for _, t := range titles {
		if g, ok := honorificGroup[strings.ToLower(strings.TrimSuffix(t, "."))]; ok {
			groups[g] = true
		}
	}
	return groups
}

// GroupsCompatible reports whether two title-group sets can belong to the
// same person. An empty set is compatible with anything; two non-empty sets
// must not contain differing groups.
func GroupsCompatible(a, b map[string]bool) bool {
	for ga := range a {
		for gb := range b {
			if ga != gb {
				return false
			}
		}
	}
	return true
}

// honorificPattern matches a leading honorific followed by optional period.
var honorificPattern = regexp.MustCompile(`^\s*(\S+)\.?\s+`)

// StripHonorifics removes leading honorifics from a name and returns the
// normalized name head along with any extracted titles.
// Example: "Officer Ruiz" → head: "Ruiz", titles: ["Officer"]
// Example: "Dr. Carlos Ruiz" → head: "Carlos Ruiz", titles: ["Dr"]
func StripHonorifics(name string, honorifics map[string]bool) (head string, titles []string) {
	if honorifics == nil {
		honorifics = DefaultHonorifics
	}

	name = strings.TrimSpace(name)
	titles = []string{}

	// Keep stripping leading honorifics
	for {
		match := honorificPattern.FindStringSubmatch(name)
		if match == nil {
			break
		}

		word := match[1]
		wordLower := strings.ToLower(strings.TrimSuffix(word, "."))

		if honorifics[wordLower] {
			titles = append(titles, word)
			name = strings.TrimSpace(name[len(match[0]):])
		} else {
			break
		}
	}

	head = name
	return
}

// TokenizeName splits a name into tokens for matching.
// Example: "Carlos Ruiz" → ["Carlos", "Ruiz"]
// Example: "Mary-Jane Watson" → ["Mary-Jane", "Watson"]
func TokenizeName(name string) []string {
	// Split on whitespace
	parts := strings.Fields(name)
	tokens := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			tokens = append(tokens, p)
		}
	}
	return tokens
}

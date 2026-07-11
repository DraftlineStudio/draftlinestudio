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
	"dr":        true,
	"prof":      true,
	"professor": true,
	// Military ranks
	"private":   true,
	"corporal":  true,
	"sergeant":  true,
	"sgt":       true,
	"lieutenant": true,
	"lt":        true,
	"captain":   true,
	"capt":      true,
	"major":     true,
	"colonel":   true,
	"col":       true,
	"general":   true,
	"gen":       true,
	"admiral":   true,
	"commander": true,
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

// NormalizeToken normalizes a single name token for comparison.
// Lowercases and removes punctuation except hyphens.
func NormalizeToken(token string) string {
	var sb strings.Builder
	for _, r := range strings.ToLower(token) {
		if (r >= 'a' && r <= 'z') || r == '-' || r == '\'' {
			sb.WriteRune(r)
		}
	}
	return sb.String()
}

// GetNameHead returns the "head" of a name - typically the last token (surname).
// For single-token names, returns that token.
// Example: "Carlos Ruiz" → "Ruiz"
// Example: "Ruiz" → "Ruiz"
func GetNameHead(tokens []string) string {
	if len(tokens) == 0 {
		return ""
	}
	return tokens[len(tokens)-1]
}

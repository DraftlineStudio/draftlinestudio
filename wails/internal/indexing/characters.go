package indexing

import (
	"regexp"
	"strings"
)

// StripHTML removes HTML tags from content.
func StripHTML(html string) string {
	// Simple regex to remove HTML tags
	re := regexp.MustCompile(`<[^>]*>`)
	text := re.ReplaceAllString(html, " ")
	// Clean up whitespace
	text = regexp.MustCompile(`\s+`).ReplaceAllString(text, " ")
	return strings.TrimSpace(text)
}

// DetectCharacterNames extracts potential character names from text.
// Uses quote-aware dialogue patterns to minimize false positives.
// Errs on the side of false negatives - users can add missed characters manually.
func DetectCharacterNames(text string) map[string]int {
	mentions := make(map[string]int)

	// QUOTE-AWARE DIALOGUE PATTERNS
	// These are the most reliable indicators of character names in fiction

	// Pattern 1: Speaker AFTER closing quote: '," said John' or '." John said'
	// Matches: "Hello," said John. | "Hello." John replied.
	afterQuote := regexp.MustCompile(`[,.]"\s*(?i:` + DialogueVerbs + `)\s+([A-Z][a-z]{2,})\b`)
	afterQuote2 := regexp.MustCompile(`[,.]"\s*([A-Z][a-z]{2,})\s+(?i:` + DialogueVerbs + `)`)

	// Pattern 2: Speaker BEFORE opening quote: 'John said, "' or 'John said "'
	// Matches: John said, "Hello" | John replied "Hi"
	beforeQuote := regexp.MustCompile(`\b([A-Z][a-z]{2,})\s+(?i:` + DialogueVerbs + `)\s*,?\s*"`)

	// Pattern 3: Title + Name (Mr. Smith, Dr. Jones, Captain Kirk)
	// These are very reliable indicators
	afterTitle := regexp.MustCompile(`\b(Mr|Mrs|Ms|Miss|Dr|Prof|Professor|Captain|Colonel|General|Lieutenant|Sergeant|Officer|Detective|Agent|Lord|Lady|Sir|Dame|King|Queen|Prince|Princess|Senator|Governor|Mayor|Chief|Father|Mother|Sister|Brother|Uncle|Aunt)\.?\s+([A-Z][a-z]{2,})\b`)

	// Find speakers after closing quotes
	for _, match := range afterQuote.FindAllStringSubmatch(text, -1) {
		name := match[1]
		if !CommonWords[name] && !LooksLikeCommonWord(name) {
			mentions[name] += 3
		}
	}
	for _, match := range afterQuote2.FindAllStringSubmatch(text, -1) {
		name := match[1]
		if !CommonWords[name] && !LooksLikeCommonWord(name) {
			mentions[name] += 3
		}
	}

	// Find speakers before opening quotes
	for _, match := range beforeQuote.FindAllStringSubmatch(text, -1) {
		name := match[1]
		if !CommonWords[name] && !LooksLikeCommonWord(name) {
			mentions[name] += 3
		}
	}

	// Find names after titles
	for _, match := range afterTitle.FindAllStringSubmatch(text, -1) {
		name := match[2]
		if !LooksLikeCommonWord(name) {
			mentions[name] += 2
		}
	}

	return mentions
}

// LooksLikeCommonWord checks if a word has suffixes/patterns typical of
// common English words rather than names (adverbs, gerunds, etc.)
func LooksLikeCommonWord(word string) bool {
	lower := strings.ToLower(word)

	for _, suffix := range CommonWordSuffixes {
		if strings.HasSuffix(lower, suffix) && len(lower) > len(suffix)+2 {
			return true
		}
	}

	return TemporalWords[lower]
}

// ExtractAttributes looks for character attributes in surrounding context.
func ExtractAttributes(text, name string) map[string]string {
	attrs := make(map[string]string)
	nameLower := strings.ToLower(name)
	textLower := strings.ToLower(text)

	// Patterns for attribute extraction
	// Eye color: "Kira's blue eyes", "her green eyes", "eyes were brown"
	eyeColors := []string{"blue", "green", "brown", "hazel", "gray", "grey", "black", "amber", "violet", "golden"}
	for _, color := range eyeColors {
		patterns := []string{
			nameLower + `'s\s+` + color + `\s+eyes?`,
			nameLower + `\s+.*\b` + color + `\s+eyes?`,
			`\b` + color + `\s+eyes?.*` + nameLower,
		}
		for _, pattern := range patterns {
			if matched, _ := regexp.MatchString(pattern, textLower); matched {
				attrs["eye_color"] = color
				break
			}
		}
	}

	// Hair color
	hairColors := []string{"blonde", "blond", "brunette", "brown", "black", "red", "auburn", "gray", "grey", "white", "silver", "golden", "dark", "light"}
	for _, color := range hairColors {
		patterns := []string{
			nameLower + `'s\s+` + color + `\s+hair`,
			nameLower + `\s+.*\b` + color + `\s+hair`,
			`\b` + color + `\s+hair.*` + nameLower,
		}
		for _, pattern := range patterns {
			if matched, _ := regexp.MatchString(pattern, textLower); matched {
				attrs["hair_color"] = color
				break
			}
		}
	}

	// Age patterns: "twenty-year-old", "aged 30", "30 years old"
	agePattern := regexp.MustCompile(`\b(\d{1,2})[\s-]?year[\s-]?old\b`)
	if matches := agePattern.FindStringSubmatch(textLower); len(matches) > 1 {
		attrs["age"] = matches[1]
	}

	return attrs
}

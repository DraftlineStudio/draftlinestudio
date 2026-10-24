package indexing

import (
	"strings"
	"unicode"
)

var actorVerbs = map[string]bool{
	"said": true, "asked": true, "replied": true, "answered": true,
	"whispered": true, "muttered": true, "shouted": true,
	"nodded": true, "shrugged": true, "smiled": true, "frowned": true,
	"laughed": true, "wept": true, "grinned": true, "sighed": true,
}

func hasActorContext(text string, end int) bool {
	tail := strings.TrimLeft(text[end:], " \t")
	if len(tail) == len(text[end:]) {
		return false
	}
	stop := len(tail)
	for i, r := range tail {
		if !unicode.IsLetter(r) {
			stop = i
			break
		}
	}
	return actorVerbs[strings.ToLower(tail[:stop])]
}

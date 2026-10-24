package indexing

import (
	"strings"

	"draftline/internal/entityresolution"
)

func spellingAliases(canonical string, aliases []string) []string {
	head, _ := entityresolution.StripHonorifics(canonical, entityresolution.DefaultHonorifics)
	tokens := entityresolution.TokenizeName(head)
	suspect := map[string]bool{}
	for _, alias := range aliases {
		aliasHead, _ := entityresolution.StripHonorifics(alias, entityresolution.DefaultHonorifics)
		other := entityresolution.TokenizeName(aliasHead)
		if len(tokens) < 2 || len(other) != len(tokens) {
			continue
		}
		samePrefix := true
		for i := 0; i < len(tokens)-1; i++ {
			if !strings.EqualFold(tokens[i], other[i]) {
				samePrefix = false
				break
			}
		}
		last := other[len(other)-1]
		if samePrefix && !strings.EqualFold(tokens[len(tokens)-1], last) && entityresolution.IsLikelyTypo(tokens[len(tokens)-1], last) {
			suspect[strings.ToLower(last)] = true
		}
	}
	safe := []string{}
	for _, alias := range aliases {
		aliasHead, _ := entityresolution.StripHonorifics(alias, entityresolution.DefaultHonorifics)
		blocked := false
		for _, token := range entityresolution.TokenizeName(aliasHead) {
			if suspect[strings.ToLower(token)] {
				blocked = true
				break
			}
		}
		if !blocked {
			safe = append(safe, alias)
		}
	}
	return safe
}

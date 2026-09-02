package fingerprint

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"

	"draftline/internal/types"
)

func stableID(kind string, parts ...string) string {
	hash := sha256.Sum256([]byte(strings.Join(parts, "\x00")))
	return kind + "-" + hex.EncodeToString(hash[:8])
}

func reconcileCorrections(model *types.StoryFingerprint) {
	known := map[string]bool{}
	for _, event := range model.Events {
		known[event.ID] = true
	}
	for _, context := range model.Contexts {
		known[context.ID] = true
	}
	for index := range model.AuthorModel.Corrections {
		correction := &model.AuthorModel.Corrections[index]
		if known[correction.TargetID] {
			correction.Status = "active"
		} else {
			correction.Status = "orphaned"
		}
	}
}

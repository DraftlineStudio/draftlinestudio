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

func applyEvidenceContextCorrections(corrections []types.FingerprintCorrection, contexts map[string]string) {
	for _, correction := range corrections {
		if correction.Kind != "context" || correction.Value == "" {
			continue
		}
		for _, evidenceID := range correction.EvidenceIDs {
			if _, exists := contexts[evidenceID]; exists {
				contexts[evidenceID] = correction.Value
			}
		}
	}
}

package fingerprint

// Event identity resolution (v5). Landing in an upcoming build: frames are
// clustered into likely same events from structured properties — event
// class, participants, scope, time — never from shared vocabulary, with
// conflicting accounts preserved as conflicting property values.

import "draftline/internal/types"

func resolveEventIdentities(frames []types.NarrativeFrame) []types.NarrativeEventIdentity {
	return nil
}

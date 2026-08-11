package entityresolution

import "strings"

// LevenshteinDistance computes the Levenshtein (edit) distance between two strings.
// This is the minimum number of single-character edits (insertions, deletions,
// or substitutions) required to transform one string into the other.
func LevenshteinDistance(s1, s2 string) int {
	// Normalize to lowercase for comparison
	s1 = strings.ToLower(s1)
	s2 = strings.ToLower(s2)

	// Early exit for identical strings
	if s1 == s2 {
		return 0
	}

	len1 := len(s1)
	len2 := len(s2)

	// Early exit if one is empty
	if len1 == 0 {
		return len2
	}
	if len2 == 0 {
		return len1
	}

	// Create two rows for the dynamic programming table
	// We only need the previous row and current row
	prev := make([]int, len2+1)
	curr := make([]int, len2+1)

	// Initialize first row
	for j := 0; j <= len2; j++ {
		prev[j] = j
	}

	// Fill in the table
	for i := 1; i <= len1; i++ {
		curr[0] = i

		for j := 1; j <= len2; j++ {
			cost := 1
			if s1[i-1] == s2[j-1] {
				cost = 0
			}

			// Minimum of:
			// - deletion (curr[j-1] + 1)
			// - insertion (prev[j] + 1)
			// - substitution (prev[j-1] + cost)
			curr[j] = min(curr[j-1]+1, min(prev[j]+1, prev[j-1]+cost))
		}

		// Swap rows
		prev, curr = curr, prev
	}

	return prev[len2]
}

// IsTypo checks if two strings are within the allowed edit distance for typos.
// By default, allows edit distance of 1 for strings of length 4+.
// For shorter strings, requires exact match to avoid false positives.
func IsTypo(s1, s2 string, maxDistance int) bool {
	s1 = strings.ToLower(s1)
	s2 = strings.ToLower(s2)

	// Exact match is always true
	if s1 == s2 {
		return true
	}

	// For short names (< 4 chars), don't allow typos (too many false positives)
	// e.g., "Joe" vs "Jon" should not merge
	if len(s1) < 4 || len(s2) < 4 {
		return false
	}

	// Length difference check - if lengths differ by more than maxDistance,
	// they can't be within edit distance
	lenDiff := len(s1) - len(s2)
	if lenDiff < 0 {
		lenDiff = -lenDiff
	}
	if lenDiff > maxDistance {
		return false
	}

	return LevenshteinDistance(s1, s2) <= maxDistance
}

// IsLikelyTypo is a stricter version that also checks:
// - First letter must match (common typos don't change the first letter)
// - Length difference of at most 1
func IsLikelyTypo(s1, s2 string) bool {
	s1 = strings.ToLower(s1)
	s2 = strings.ToLower(s2)

	if s1 == s2 {
		return true
	}

	// Must have same first letter
	if len(s1) == 0 || len(s2) == 0 || s1[0] != s2[0] {
		return false
	}

	// For short names, require exact match
	if len(s1) < 4 || len(s2) < 4 {
		return false
	}

	// Length difference of at most 1
	lenDiff := len(s1) - len(s2)
	if lenDiff < 0 {
		lenDiff = -lenDiff
	}
	if lenDiff > 1 {
		return false
	}

	return LevenshteinDistance(s1, s2) <= 1
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

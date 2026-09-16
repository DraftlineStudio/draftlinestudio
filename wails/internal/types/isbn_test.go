package types

import "testing"

// Real published ISBNs are used as fixtures because a check digit is only
// meaningful against numbers that actually check out. They identify no
// manuscript and carry no text.
func TestValidISBNAcceptsWellFormedNumbersAndRejectsMistypes(t *testing.T) {
	valid := []string{
		"978-0-306-40615-7", // the canonical ISBN-13 worked example
		"9780306406157",
		"0-306-40615-2", // its ISBN-10 form
		"0306406152",
		"080442957X", // an ISBN-10 whose check digit is X
	}
	for _, value := range valid {
		if !ValidISBN(value) {
			t.Errorf("%q should be a valid ISBN", value)
		}
	}

	invalid := []string{
		"",
		"978-0-306-40615-8", // one digit off in the checksum
		"0306406153",
		"97803064061",      // too short
		"97803064061570",   // too long
		"978030640615X",    // X is only legal as an ISBN-10 check digit
		"not-a-number-abc", // no digits at all
	}
	for _, value := range invalid {
		if ValidISBN(value) {
			t.Errorf("%q should not be a valid ISBN", value)
		}
	}
}

func TestISBN13AndISBN10ConvertBothWays(t *testing.T) {
	if got := ISBN13("0-306-40615-2"); got != "9780306406157" {
		t.Errorf("ISBN-10 to ISBN-13: got %q", got)
	}
	if got := ISBN13("978-0-306-40615-7"); got != "9780306406157" {
		t.Errorf("ISBN-13 passes through: got %q", got)
	}
	if got := ISBN10("978-0-306-40615-7"); got != "0306406152" {
		t.Errorf("ISBN-13 to ISBN-10: got %q", got)
	}
	// A 979 number has no ISBN-10 at all. Saying so is the right answer;
	// inventing one would be worse than an empty field.
	if got := ISBN10("9791234567896"); got != "" {
		t.Errorf("a 979 ISBN has no ISBN-10, got %q", got)
	}
	if got := ISBN13("nonsense"); got != "" {
		t.Errorf("a non-ISBN converts to nothing, got %q", got)
	}
}

func TestHyphenateIsReadableAndLeavesOtherValuesAlone(t *testing.T) {
	// Prefix, group digit, body, check digit. Not a registrant-accurate
	// grouping, and not presented as one.
	if got := Hyphenate("9780306406157"); got != "978-0-30640615-7" {
		t.Errorf("Hyphenate: got %q", got)
	}
	if got := Hyphenate("0306406152"); got != "0306406152" {
		t.Errorf("a non-13 value is returned untouched, got %q", got)
	}
}

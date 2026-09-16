package types

import "strings"

// ISBN checking and conversion. An ISBN is not just a number: its last digit
// is a checksum over the ones before it, so a mistyped digit is detectable
// rather than silently wrong. Draftline never invents or assigns an ISBN; it
// only tells the author whether the one they typed can be an ISBN at all.

// NormalizeISBN strips the separators people type and upper-cases the X that
// can stand in for the check digit of an ISBN-10. It does not validate.
func NormalizeISBN(value string) string {
	var b strings.Builder
	for _, r := range value {
		switch {
		case r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == 'x' || r == 'X':
			b.WriteRune('X')
		}
	}
	return b.String()
}

// ValidISBN reports whether a value is a well-formed ISBN-13 or ISBN-10,
// check digit included. An empty value is not an ISBN and is not an error
// either; callers decide what a blank field means.
func ValidISBN(value string) bool {
	digits := NormalizeISBN(value)
	switch len(digits) {
	case 13:
		return validISBN13(digits)
	case 10:
		return validISBN10(digits)
	default:
		return false
	}
}

// ISBN13 is the ISBN-13 form of a value that is already one, or the
// converted form of an ISBN-10. It returns "" when the value is neither.
func ISBN13(value string) string {
	digits := NormalizeISBN(value)
	switch {
	case len(digits) == 13 && validISBN13(digits):
		return digits
	case len(digits) == 10 && validISBN10(digits):
		body := "978" + digits[:9]
		return body + string(checkDigit13(body))
	}
	return ""
}

// ISBN10 is the ISBN-10 form of an ISBN-13 in the 978 range, which is the
// only range that converts. A 979 ISBN has no ISBN-10 and returns "", which
// is a fact about the number rather than a failure.
func ISBN10(value string) string {
	digits := NormalizeISBN(value)
	if len(digits) == 10 && validISBN10(digits) {
		return digits
	}
	if len(digits) != 13 || !validISBN13(digits) || !strings.HasPrefix(digits, "978") {
		return ""
	}
	body := digits[3:12]
	return body + string(checkDigit10(body))
}

// Hyphenate puts an ISBN-13 back into the shape a reader expects. Correct
// hyphenation needs the registration-group ranges, which Draftline does not
// carry, so this only splits off the prefix and the check digit: enough to
// read, never presented as a registrant-accurate grouping.
func Hyphenate(value string) string {
	digits := NormalizeISBN(value)
	if len(digits) != 13 {
		return value
	}
	return digits[:3] + "-" + digits[3:4] + "-" + digits[4:12] + "-" + digits[12:]
}

func validISBN13(d string) bool {
	sum := 0
	for i, r := range d {
		if r < '0' || r > '9' {
			return false
		}
		n := int(r - '0')
		if i%2 == 1 {
			n *= 3
		}
		sum += n
	}
	return sum%10 == 0
}

func validISBN10(d string) bool {
	sum := 0
	for i, r := range d {
		var n int
		switch {
		case r >= '0' && r <= '9':
			n = int(r - '0')
		case r == 'X' && i == 9:
			n = 10
		default:
			return false
		}
		sum += n * (10 - i)
	}
	return sum%11 == 0
}

// checkDigit13 is the final digit of a 12-digit ISBN-13 body.
func checkDigit13(body string) byte {
	sum := 0
	for i, r := range body {
		n := int(r - '0')
		if i%2 == 1 {
			n *= 3
		}
		sum += n
	}
	return byte('0' + (10-sum%10)%10)
}

// checkDigit10 is the final character of a 9-digit ISBN-10 body, which is
// 'X' when the checksum is 10.
func checkDigit10(body string) byte {
	sum := 0
	for i, r := range body {
		sum += int(r-'0') * (10 - i)
	}
	switch check := (11 - sum%11) % 11; check {
	case 10:
		return 'X'
	default:
		return byte('0' + check)
	}
}

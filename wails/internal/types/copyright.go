package types

import (
	"strconv"
	"strings"
)

// The copyright page, written from the record rather than typed.
//
// Every fact on a copyright page is already somewhere else in the project: the
// title, the holder, the year, the edition, the imprint, the ISBN of the
// format in hand. Typing them again is how a book ends up with the paperback's
// ISBN on the ebook's copyright page. So the page is generated, and the fields
// are the only place the author edits.
//
// This is the Go half of the generator. The frontend half, which draws the
// same page live as the author types, is
// frontend/src/components/dialogs/editionModel.ts. Both are checked against
// one shared table of worked examples — internal/types/testdata/
// copyright_cases.json — so the preview on screen and the page in the exported
// book cannot drift apart.

// CopyrightLines builds the copyright page for one format of one edition.
//
// priorYears are the copyright years of the editions this one supersedes,
// oldest first; EditionIndex.PriorYears walks them. They are passed in rather
// than looked up so that this stays a pure function of the three records the
// page actually prints from.
//
// A line whose substance is missing is left out rather than printed empty: a
// book with no ISBN yet gets a copyright page without an ISBN line, not one
// reading "ISBN  (paperback)".
func CopyrightLines(meta Metadata, edition Edition, format EditionFormat, priorYears []string) []string {
	var head []string
	if title := strings.TrimSpace(meta.Title); title != "" {
		head = append(head, title)
	}
	if holder := copyrightHolder(meta); holder != "" {
		years := copyrightYears(priorYears, edition.Year)
		if years != "" {
			head = append(head, "Copyright © "+years+" by "+holder)
		}
	}
	head = append(head, RightsSentence(format))

	var body []string
	if statement := editionStatement(edition, format); statement != "" {
		body = append(body, statement)
	}
	if imprint := copyrightImprint(meta, format); imprint != "" {
		body = append(body, "Published by "+imprint)
	}
	if isbn := strings.TrimSpace(format.ISBN13); isbn != "" {
		body = append(body, "ISBN "+isbn+" ("+strings.ToLower(strings.TrimSpace(format.Format))+")")
	}
	if lccn := strings.TrimSpace(format.LCCN); lccn != "" {
		body = append(body, "Library of Congress Control Number: "+lccn)
	}
	// A revision note belongs on a later edition. Printing "Original release."
	// under a first edition's own ISBN says nothing a reader did not know.
	if note := strings.TrimSpace(edition.RevisionNote); note != "" && strings.TrimSpace(edition.PreviousEditionID) != "" {
		body = append(body, note)
	}

	switch {
	case len(head) == 0:
		return body
	case len(body) == 0:
		return head
	default:
		lines := make([]string, 0, len(head)+len(body)+1)
		lines = append(lines, head...)
		lines = append(lines, "")
		return append(lines, body...)
	}
}

// copyrightHolder is who the copyright line is made out to. It is not always
// the author, but when nobody has said otherwise it is.
func copyrightHolder(meta Metadata) string {
	if holder := strings.TrimSpace(meta.CopyrightHolder); holder != "" {
		return holder
	}
	return strings.TrimSpace(meta.Author)
}

// copyrightYears is the cumulative year list a copyright line carries: every
// year an edition of this book established, in order, with repeats dropped.
// A second edition published in 2030 off a first edition of 2026 prints
// "2026, 2030", which is what the copyright in the earlier text still runs
// from.
func copyrightYears(prior []string, year string) string {
	seen := map[string]bool{}
	var years []string
	for _, candidate := range append(append([]string{}, prior...), year) {
		candidate = strings.TrimSpace(candidate)
		if candidate == "" || seen[candidate] {
			continue
		}
		seen[candidate] = true
		years = append(years, candidate)
	}
	return strings.Join(years, ", ")
}

// RightsSentence is the rights line, punctuated. The record holds the notice
// as a phrase ("All rights reserved"), because that is how a rights list reads
// in a dropdown; the page needs it as a sentence, and so does the dc:rights an
// exported package declares.
func RightsSentence(format EditionFormat) string {
	notice := strings.TrimSpace(format.RightsNotice)
	if notice == "" {
		notice = "All rights reserved"
	}
	if strings.HasSuffix(notice, ".") {
		return notice
	}
	return notice + "."
}

// editionStatement is the edition line with its month and year: "First
// edition, April 2026".
//
// The words come from the edition's own label, which is what the copyright
// page of a printed book says. A format's edition_statement is the longer
// wording set on the title page ("Second edition, revised") and stands in only
// when an edition has no label of its own.
func editionStatement(edition Edition, format EditionFormat) string {
	statement := strings.TrimSpace(edition.Label)
	if statement == "" {
		statement = strings.TrimSpace(format.EditionStatement)
	}
	if statement == "" {
		return ""
	}
	if when := monthAndYear(format.PublicationDate); when != "" {
		return statement + ", " + when
	}
	return statement
}

// copyrightImprint is the line the book is published under: the format's own
// imprint of record when it has one, otherwise the book's imprint, otherwise
// its publisher.
func copyrightImprint(meta Metadata, format EditionFormat) string {
	for _, candidate := range []string{format.ImprintOfRecord, meta.Imprint, meta.Publisher} {
		if trimmed := strings.TrimSpace(candidate); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

var copyrightMonths = [...]string{
	"January", "February", "March", "April", "May", "June",
	"July", "August", "September", "October", "November", "December",
}

// monthAndYear turns a stored publication date into the words a copyright page
// prints. Dates are stored as an ISO day ("2026-04-14") because that is what a
// date field yields; a page that is not sure of the month prints just the year
// rather than guessing one.
func monthAndYear(date string) string {
	date = strings.TrimSpace(date)
	if len(date) < 4 {
		return ""
	}
	year := date[:4]
	if _, err := strconv.Atoi(year); err != nil {
		return ""
	}
	if len(date) < 7 || date[4] != '-' {
		return year
	}
	month, err := strconv.Atoi(date[5:7])
	if err != nil || month < 1 || month > 12 {
		return year
	}
	return copyrightMonths[month-1] + " " + year
}

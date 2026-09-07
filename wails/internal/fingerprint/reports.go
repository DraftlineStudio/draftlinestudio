package fingerprint

// Human-readable text diagnostics for the manuscript-memory model. These are
// the primary acceptance surface for v5: the frame report shows exactly what
// the engine remembers (and where it abstained), the development report
// should read as a recognizable account of the story, and the inspection
// report lists continuity findings with their scope assessment.

import (
	"fmt"
	"strings"

	"draftline/internal/types"
)

func reportHeader(title string, model *types.StoryFingerprint) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s\nEngine: %s (schema %d)\n", title, model.Engine, model.Version)
	s := model.CorpusStats
	fmt.Fprintf(&b, "Evidence atoms: %d\nFrames: %d (abstentions recorded on %d)\nLedgers: %d\nEvent identities: %d\nDevelopments: %d\nInspections: %d\n\n",
		s.EvidenceAtoms, s.Frames, s.Abstained, s.Ledgers, s.EventIdentities, s.Developments, s.Inspections)
	return b.String()
}

func writeSpans(b *strings.Builder, indent string, spans []types.NarrativeEvidenceSpan) {
	for _, span := range spans {
		fmt.Fprintf(b, "%s- chapter %d, paragraph %d, sentence %d [%d:%d]: %q\n",
			indent, span.ChapterIndex+1, span.ParagraphIndex+1, span.SentenceIndex+1,
			span.StartOffset, span.EndOffset, span.Quote)
	}
}

func participantLine(participants []types.NarrativeParticipant) string {
	parts := make([]string, 0, len(participants))
	for _, p := range participants {
		parts = append(parts, fmt.Sprintf("%s=%s", p.Role, p.EntityName))
	}
	return strings.Join(parts, ", ")
}

func buildFrameReport(model *types.StoryFingerprint) string {
	var b strings.Builder
	b.WriteString(reportHeader("MANUSCRIPT MEMORY — FRAMES", model))
	for i, frame := range model.Frames {
		fmt.Fprintf(&b, "%d. [%s] %s\n", i+1, frame.Type, frameStatement(frame))
		fmt.Fprintf(&b, "   ID: %s\n", frame.ID)
		if line := participantLine(frame.Participants); line != "" {
			fmt.Fprintf(&b, "   Participants: %s\n", line)
		}
		if frame.Value != "" {
			fmt.Fprintf(&b, "   Value: %s\n", frame.Value)
		}
		fmt.Fprintf(&b, "   Polarity: %s · Epistemic: %s · Attribution: %s%s\n",
			frame.Polarity, frame.Epistemic, frame.Attribution.Kind, attributionName(frame.Attribution))
		fmt.Fprintf(&b, "   Scope: %s (%s) · Confidence: %.2f\n", frame.Scope.Kind, frame.Scope.Label, frame.Confidence)
		if len(frame.Abstentions) > 0 {
			fmt.Fprintf(&b, "   Abstained: %s\n", strings.Join(frame.Abstentions, "; "))
		}
		b.WriteString("   Evidence:\n")
		writeSpans(&b, "     ", frame.EvidenceSpans)
		b.WriteString("\n")
	}
	if len(model.Frames) == 0 {
		b.WriteString("(no frames)\n")
	}
	b.WriteString("\nSTATE LEDGERS\n\n")
	for _, ledger := range model.Ledgers {
		qualifier := ""
		if ledger.Qualifier != "" {
			qualifier = " · " + ledger.Qualifier
		}
		fmt.Fprintf(&b, "%s — %s%s\n", ledger.EntityName, ledger.Aspect, qualifier)
		for _, entry := range ledger.Entries {
			fmt.Fprintf(&b, "  ch %d ¶ %d: %s %q (%s, %s)\n",
				entry.ChapterIndex+1, entry.ParagraphIndex+1, entry.Operation, entry.Value, entry.Epistemic, entry.Scope.Kind)
		}
	}
	if len(model.Ledgers) == 0 {
		b.WriteString("(no ledgers)\n")
	}
	b.WriteString("\nEVENT IDENTITIES\n\n")
	for _, identity := range model.EventIdentities {
		fmt.Fprintf(&b, "%s [%s] frames=%d status=%s\n", identity.ID, identity.EventClass, len(identity.FrameIDs), identity.Status)
		for _, property := range identity.Properties {
			for _, value := range property.Values {
				fmt.Fprintf(&b, "  %s = %q (%s)\n", property.Name, value.Value, value.Epistemic)
			}
		}
	}
	if len(model.EventIdentities) == 0 {
		b.WriteString("(no event identities)\n")
	}
	return b.String()
}

func frameStatement(frame types.NarrativeFrame) string {
	subject := ""
	for _, p := range frame.Participants {
		if p.Role == "subject" {
			subject = p.EntityName
			break
		}
	}
	payload := frame.Detail
	if payload == "" {
		payload = frame.Value
	}
	if subject == "" {
		return payload
	}
	if payload == "" {
		return subject
	}
	return subject + " — " + payload
}

func attributionName(attribution types.NarrativeAttribution) string {
	if attribution.EntityName == "" {
		return ""
	}
	return " (" + attribution.EntityName + ")"
}

func buildDevelopmentReport(model *types.StoryFingerprint) string {
	var b strings.Builder
	b.WriteString(reportHeader("NARRATIVE DEVELOPMENTS", model))
	for i, development := range model.Developments {
		fmt.Fprintf(&b, "%d. [ch %d] %s: %s\n", i+1, development.ChapterIndex+1, development.Kind, development.Summary)
		if line := participantLine(development.Entities); line != "" {
			fmt.Fprintf(&b, "   Entities: %s\n", line)
		}
		if len(development.Basis) > 0 {
			fmt.Fprintf(&b, "   Basis: %s\n", strings.Join(development.Basis, "; "))
		}
		b.WriteString("\n")
	}
	if len(model.Developments) == 0 {
		b.WriteString("(no developments)\n")
	}
	return b.String()
}

func buildInspectionReport(model *types.StoryFingerprint) string {
	var b strings.Builder
	b.WriteString(reportHeader("CONTINUITY INSPECTIONS", model))
	for i, inspection := range model.Inspections {
		fmt.Fprintf(&b, "%d. [%s/%s] %s\n", i+1, inspection.Severity, inspection.Kind, inspection.Title)
		fmt.Fprintf(&b, "   %s\n   Scope assessment: %s · Confidence: %.2f\n",
			inspection.Detail, inspection.ScopeAssessment, inspection.Confidence)
		for _, side := range inspection.Sides {
			fmt.Fprintf(&b, "   %s:\n", side.Label)
			writeSpans(&b, "     ", side.EvidenceSpans)
		}
		b.WriteString("\n")
	}
	if len(model.Inspections) == 0 {
		b.WriteString("(no inspections)\n")
	}
	return b.String()
}

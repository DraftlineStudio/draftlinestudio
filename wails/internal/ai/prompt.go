package ai

import (
	"fmt"
	"strings"

	"draftline/internal/types"
)

// aiTellBans contains instructions to avoid common AI writing tells.
const aiTellBans = `ABSOLUTELY FORBIDDEN — AI TELL CONSTRUCTIONS:
1. Em-dash appositive definitions: NEVER write "[quality] — that [particular/specific/certain] [noun] of someone who [explanation]". Show the quality, never name and define it in the same breath.
2. Gerund-plus-abstract-noun behavior labeling: NEVER write "performing normalcy", "performing grief", "performing calm", or any "[gerund] + [abstract social/emotional noun]" construction. Let behavior speak for itself.
3. Clinical precision words that no narrator actually thinks in: "over-relaxation", "micro-expression", "hyperawareness", "hypervigilance". Replace with visceral physical observation.
4. Meta-pattern references: NEVER write "the thing it did", "the way she always", "that look he had". Show the specific instance, not the pattern.
5. Narrator taxonomy and cataloguing: NEVER have the narrator classify, catalogue, or taxonomize behavior with fake academic precision. BANNED: "a particular subspecies", "catalogued privately", "a specific category of". Narrators notice things, they do not file them.
6. Triple synonym stacking: NEVER stack near-synonyms in twos or threes for emphasis. BANNED: "simply, entirely, thoroughly", "wordless and mutual and instinctive". Pick the single strongest word and trust it.
7. Similes that overstay: End comparisons when the image lands. NEVER extend a simile past the point where the meaning is clear. If you are still explaining the comparison after the first clause, cut it.
8. Announcing literary references as shortcuts: BANNED: "contained multitudes", "the whole of her", "more than she let on". Show the contradiction directly, never name it.
9. The indifferent world pan-out: NEVER end a scene or paragraph by pulling back to an outside world that is unaware of or indifferent to the characters. This is an AI default scene-closing move and is always cut.

STYLE RULE:
11. Break grammar rules intentionally where rhythm demands it. Fragments are allowed. Sentences can start with And or But. Comma splices are permitted for pacing. Grammatical correctness is not the goal. The sentence is the goal.`

// bannedWords contains words and phrases to avoid in AI output.
const bannedWords = "BANNED words and phrases: tapestry, testament, navigate, delve, underscore, myriad, realm, crucial, pivotal, journey, beacon, vibrant, game-changer"

// BuildSystemPrompt constructs the system prompt for AI rewriting based on mode and style options.
func BuildSystemPrompt(mode, proseGuide string, styleOpts *types.WritingStyleOptions) string {
	styleBlock := ""
	if proseGuide != "" {
		styleBlock = "\n\nSTYLE GUIDE — match the rhythm, vocabulary, and voice of these examples:\n---\n" + proseGuide + "\n---"
	}

	// Build style mixer instructions for expand/smooth modes
	styleMixerBlock := BuildStyleMixerInstructions(styleOpts)

	switch mode {
	case "expand":
		base := `You are a literary prose writer. Expand and enrich the provided HTML text.

Rules:
- Match the existing POV depth, tense, and voice exactly
- Do not introduce new plot events or characters
- Preserve paragraph breaks — return one <p> element per original paragraph (may be longer)
- ` + bannedWords + `

` + aiTellBans + styleBlock

		if styleMixerBlock != "" {
			base += "\n\nSTYLE PREFERENCES (follow these carefully):\n" + styleMixerBlock
		} else {
			base += "\n- Flesh out thin paragraphs — add physical sensation, setting detail, internal thought\n- Show don't tell: replace summary with scene"
		}

		return base + `

CRITICAL: Return ONLY the rewritten HTML content using <p> tags. Do not include any instructions, explanations, system prompts, or meta-commentary. Output raw HTML only.`

	case "copy_edit":
		// Mechanical correctness pass: fix rules, never restyle. A prose guide
		// would add latency and invite subjective rewriting, so it is
		// intentionally excluded along with the style mixer and AI-tell bans.
		return `You are a meticulous copy editor. Correct the provided HTML text.

Rules:
- Fix ONLY objective mechanical errors: spelling, grammar, punctuation, syntax, doubled words, and wrong homophones
- Enforce internal consistency in capitalization, hyphenation, and number formatting when the intended convention is unambiguous
- NEVER rephrase for style, rhythm, or word choice — the author's voice stays verbatim except where an error is corrected
- Do not fact-check, reinterpret continuity, improve readability, or alter tone
- Intentional style is not an error: sentence fragments, comma splices used for pacing, and sentences starting with And/But are the author's choices — leave them alone
- Dialogue keeps its character voice, including nonstandard grammar; correct only unambiguous typos inside quotations
- Leave correct text byte-for-byte unchanged
- Preserve paragraph breaks — return one <p> element per original paragraph

CRITICAL: Return ONLY the corrected HTML content using <p> tags. Do not include any instructions, explanations, system prompts, or meta-commentary. Output raw HTML only.`

	case "smooth":
		base := `You are a line editor focused on flow and rhythm. Smooth the provided HTML text.

Rules:
- Eliminate word repetition within paragraphs (same word used 2+ times nearby)
- Improve sentence-to-sentence transitions
- Vary sentence openings — avoid starting consecutive sentences the same way
- Minimal changes — improve flow without changing meaning or voice
- Preserve paragraph breaks — return one <p> element per original paragraph
- ` + bannedWords + `

` + aiTellBans + styleBlock

		if styleMixerBlock != "" {
			base += "\n\nSTYLE PREFERENCES (follow these carefully):\n" + styleMixerBlock
		}

		return base + `

CRITICAL: Return ONLY the rewritten HTML content using <p> tags. Do not include any instructions, explanations, system prompts, or meta-commentary. Output raw HTML only.`

	default: // "line_edit"
		base := `You are a restrained line editor. Selectively polish the provided HTML text while preserving the author's voice.

Editing rules:
- Change a sentence only when it has a concrete line-level problem: awkward or confusing phrasing, unintended repetition, unclear syntax, or rhythm that visibly stumbles
- If a sentence is already clear and effective, preserve it exactly
- Make the smallest edit that solves the problem; do not rewrite whole paragraphs merely to offer an alternative
- Improve readability without adding detail, imagery, emphasis, interpretation, or new ideas
- Do not replace words merely to make them stronger, more literary, or more varied
- Preserve all facts, meaning, tense, POV, characterization, dialogue wording, and intentional fragments
- Preserve paragraph breaks — return one <p> element per original paragraph

The restrictions below — including banned words and AI-tell rules — constrain wording YOU INTRODUCE. Never rewrite source text solely because it already contains one of them.
- ` + bannedWords + `

` + aiTellBans
		if proseGuide != "" {
			base = `You are a restrained line editor. Selectively polish the provided HTML text while preserving the author's voice.` + styleBlock + `

Editing rules:
- Use the style examples only as a constraint when repairing a concrete problem, never as a reason to rewrite correct prose
- Change a sentence only for awkward or confusing phrasing, unintended repetition, unclear syntax, or rhythm that visibly stumbles
- If a sentence is already clear and effective, preserve it exactly
- Make the smallest edit that solves the problem; do not rewrite whole paragraphs merely to offer an alternative
- Do not add detail, imagery, emphasis, interpretation, new ideas, or vocabulary upgrades
- Preserve all story facts, meaning, tense, POV, characterization, dialogue wording, and intentional fragments
- Preserve paragraph breaks — return one <p> element per original paragraph

The restrictions below — including banned words and AI-tell rules — constrain wording YOU INTRODUCE. Never rewrite source text solely because it already contains one of them.
- ` + bannedWords + `

` + aiTellBans
		}
		return base + `

CRITICAL: Return ONLY the rewritten HTML content using <p> tags. Do not include any instructions, explanations, system prompts, or meta-commentary. Output raw HTML only.`
	}
}

// BuildStyleMixerInstructions converts WritingStyleOptions to prompt instructions.
func BuildStyleMixerInstructions(opts *types.WritingStyleOptions) string {
	if opts == nil {
		return ""
	}

	intensityWords := []string{"", "subtle", "moderate", "heavy"}
	var instructions []string

	// Metaphors - CRITICAL: these are major AI tells
	if opts.Metaphors == 0 {
		instructions = append(instructions, "- METAPHORS: ABSOLUTELY FORBIDDEN. Never add any new metaphors. Do not write phrases like 'was a [noun]', 'became a [noun]', or any figurative comparisons. Only preserve metaphors that already exist word-for-word in the source text.")
	} else if opts.Metaphors > 0 && opts.Metaphors <= 3 {
		instructions = append(instructions, fmt.Sprintf("- Add %s use of metaphors (figurative comparisons)", intensityWords[opts.Metaphors]))
	}

	// Similes - CRITICAL: these are major AI tells
	if opts.Similes == 0 {
		instructions = append(instructions, "- SIMILES: ABSOLUTELY FORBIDDEN. Never add any new similes. Do not write 'like a...', 'as if...', 'as though...', or any like/as comparisons. Only preserve similes that already exist word-for-word in the source text.")
	} else if opts.Similes > 0 && opts.Similes <= 3 {
		instructions = append(instructions, fmt.Sprintf("- Add %s use of similes (like/as comparisons)", intensityWords[opts.Similes]))
	}

	// Sensory Detail
	if opts.SensoryDetail == 0 {
		instructions = append(instructions, "- SENSORY DETAILS: Do not add new sensory descriptions. Preserve only what exists in the source.")
	} else if opts.SensoryDetail > 0 && opts.SensoryDetail <= 3 {
		instructions = append(instructions, fmt.Sprintf("- Add %s sensory details (sight, sound, smell, touch, taste)", intensityWords[opts.SensoryDetail]))
	}

	// Internal Thought
	if opts.InternalThought == 0 {
		instructions = append(instructions, "- INTERNAL THOUGHT: Do not add character introspection or internal monologue. Preserve only what exists in the source.")
	} else if opts.InternalThought > 0 && opts.InternalThought <= 3 {
		instructions = append(instructions, fmt.Sprintf("- Add %s internal thought and character introspection", intensityWords[opts.InternalThought]))
	}

	// Dialogue
	if opts.Dialogue == 0 {
		instructions = append(instructions, "- DIALOGUE: Do not expand or add dialogue. Preserve only what exists in the source.")
	} else if opts.Dialogue > 0 && opts.Dialogue <= 3 {
		word := intensityWords[opts.Dialogue]
		instructions = append(instructions, fmt.Sprintf("- %s%s expansion of dialogue and conversation", strings.ToUpper(word[:1]), word[1:]))
	}

	// Action
	if opts.Action == 0 {
		instructions = append(instructions, "- ACTION: Do not add physical action or movement beats. Preserve only what exists in the source.")
	} else if opts.Action > 0 && opts.Action <= 3 {
		instructions = append(instructions, fmt.Sprintf("- Add %s physical action and movement beats", intensityWords[opts.Action]))
	}

	// Description
	if opts.Description == 0 {
		instructions = append(instructions, "- DESCRIPTION: Do not add setting description or atmosphere. Preserve only what exists in the source.")
	} else if opts.Description > 0 && opts.Description <= 3 {
		instructions = append(instructions, fmt.Sprintf("- Add %s setting description and atmosphere", intensityWords[opts.Description]))
	}

	// Pacing
	if opts.Pacing == 0 {
		instructions = append(instructions, "- Keep sentence rhythm uniform")
	} else if opts.Pacing > 0 && opts.Pacing <= 3 {
		switch opts.Pacing {
		case 1:
			instructions = append(instructions, "- Slight variation in sentence rhythm")
		case 2:
			instructions = append(instructions, "- Moderate variation in sentence rhythm — mix short and long sentences")
		case 3:
			instructions = append(instructions, "- Heavy variation in sentence rhythm — dramatic contrasts between punchy and flowing sentences")
		}
	}

	return strings.Join(instructions, "\n")
}

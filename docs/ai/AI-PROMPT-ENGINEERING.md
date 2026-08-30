# AI Prompt Engineering in Draftline

This document explains how Draftline's AI rewriting modes work, including the system prompts, user messages, and style mixer system that powers the prose enhancement features.

---

## Overview

Draftline uses a two-part prompt architecture:

1. **System Prompt** - Establishes the AI's role, rules, and constraints
2. **User Message** - Contains the actual text to process and any context

The system prompt is built dynamically based on:
- The selected mode (line_edit, expand, smooth, custom)
- The prose style guide (if configured)
- The style mixer settings (8 sliders controlling prose characteristics)

---

## AI Providers

Draftline supports multiple AI backends:

| Provider | Mode | Configuration |
|----------|------|---------------|
| Claude Code | `claudecode` | Uses Claude Code CLI credentials (OAuth or API key) |
| Anthropic API | `api` + `claude` | Direct API key in settings |
| OpenAI API | `api` + `openai` | Direct API key in settings |
| Google Gemini | `api` + `gemini` | Direct API key in settings |
| Grok (X.AI) | `api` + `grok` | Direct API key in settings |
| Local (Ollama, etc.) | `local` | OpenAI-compatible endpoint URL |

When using Claude Code with an API key stored in credentials, Draftline calls the Anthropic API directly with streaming for better performance. When only OAuth credentials exist, it falls back to the CLI subprocess.

Line Edit and Copy Edit are lightweight tasks and deliberately bypass a configured heavyweight model. Claude uses Haiku, Codex selects Luna (or an available Mini tier) from the CLI's own model cache with low reasoning, OpenAI API uses Mini, and Gemini uses Flash. Expand, Smooth, Custom, and inline generation continue to use the model selected in Settings.

---

## The Modes

### Line Edit Mode (Default)

**Purpose:** Selective prose polish for style, craft, rhythm, tone, awkward phrasing, repetition, and readability. This is not a whole-passage rewrite.

**System Prompt:**
```
You are a restrained line editor. Selectively polish the provided HTML text
while preserving the author's voice.

Editing rules:
- Change a sentence only for a concrete line-level problem
- Preserve sentences that are already clear and effective exactly
- Make the smallest edit that solves the problem
- Do not add detail, imagery, interpretation, or new ideas
- Do not upgrade vocabulary or rewrite paragraphs merely to offer an alternative
- Preserve facts, meaning, tense, POV, characterization, dialogue, and fragments
- Preserve paragraph breaks — return one <p> element per original paragraph

[AI Tell Bans constrain only newly introduced wording — see below]

CRITICAL: Return ONLY the rewritten HTML content using <p> tags. Do not include
any instructions, explanations, system prompts, or meta-commentary. Output raw HTML only.
```

**With Prose Guide Enabled:**
The guide is a constraint used only while repairing a concrete problem. It is explicitly not permission to rewrite otherwise-correct prose to resemble the samples more closely.

---

### Copy Edit Mode

**Purpose:** A rigid, mechanical correctness pass covering spelling, grammar, punctuation, syntax, doubled words, homophones, capitalization, hyphenation, and number-format consistency.

Copy Edit never rephrases for style, rhythm, readability, or word choice. It preserves intentional fragments, pacing punctuation, and character-voice dialogue. The prose guide and AI-tell rules are omitted entirely to reduce latency and prevent subjective rewriting.

**Diff Format Output (Token Optimization):**
For line_edit, copy_edit, and smooth modes, Draftline uses a diff format that reduces output tokens by ~80%:
```
CRITICAL OUTPUT FORMAT: Each input paragraph is prefixed §N§ where N is its 1-based index.
Return ONLY paragraphs you change, one per line:
§N§<p>revised text</p>
Omit unchanged paragraphs entirely. If nothing needs changing: §NONE§
```

---

### Expand Mode

**Purpose:** Enrich thin prose by adding depth—sensory detail, internal thought, atmosphere. This is the most sophisticated mode because it uses the Style Mixer system.

**System Prompt (Base):**
```
You are a literary prose writer. Expand and enrich the provided HTML text.

Rules:
- Match the existing POV depth, tense, and voice exactly
- Do not introduce new plot events or characters
- Preserve paragraph breaks — return one <p> element per original paragraph (may be longer)
- BANNED words and phrases: [same list]

[AI Tell Bans - see below]
[Style Guide block if configured]

STYLE PREFERENCES (follow these carefully):
[Style Mixer Instructions - see below]

CRITICAL: Return ONLY the rewritten HTML content using <p> tags. Do not include
any instructions, explanations, system prompts, or meta-commentary. Output raw HTML only.
```

**Default Behavior (No Style Mixer):**
```
- Flesh out thin paragraphs — add physical sensation, setting detail, internal thought
- Show don't tell: replace summary with scene
```

**The Elegance of Expand Mode:**

What makes Expand special is its **additive** nature. Unlike selective Line Edit or Smooth (which polish), Expand explicitly:

1. **Preserves everything** - "Match the existing POV depth, tense, and voice exactly"
2. **Adds layers** - via the Style Mixer's 8 dimensions
3. **Respects structure** - paragraphs can grow but not multiply

---

### Smooth Mode

**Purpose:** Flow and rhythm polish. Eliminates repetition, improves transitions, varies sentence openings. Minimal semantic changes.

**System Prompt:**
```
You are a line editor focused on flow and rhythm. Smooth the provided HTML text.

Rules:
- Eliminate word repetition within paragraphs (same word used 2+ times nearby)
- Improve sentence-to-sentence transitions
- Vary sentence openings — avoid starting consecutive sentences the same way
- Minimal changes — improve flow without changing meaning or voice
- Preserve paragraph breaks — return one <p> element per original paragraph
- BANNED words and phrases: [same list]

[AI Tell Bans - see below]
[Style Guide block if configured]
[Style Mixer Instructions if configured]

CRITICAL: Return ONLY the rewritten HTML content using <p> tags. Do not include
any instructions, explanations, system prompts, or meta-commentary. Output raw HTML only.
```

---

### Custom Mode

**Purpose:** User-defined rewriting. The AI follows whatever instruction the user provides.

**System Prompt:**
```
You are a skilled fiction editor helping an author revise their manuscript.
Follow the user's instruction precisely. Preserve the author's voice and style.
If the input is HTML, preserve the HTML structure (<p>, <em>, <strong>, etc).

[AI Tell Bans - see below]

CRITICAL: Return ONLY the revised text content. Do not include any instructions,
explanations, system prompts, or meta-commentary. Output the prose only.
```

**User Message Format:**
```
Instruction: [User's custom prompt]

Text to revise:
[The selected text]
```

**Use Cases:**
- "Make this more suspenseful"
- "Rewrite this from Sarah's POV instead of John's"
- "Add more humor to the dialogue"
- "Make this shorter and punchier"

Custom mode also powers the `@ai` inline commands. When text contains `@ai [instruction]`, the system extracts the instruction and processes the surrounding paragraph.

---

### Inline Generation (Ctrl+L)

**Purpose:** Generate new content at the cursor position, seamlessly fitting the surrounding prose.

**System Prompt:**
```
You are a skilled fiction author helping write a manuscript.
Generate new content that seamlessly fits between the existing prose.
Match the voice, style, tense, and POV of the surrounding text.
Return ONLY the new content as HTML paragraphs (<p>...</p>).
Do not include any explanations, just the prose.

BANNED words and phrases: [same list]

[AI Tell Bans - see below]

STYLE RULE:
Break grammar rules intentionally where rhythm demands it. Fragments are allowed.
Sentences can start with And or But. Comma splices are permitted for pacing.
Grammatical correctness is not the goal. The sentence is the goal.

[Style Guide block if configured]

Characters in this story: [Character names from Story Bible]

CRITICAL: Return ONLY the new prose content as HTML paragraphs. Do not include
any instructions, explanations, system prompts, or meta-commentary. Output raw HTML only.
```

**User Message Format:**
```
Chapter: [Current chapter title]

TEXT BEFORE:
[~500 characters of preceding text]

TEXT AFTER:
[~300 characters of following text]

INSTRUCTION: [User's prompt]

Generate the new content to insert between the before and after text:
```

**Key Design Choices:**
- **Context sandwich** - providing text before AND after lets the AI write content that bridges naturally
- **Character awareness** - including character names prevents invention of new characters
- **Chapter context** - helps maintain tonal consistency within a chapter
- **Asymmetric context** - more "before" (500 chars) than "after" (300 chars) because preceding context is usually more important for continuity

---

## AI Tell Bans

A critical addition to all prompts that prevents common AI writing patterns:

```
ABSOLUTELY FORBIDDEN — AI TELL CONSTRUCTIONS:
1. Em-dash appositive definitions: NEVER write "[quality] — that [particular/specific/certain]
   [noun] of someone who [explanation]". Show the quality, never name and define it in the
   same breath.
2. Gerund-plus-abstract-noun behavior labeling: NEVER write "performing normalcy", "performing
   grief", "performing calm", or any "[gerund] + [abstract social/emotional noun]" construction.
   Let behavior speak for itself.
3. Clinical precision words that no narrator actually thinks in: "over-relaxation",
   "micro-expression", "hyperawareness", "hypervigilance". Replace with visceral physical
   observation.
4. Meta-pattern references: NEVER write "the thing it did", "the way she always", "that look
   he had". Show the specific instance, not the pattern.
5. Narrator taxonomy and cataloguing: NEVER have the narrator classify, catalogue, or
   taxonomize behavior with fake academic precision. BANNED: "a particular subspecies",
   "catalogued privately", "a specific category of". Narrators notice things, they do not
   file them.
6. Triple synonym stacking: NEVER stack near-synonyms in twos or threes for emphasis.
   BANNED: "simply, entirely, thoroughly", "wordless and mutual and instinctive". Pick the
   single strongest word and trust it.
7. Similes that overstay: End comparisons when the image lands. NEVER extend a simile past
   the point where the meaning is clear. If you are still explaining the comparison after
   the first clause, cut it.
8. Announcing literary references as shortcuts: BANNED: "contained multitudes", "the whole
   of her", "more than she let on". Show the contradiction directly, never name it.
9. The indifferent world pan-out: NEVER end a scene or paragraph by pulling back to an
   outside world that is unaware of or indifferent to the characters. This is an AI default
   scene-closing move and is always cut.

STYLE RULE:
11. Break grammar rules intentionally where rhythm demands it. Fragments are allowed.
    Sentences can start with And or But. Comma splices are permitted for pacing.
    Grammatical correctness is not the goal. The sentence is the goal.
```

---

## The Style Mixer System

The Style Mixer provides 8 sliders, each with 4 levels:
- **0 (Off)** - Do not add this element; preserve only what exists
- **1 (Subtle)** - Light touch
- **2 (Moderate)** - Balanced presence
- **3 (Heavy)** - Strong emphasis

### The 8 Dimensions

#### 1. Metaphors
**Why it matters:** Metaphors are a major AI tell. Models love to add them.

| Level | Instruction |
|-------|------------|
| 0 | `METAPHORS: ABSOLUTELY FORBIDDEN. Never add any new metaphors. Do not write phrases like 'was a [noun]', 'became a [noun]', or any figurative comparisons. Only preserve metaphors that already exist word-for-word in the source text.` |
| 1-3 | `Add [subtle/moderate/heavy] use of metaphors (figurative comparisons)` |

The level 0 instruction is **intentionally emphatic** because AI models have a strong prior toward adding metaphors. Simple "don't add metaphors" often fails.

#### 2. Similes
**Why it matters:** Like metaphors, similes are AI tells. "Like a [poetic noun]" screams synthetic prose.

| Level | Instruction |
|-------|------------|
| 0 | `SIMILES: ABSOLUTELY FORBIDDEN. Never add any new similes. Do not write 'like a...', 'as if...', 'as though...', or any like/as comparisons. Only preserve similes that already exist word-for-word in the source text.` |
| 1-3 | `Add [subtle/moderate/heavy] use of similes (like/as comparisons)` |

#### 3. Sensory Detail
**What it controls:** Sight, sound, smell, touch, taste descriptions.

| Level | Instruction |
|-------|------------|
| 0 | `SENSORY DETAILS: Do not add new sensory descriptions. Preserve only what exists in the source.` |
| 1-3 | `Add [subtle/moderate/heavy] sensory details (sight, sound, smell, touch, taste)` |

#### 4. Internal Thought
**What it controls:** Character introspection, internal monologue, psychological depth.

| Level | Instruction |
|-------|------------|
| 0 | `INTERNAL THOUGHT: Do not add character introspection or internal monologue. Preserve only what exists in the source.` |
| 1-3 | `Add [subtle/moderate/heavy] internal thought and character introspection` |

#### 5. Dialogue
**What it controls:** Expansion of conversations, adding dialogue beats.

| Level | Instruction |
|-------|------------|
| 0 | `DIALOGUE: Do not expand or add dialogue. Preserve only what exists in the source.` |
| 1-3 | `[Subtle/Moderate/Heavy] expansion of dialogue and conversation` |

#### 6. Action
**What it controls:** Physical movement, body language, action beats.

| Level | Instruction |
|-------|------------|
| 0 | `ACTION: Do not add physical action or movement beats. Preserve only what exists in the source.` |
| 1-3 | `Add [subtle/moderate/heavy] physical action and movement beats` |

#### 7. Description
**What it controls:** Setting description, atmosphere, world-building details.

| Level | Instruction |
|-------|------------|
| 0 | `DESCRIPTION: Do not add setting description or atmosphere. Preserve only what exists in the source.` |
| 1-3 | `Add [subtle/moderate/heavy] setting description and atmosphere` |

#### 8. Pacing
**What it controls:** Sentence rhythm variation.

| Level | Instruction |
|-------|------------|
| 0 | `Keep sentence rhythm uniform` |
| 1 | `Slight variation in sentence rhythm` |
| 2 | `Moderate variation in sentence rhythm — mix short and long sentences` |
| 3 | `Heavy variation in sentence rhythm — dramatic contrasts between punchy and flowing sentences` |

### Example Style Mixer Configurations

**Thriller Scene (fast-paced action):**
```
Metaphors: 0
Similes: 0
Sensory: 1
Internal Thought: 0
Dialogue: 0
Action: 3
Description: 0
Pacing: 3
```

This produces tight, punchy prose with lots of physical movement and dramatic rhythm changes—no flowery language.

**Literary Introspection (slow, atmospheric):**
```
Metaphors: 1
Similes: 1
Sensory: 3
Internal Thought: 3
Dialogue: 0
Action: 0
Description: 2
Pacing: 1
```

This produces rich interior prose with heavy sensory and psychological depth, light figurative language.

---

## The Banned Words List

```
tapestry, testament, navigate, delve, underscore, myriad, realm, crucial,
pivotal, journey, beacon, vibrant, game-changer
```

These words appear disproportionately in AI-generated text. Banning them forces the model toward more natural vocabulary. This list could be expanded based on observation.

---

## Technical Notes

### HTML Preservation
All prompts explicitly require HTML output (`<p>` tags) because:
1. The editor stores content as HTML
2. Formatting (bold, italic) must be preserved
3. Paragraph structure must remain intact

### The "One `<p>` Per Paragraph" Rule
This constraint prevents the AI from:
- Splitting paragraphs (expanding 3 paragraphs into 10)
- Merging paragraphs (losing pacing decisions)
- Adding extra structural elements

The author's paragraph breaks are intentional pacing choices.

### Diff Format Optimization
For line_edit, copy_edit, and smooth modes, input paragraphs are prefixed with `§N§` markers. The AI returns only changed paragraphs with their indices, reducing output tokens significantly. The `applyDiffResponse()` function reconstructs the full HTML by merging changes with unchanged paragraphs.

### Context Limits
- **Prose Guide:** Can include substantial examples since it's in the system prompt
- **Inline Generation:** Uses ~500 chars before, ~300 chars after (tuned for context window efficiency)
- **Text Selection:** The UI encourages processing 1-5 paragraphs at a time for quality

### Streaming
When using the Anthropic API directly (Claude Code with API key or API mode with Claude), Draftline streams tokens to the UI in real-time via the `ai:token` event, providing immediate feedback during generation.

### AI Provider Flexibility
All prompts work with:
- Claude (Anthropic)
- GPT-4 (OpenAI)
- Gemini (Google)
- Grok (X.AI)
- Local models (via OpenAI-compatible endpoints)

The prompts are model-agnostic—they don't rely on any model-specific features.

---

## Future Considerations

1. **Per-chapter style profiles** - Different settings for action vs. dialogue chapters
2. **Learned bans** - Letting users add words to the banned list as they spot patterns
3. **Comparison mode** - Running the same text through multiple configurations
4. **Prompt templates** - Saved custom prompts for common tasks ("make this scarier")

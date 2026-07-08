# PlotWalker: Story Structure Tools

PlotWalker is Draftline's suite of story structure and continuity tracking tools. It lives in the **Bible** tab of the right sidebar and helps authors maintain consistency across complex narratives.

---

## Overview

PlotWalker consists of six integrated subsystems:

| Tool | Purpose | Data Location |
|------|---------|---------------|
| **Characters** | Track character details, appearances, and relationships | `story_bible.characters[]` |
| **Plot Notes** | Freeform outline of acts, themes, subplots | `story_bible.plot_notes` |
| **Timeline** | Chronological event tracking | `story_bible.timeline` |
| **Beat Sheet** | Save the Cat! story beats per chapter | `beat_sheet.beats[]` |
| **Foreshadowing** | Plant → Reinforce → Payoff tracking | `foreshadowing.items[]` |
| **Knowledge Matrix** | Who knows what secrets, when | `knowledge_matrix.secrets[]`, `knowledge_matrix.entries[]` |

All data persists in the `.draftline` project file (ZIP archive with JSON).

---

## Characters

### Overview
The character system combines **auto-detection** with manual entry. Draftline scans your prose to find character names and tracks their mentions across chapters.

### Auto-Detection Algorithm

Located in `app.go`, the detection system uses:

1. **Dialogue Attribution Patterns** (highest confidence)
   ```
   "Hello," said John.        → John (after quote)
   John said, "Hello."        → John (before quote)
   ```
   Dialogue verbs: `said, asked, replied, answered, whispered, shouted, yelled, muttered, exclaimed, cried, called, screamed, murmured, snapped, growled, laughed, sighed, groaned, demanded, insisted, suggested, agreed, admitted, explained, continued, added, interrupted, announced, declared, observed, remarked, noted, commented, wondered, mused, thought, began, finished, concluded`

2. **Title + Name Patterns** (high confidence)
   ```
   Mr. Smith, Dr. Jones, Captain Kirk
   ```
   Recognized titles: `Mr, Mrs, Ms, Miss, Dr, Prof, Professor, Captain, Colonel, General, Lieutenant, Sergeant, Officer, Detective, Agent, Lord, Lady, Sir, Dame, King, Queen, Prince, Princess, Senator, Governor, Mayor, Chief, Father, Mother, Sister, Brother, Uncle, Aunt`

3. **Common Word Filtering**
   Names that match common English words are excluded (pronouns, conjunctions, days, months, etc.)

4. **Suffix Filtering**
   Words ending in `-ly`, `-ing`, `-tion`, `-ness`, `-ment`, etc. are excluded as likely non-names.

### Character Data Model

```typescript
interface Character {
  id: string
  name: string
  role: 'protagonist' | 'antagonist' | 'supporting' | 'minor' | 'other'
  description: string
  appearance: string
  personality: string
  motivation: string
  notes: string
  // Auto-detection fields
  is_auto_detected: boolean
  aliases: string[]              // Alternative names/nicknames
  mention_count: number          // Total mentions across all chapters
  first_chapter: number          // Chapter index where first mentioned
  chapter_mentions: Record<number, number>  // Chapter → mention count
  attributes: {                  // Extracted physical attributes
    eye_color?: string
    hair_color?: string
    age?: string
  }
}
```

### UI Features

**Character List** (in current chapter)
- Shows only characters mentioned in the active chapter
- Sort by: Mentions, First Appearance, Name
- Quick stats: mention count, first chapter

**Character Card**
- Expandable for full details
- Highlight button: highlights character name in editor
- Edit/Delete actions
- Shows aliases, attributes, chapter breakdown

**Merge Mode**
- Select 2+ characters that are the same person
- Choose which name becomes primary
- Others become aliases
- Mention counts combine

**Full Codex**
- Opens dedicated view with all characters across book
- Re-index button to refresh auto-detection

### Indexing

**Full Book Index** (`IndexBook`)
- Scans all chapters (front matter, body, back matter)
- Extracts names using dialogue patterns
- Filters by minimum 3 mentions
- Extracts attributes (eye color, hair color, age)
- Updates existing characters, creates new ones

**Incremental Index** (`IndexChapter`)
- Scans single chapter
- Minimum 2 mentions threshold
- Faster for real-time updates

---

## Plot Notes

### Overview
Freeform textarea for story structure planning. No structured data—just markdown-style notes.

### Suggested Format
```
Act 1:
  Opening image — Marcus alone in apartment
  Inciting incident — Letter arrives

Act 2:
  B-story begins — Clara introduced
  Midpoint — Discovery in basement

Act 3:
  Climax — Confrontation
  Resolution — New equilibrium

Themes:
  - Isolation vs. connection
  - Truth vs. comfortable lies

Subplots:
  - Marcus/Clara relationship arc
  - Office politics thread
```

### Data Storage
Stored as plain string in `story_bible.plot_notes`.

---

## Timeline

### Overview
Chronological event tracking in story time. One event per line, freeform format.

### Suggested Format
```
Day 1 — Marcus arrives in the city (Ch. 1)
Day 1, evening — He meets Clara at the bar (Ch. 2)
Day 2 — First day at new job (Ch. 3)
Day 3 — The mysterious letter arrives (Ch. 4)
Day 3, night — Break-in attempt (Ch. 5)
```

### Data Storage
Stored as plain string in `story_bible.timeline`.

---

## Beat Sheet

### Overview
Structured story beat tracking based on Blake Snyder's "Save the Cat!" methodology. Each beat is linked to a specific chapter.

### Beat Types

| Beat | Description |
|------|-------------|
| Opening Image | The "before" snapshot of protagonist's world |
| Theme Stated | Hint at the story's deeper meaning |
| Setup | Establish status quo, introduce characters |
| Catalyst | The event that disrupts everything |
| Debate | Protagonist hesitates, weighs options |
| Break into Two | Commitment to the journey |
| B Story | Secondary storyline begins (often love interest) |
| Fun & Games | The "promise of the premise" delivered |
| Midpoint | False victory or false defeat; stakes raised |
| Bad Guys Close In | External/internal pressure mounts |
| All Is Lost | Lowest point, "whiff of death" |
| Dark Night | Protagonist processes the loss |
| Break into Three | Synthesis of A and B stories |
| Finale | Climax and resolution |
| Final Image | The "after" snapshot, mirror of opening |
| Custom | User-defined beat type |

### Data Model

```typescript
interface Beat {
  id: string
  chapter_index: number      // Which chapter this beat occurs in
  beat_type: BeatType        // One of the types above
  description: string        // What happens in this beat
  notes?: string             // Optional additional notes
}

interface BeatSheet {
  beats: Beat[]
}
```

### UI Features

**Add Beat Form**
- Chapter selector (dropdown of all chapters)
- Beat type selector
- Description textarea
- Optional notes

**Beat List**
- Grouped by chapter
- Shows beat type badge
- Inline edit/delete
- Ordered by chapter sequence

### Storage
Saved in `beat_sheet.json` within the `.draftline` archive.

---

## Foreshadowing Ledger

### Overview
Track the lifecycle of foreshadowing elements through plant → reinforce → payoff stages.

### Concept

**Plant**: The initial subtle hint or setup
**Reinforce**: Subsequent reminders that keep it in reader's mind
**Payoff**: The revelation or callback

Example:
- **Plant (Ch. 2)**: Marcus notices the locked drawer in his father's desk
- **Reinforce (Ch. 5)**: Father catches Marcus near the desk, overreacts
- **Reinforce (Ch. 8)**: Marcus finds a key that doesn't fit any door he knows
- **Payoff (Ch. 12)**: Marcus opens the drawer, finds the letters

### Data Model

```typescript
interface ForeshadowingItem {
  id: string
  name: string                    // Short label (e.g., "Locked Drawer")
  description: string             // Full description of what's being foreshadowed
  plant_chapter: number           // Chapter index where planted
  reinforce_chapters: number[]    // Chapters where reinforced
  payoff_chapter?: number         // Chapter where paid off (nullable)
  status: 'planted' | 'active' | 'resolved'
  notes?: string
}

interface ForeshadowingLedger {
  items: ForeshadowingItem[]
}
```

### Status Colors
- **Planted** (orange): Just introduced, needs reinforcement
- **Active** (blue): Being reinforced, awaiting payoff
- **Resolved** (green): Payoff delivered

### UI Features

**Add Foreshadowing Form**
- Name and description
- Plant chapter (dropdown)
- Reinforce chapters (comma-separated numbers)
- Payoff chapter (optional)
- Status selector

**Foreshadowing List**
- Shows all items with status badges
- Chapter references displayed
- Inline edit/delete

### Use Cases

1. **Chekhov's Gun tracking**: Ensure setup elements pay off
2. **Mystery clues**: Track reveals and misdirections
3. **Character arc setups**: Early hints of character growth
4. **World-building seeds**: Details mentioned early, expanded later

### Storage
Saved in `foreshadowing.json` within the `.draftline` archive.

---

## Knowledge Matrix

### Overview
Track which characters know which secrets, and when they learn them. Essential for mysteries, thrillers, and any narrative with hidden information.

### Concept

A matrix where:
- **Rows**: Secrets/information pieces
- **Columns**: Characters
- **Cells**: Chapter number when character learns the secret

### Data Model

```typescript
interface SecretInfo {
  id: string
  name: string              // Short label (e.g., "Marcus is adopted")
  description: string       // Full description
}

interface KnowledgeEntry {
  secret_id: string
  character_id: string
  learns_chapter?: number           // When they definitively learn it
  suspected_chapter?: number        // When they start to suspect (future feature)
}

interface KnowledgeMatrix {
  secrets: SecretInfo[]
  entries: KnowledgeEntry[]
}
```

### UI Features

**Secret Management**
- Add/edit/delete secrets
- Name and description fields

**Matrix Grid**
- Secrets as rows
- Characters as columns (first 5 shown)
- Each cell: input for chapter number
- Empty = doesn't know

**Full Matrix**
- Available in Codex view for all characters

### Use Cases

1. **Mystery plotting**: Track when detective learns each clue
2. **Dramatic irony**: Visualize what reader knows vs. characters
3. **Reveal timing**: Plan revelation sequence
4. **Continuity check**: Ensure characters don't know things prematurely

### Example

| Secret | Marcus | Clara | Detective | Reader |
|--------|--------|-------|-----------|--------|
| Father's affair | Ch. 12 | — | Ch. 15 | Ch. 3 |
| Hidden inheritance | — | Ch. 8 | Ch. 14 | Ch. 8 |
| True killer identity | Ch. 18 | Ch. 18 | Ch. 16 | Ch. 1 |

### Storage
Saved in `knowledge_matrix.json` within the `.draftline` archive.

---

## Integration with Editor

### Character Highlighting
When viewing a chapter, you can click the highlight button on any character card. This activates the `CharacterHighlight` Tiptap extension which:
1. Marks all mentions of that character's name in the editor
2. Includes alias matches
3. Visual highlight style applied

### AI Context
The Characters list feeds into AI prompts:
- Inline generation includes character names to prevent invention
- Custom prompts can reference existing characters

### Chapter-Scoped Display
The Characters section filters to show only characters appearing in the current chapter. This reduces cognitive load while writing.

---

## File Format

All PlotWalker data is stored in the `.draftline` project file (ZIP archive):

```
project.draftline/
├── manifest.json
├── story_bible.json      # Characters, plot_notes, timeline
├── beat_sheet.json       # Beat sheet beats
├── foreshadowing.json    # Foreshadowing items
├── knowledge_matrix.json # Secrets and entries
├── body/
│   ├── 000.html
│   ├── 001.html
│   └── ...
└── ...
```

Each JSON file is only created when data exists (graceful handling of older files).

---

## Future Enhancements

1. **Character relationship graph**: Visual web of character connections
2. **Timeline visualization**: Graphical timeline with chapter markers
3. **Beat sheet templates**: Genre-specific beat structures
4. **Foreshadowing warnings**: Alert if planted element lacks payoff
5. **Knowledge matrix export**: Generate spoiler-free plot summaries
6. **AI-assisted indexing**: Use LLM to identify characters with higher accuracy
7. **Cross-reference search**: Find all chapters mentioning a character/secret
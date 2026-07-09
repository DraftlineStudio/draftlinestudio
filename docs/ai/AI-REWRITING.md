# AI Features

Draftline includes AI-powered writing assistance that helps authors rewrite, expand, and polish their prose.

## Feature Status

AI features are **fully implemented** but controlled via settings:

| Setting | Location | Effect |
|---------|----------|--------|
| `ai_enabled` | Settings > AI | Master toggle for all AI features |
| `ai_mode` | Settings > AI | Provider selection (claudecode/api/local) |
| `ai_provider` | Settings > AI | Which API to use (claude/openai/gemini/grok) |

When AI is disabled, the AI Studio panel shows setup guidance instead of controls.

## Supported Providers

| Provider | Mode | Notes |
|----------|------|-------|
| **Claude Code** | `claudecode` | Uses Claude Code CLI (OAuth or API key from creds) |
| **Anthropic API** | `api` + `claude` | Direct API key |
| **OpenAI** | `api` + `openai` | GPT-4 and variants |
| **Google Gemini** | `api` + `gemini` | Gemini Pro |
| **Grok (xAI)** | `api` + `grok` | Grok models |
| **Local (Ollama)** | `local` | Any OpenAI-compatible endpoint |

## AI Modes

### Line Edit
General prose polish. Improves rhythm, word choice, clarity while preserving content exactly.

### Expand
Enriches thin prose with depth - sensory detail, internal thought, atmosphere. Uses the Style Mixer for fine control.

### Smooth
Flow and rhythm polish. Eliminates repetition, improves transitions, varies sentence openings.

### Custom
User-defined instruction. "Make this more suspenseful", "Add humor to dialogue", etc.

### Inline Generation (Ctrl+L)
Generate new content at cursor position, seamlessly fitting surrounding prose.

## The Style Mixer

8 sliders (0-3 each) that control prose characteristics:

| Dimension | Level 0 | Level 3 |
|-----------|---------|---------|
| Metaphors | Forbidden | Heavy use |
| Similes | Forbidden | Heavy use |
| Sensory | Preserve only | Heavy detail |
| Internal Thought | Preserve only | Heavy introspection |
| Dialogue | Preserve only | Heavy expansion |
| Action | Preserve only | Heavy movement |
| Description | Preserve only | Heavy atmosphere |
| Pacing | Uniform rhythm | Dramatic variation |

## Documentation

- **[AI Prompt Engineering](AI-PROMPT-ENGINEERING.md)** - Detailed documentation of all prompts, the style mixer system, banned words, and AI tell prevention.

This is the key document for understanding how Draftline's AI features work under the hood.

## Architecture

```
User triggers rewrite
       |
       v
Frontend: bookStore.rewriteSelection()
       |
       v
Wails: RewriteText(html, mode, styleOpts)
       |
       v
Go: buildSystemPrompt()
    - Base prompt for mode
    - Style mixer instructions
    - Banned words list
    - AI tell bans
    - Prose guide (if configured)
       |
       v
Go: callProvider()
    - Claude Code CLI subprocess, OR
    - Direct API call (Anthropic/OpenAI/etc)
       |
       v
Stream tokens via "ai:token" event
       |
       v
Frontend: Accumulate result, show diff
       |
       v
User: Accept/reject changes
```

## Token Optimization

For line_edit and smooth modes, Draftline uses a diff format that reduces output tokens by ~80%:

```
Input:  §1§<p>First paragraph</p>
        §2§<p>Second paragraph</p>
        §3§<p>Third paragraph</p>

Output: §2§<p>Revised second paragraph</p>
        (Only changed paragraphs returned)
```

The `applyDiffResponse()` function reconstructs the full HTML.

## Error Handling

AI calls can fail for various reasons:
- Network issues
- API rate limits
- Invalid API keys
- Context length exceeded

All errors are captured and shown to the user with actionable messages.

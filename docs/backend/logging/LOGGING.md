# Logging Package

`internal/logging/` provides debug logging for AI operations.

## Functions

### Init()
Initializes the debug log file (no error return; failures silently disable file logging). Called lazily on first log write. Creates a timestamped `ai-*.log` under the OS config directory and rotates, keeping the last 5 log files.

### SetEnabled(on bool)
Turns AI debug logging on or off. Off by default; wired to the `ai_debug_logging` setting.

### AI(format string, args ...interface{})
Logs a formatted message to the AI debug log. No-op unless enabled.

### AIContent(label string, content string)
Logs content blocks (prompts, responses) with a label prefix, truncated to 2,000 characters. No-op unless enabled.

## Log File

- **Location:** `os.UserConfigDir()/draftline/logs/ai-<timestamp>.log`, created user-only (0600/0700)
- **Format:** Timestamped entries with operation labels
- **Content:** AI prompts, responses, provider info, errors

## Usage

```go
logging.AI("========== RewriteText START ==========")
logging.AI("mode=%s provider=%s", mode, provider)
logging.AIContent("INPUT_HTML", html)
logging.AIContent("OUTPUT_RESULT", result)
```

## Enabling

Debug logging is opt-in via `SetEnabled(true)`, wired to the `ai_debug_logging` app setting. Nothing is written while disabled.

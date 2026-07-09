# Logging Package

`internal/logging/` provides debug logging for AI operations.

## Functions

### Init() error
Initializes the debug logger. Creates `ai_debug.log` in the user's home directory if AI debugging is enabled.

### AI(format string, args ...interface{})
Logs a formatted message to the AI debug log.

### AIContent(label string, content string)
Logs large content blocks (prompts, responses) with a label prefix.

## Log File

- **Location:** `~/ai_debug.log` (user home directory)
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

Debug logging is enabled when the `AI_DEBUG` environment variable is set, or when running in development mode.

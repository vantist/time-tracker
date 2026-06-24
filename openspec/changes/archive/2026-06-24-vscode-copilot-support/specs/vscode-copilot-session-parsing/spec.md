## ADDED Requirements

### Requirement: Parse transcripts JSONL format
The system SHALL parse VS Code Copilot Chat transcript files (`transcripts/*.jsonl`) from workspaceStorage to extract session events.

#### Scenario: Parse session start event
- **WHEN** a transcript JSONL file contains a `session.start` event
- **THEN** the system extracts sessionId, startTime, copilotVersion, vscodeVersion

#### Scenario: Parse user message event
- **WHEN** a transcript JSONL file contains a `user.message` event
- **THEN** the system extracts content, timestamp, and parentId

#### Scenario: Parse assistant message event
- **WHEN** a transcript JSONL file contains an `assistant.message` event
- **THEN** the system extracts content, toolRequests, reasoningText, messageId, timestamp

#### Scenario: Parse tool execution events
- **WHEN** a transcript JSONL file contains `tool.execution_start` or `tool.execution_complete` events
- **THEN** the system extracts toolCallId, toolName, arguments, success status, timestamps

#### Scenario: Handle malformed JSON lines
- **WHEN** a JSONL line contains invalid JSON
- **THEN** the system skips the line and continues processing remaining lines

### Requirement: Parse chatSessions JSON format
The system SHALL parse VS Code Copilot Chat session files (`chatSessions/*.json`) from workspaceStorage to extract session metadata.

#### Scenario: Extract model information
- **WHEN** a chatSessions JSON file contains requests with modelId and details
- **THEN** the system extracts model name (e.g., `copilot/gpt-5-codex`) and display name (e.g., `GPT-5-Codex (Preview)`)

#### Scenario: Extract thinking tokens
- **WHEN** a chatSessions JSON file contains thinking objects with tokens field
- **THEN** the system extracts the thinking token count

#### Scenario: Extract session metadata
- **WHEN** a chatSessions JSON file contains requesterUsername, responderUsername, initialLocation
- **THEN** the system extracts user and session metadata

### Requirement: Parse debug-logs JSONL format
The system SHALL parse VS Code Copilot Chat debug log files (`debug-logs/{sessionId}/main.jsonl`) from workspaceStorage to extract actual token usage.

#### Scenario: Extract LLM request token counts
- **WHEN** a debug log JSONL file contains `llm_request` events
- **THEN** the system extracts inputTokens, outputTokens, cachedTokens, model from each event

#### Scenario: Extract session shutdown metrics
- **WHEN** a debug log JSONL file contains `session.shutdown` events
- **THEN** the system extracts per-model usage (inputTokens, outputTokens, cacheReadTokens, cacheWriteTokens) and totalNanoAiu

#### Scenario: Handle missing debug log
- **WHEN** the debug log directory does not exist for a session
- **THEN** the system returns nil and falls back to estimation

### Requirement: Discover workspaceStorage paths
The system SHALL discover VS Code Copilot session files across different VS Code variants and operating systems.

#### Scenario: Discover on macOS
- **WHEN** running on macOS
- **THEN** the system searches `~/Library/Application Support/Code/User/workspaceStorage/*/GitHub.copilot-chat/`

#### Scenario: Discover on Linux
- **WHEN** running on Linux
- **THEN** the system searches `~/.config/Code/User/workspaceStorage/*/GitHub.copilot-chat/`

#### Scenario: Discover on Windows
- **WHEN** running on Windows
- **THEN** the system searches `%APPDATA%/Code/User/workspaceStorage/*/GitHub.copilot-chat/`

#### Scenario: Support VS Code variants
- **WHEN** VS Code variant (Insiders, VSCodium, Cursor) is installed
- **THEN** the system also searches the variant's workspaceStorage directory

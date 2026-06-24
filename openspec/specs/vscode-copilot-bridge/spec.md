# vscode-copilot-bridge Specification

## Purpose
TBD - created by archiving change vscode-copilot-support. Update Purpose after archive.
## Requirements
### Requirement: Monitor workspaceStorage for changes
The VS Code Extension SHALL monitor workspaceStorage directories for new or modified Copilot session files.

#### Scenario: Detect new session file
- **WHEN** a new transcript or chatSessions file is created in workspaceStorage
- **THEN** the extension triggers tt record command with the session ID

#### Scenario: Detect modified session file
- **WHEN** an existing session file is modified
- **THEN** the extension triggers tt record command to update the session

### Requirement: Call tt record CLI
The VS Code Extension SHALL call `tt record prompt` and `tt record response` commands to record Copilot activity.

#### Scenario: Record prompt event
- **WHEN** a new user.message event is detected in transcripts
- **THEN** the extension calls `tt record prompt --tool vscode-copilot --session <sessionId> --project <projectPath>`

#### Scenario: Record response event
- **WHEN** a new assistant.message event is detected in transcripts
- **THEN** the extension calls `tt record response --tool vscode-copilot --session <sessionId> --model <modelId>`

#### Scenario: Handle tt CLI not found
- **WHEN** the tt CLI is not installed or not in PATH
- **THEN** the extension logs a warning and continues without blocking VS Code

### Requirement: Activate on startup
The VS Code Extension SHALL activate when VS Code starts and Copilot Chat extension is present.

#### Scenario: Extension activation
- **WHEN** VS Code starts with GitHub Copilot Chat extension installed
- **THEN** the tt bridge extension activates and begins monitoring

#### Scenario: Extension activation without Copilot
- **WHEN** VS Code starts without GitHub Copilot Chat extension
- **THEN** the tt bridge extension activates but does not monitor (no Copilot sessions)


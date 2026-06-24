## ADDED Requirements

### Requirement: Extract actual tokens from debug logs
The system SHALL extract actual token counts from debug log `llm_request` events when available.

#### Scenario: Extract tokens from llm_request events
- **WHEN** debug log contains `llm_request` events with `attrs.inputTokens` and `attrs.outputTokens`
- **THEN** the system returns actual input and output token counts per model

#### Scenario: Extract cached tokens
- **WHEN** debug log contains `llm_request` events with `attrs.cachedTokens`
- **THEN** the system returns cached token count for cache-aware pricing

#### Scenario: Sum multiple LLM requests
- **WHEN** a session has multiple `llm_request` events (agent mode)
- **THEN** the system sums all token counts across events

### Requirement: Estimate tokens from text content
The system SHALL estimate token counts from text content when actual counts are unavailable.

#### Scenario: Character-to-token ratio estimation
- **WHEN** actual token counts are not available
- **THEN** the system estimates tokens using 0.25 tokens/char ratio (4 characters per token)

#### Scenario: Model-specific ratio
- **WHEN** model name matches a known model in tokenEstimators
- **THEN** the system uses the model-specific character-to-token ratio

### Requirement: Estimate input:output ratio from tool calls
The system SHALL estimate input:output token ratio based on tool call count when only output tokens are known.

#### Scenario: Heavy agent session
- **WHEN** session has 20+ tool calls
- **THEN** the system uses 130:1 input:output ratio

#### Scenario: Medium agent session
- **WHEN** session has 5-19 tool calls
- **THEN** the system uses 50:1 input:output ratio

#### Scenario: Simple chat session
- **WHEN** session has fewer than 5 tool calls
- **THEN** the system uses 10:1 input:output ratio

### Requirement: Calculate estimated cost
The system SHALL calculate estimated cost in USD based on model usage and pricing data.

#### Scenario: Calculate cost with known pricing
- **WHEN** model has pricing data in modelPricing
- **THEN** the system calculates cost = (inputTokens × inputCostPerMillion + outputTokens × outputCostPerMillion) / 1,000,000

#### Scenario: Handle unknown model pricing
- **WHEN** model has no pricing data
- **THEN** the system returns $0 cost for that model

#### Scenario: Apply cache-aware pricing
- **WHEN** cachedReadTokens are available
- **THEN** the system applies reduced cachedInputCostPerMillion rate to cached tokens

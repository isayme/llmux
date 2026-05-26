package convert

//
// Anthropic Messages API types.
// Source: https://github.com/anthropics/anthropic-sdk-typescript (npm: @anthropic-ai/sdk)
// Names aligned with SDK's Message/* interfaces, prefixed with Anthropic.
//

// AnthropicMessageRequest represents the body for POST /v1/messages.
// Maps to SDK's MessageCreateParamsBase.
type AnthropicMessageRequest struct {
	// The model that will complete your prompt. See
	// https://docs.anthropic.com/en/docs/models-overview
	Model string `json:"model"`

	// The maximum number of tokens to generate before stopping.
	// This field is required by the Anthropic Messages API.
	MaxTokens int `json:"max_tokens"`

	// Input messages. Messages alternate between "user" and "assistant" roles.
	// The first message must have role "user".
	Messages []AnthropicMessageParam `json:"messages"`

	// An optional description of what the model should do, with higher precedence
	// than messages. Equivalent to OpenAI's "system" message.
	System string `json:"system,omitempty"`

	// Configuration for enabling Claude's extended thinking.
	// When enabled, responses include "thinking" content blocks.
	Thinking *AnthropicThinkingConfigParam `json:"thinking,omitempty"`

	// How the model should use the provided tools.
	ToolChoice *AnthropicToolChoice `json:"tool_choice,omitempty"`

	// Definitions of tools that the model may use.
	Tools []AnthropicTool `json:"tools,omitempty"`

	// Custom text sequences that will cause the model to stop generating.
	StopSequences []string `json:"stop_sequences,omitempty"`

	// Whether to incrementally stream the response using server-sent events.
	Stream *bool `json:"stream,omitempty"`

	// Amount of randomness injected into the response, between 0 and 1.
	Temperature *float64 `json:"temperature,omitempty"`

	// Use nucleus sampling. Only sample from the top P percent of tokens.
	TopP *float64 `json:"top_p,omitempty"`

	// Only sample from the top K options for each subsequent token.
	TopK *int `json:"top_k,omitempty"`

	// An object describing metadata about the request.
	Metadata *AnthropicMetadata `json:"metadata,omitempty"`

	// Output configuration for structured outputs.
	OutputConfig *AnthropicOutputConfig `json:"output_config,omitempty"`
}

// AnthropicMessageParam represents a single message in the request.
// Maps to SDK's MessageParam.
type AnthropicMessageParam struct {
	// The role of the message. One of "user" or "assistant".
	Role string `json:"role"`
	// The content of the message. Can be a string or an array of content blocks.
	Content interface{} `json:"content"`
}

// AnthropicThinkingConfigParam configures Claude's extended thinking.
// One of: enabled (with budget_tokens), disabled, or adaptive.
type AnthropicThinkingConfigParam struct {
	// The type of thinking configuration. "enabled", "disabled", or "adaptive".
	Type string `json:"type"`
	// Determines how many tokens Claude can use for internal reasoning.
	// Must be >=1024 and less than max_tokens. Only for type "enabled".
	BudgetTokens int `json:"budget_tokens,omitempty"`
	// Controls how thinking content appears. "summarized" or "omitted". Default: "summarized".
	Display string `json:"display,omitempty"`
}

// AnthropicToolChoice controls how the model should use provided tools.
// Maps to SDK's ToolChoice (ToolChoiceAuto | ToolChoiceAny | ToolChoiceTool | ToolChoiceNone).
type AnthropicToolChoice struct {
	// The type of tool choice. "auto", "any", "tool", or "none".
	Type string `json:"type"`
	// The name of the tool to use (when type is "tool").
	Name string `json:"name,omitempty"`
	// Whether to disable parallel tool use. Defaults to false.
	DisableParallelToolUse *bool `json:"disable_parallel_tool_use,omitempty"`
}

// AnthropicTool defines a tool that the model may use.
// Maps to SDK's Tool interface.
type AnthropicTool struct {
	// Name of the tool. This is how the tool will be called by the model.
	Name string `json:"name"`
	// [JSON Schema](https://json-schema.org/draft/2020-12) for this tool's input.
	InputSchema *AnthropicToolInputSchema `json:"input_schema"`

	// Description of what this tool does.
	Description string `json:"description,omitempty"`

	// Create a cache control breakpoint at this content block.
	CacheControl *AnthropicCacheControlEphemeral `json:"cache_control,omitempty"`

	// If true, tool will not be included in initial system prompt.
	DeferLoading *bool `json:"defer_loading,omitempty"`

	// When true, guarantees schema validation on tool names and inputs.
	Strict *bool `json:"strict,omitempty"`

	// Indicates who can invoke this tool: "direct", "code_execution_20250825", etc.
	AllowedCallers []string `json:"allowed_callers,omitempty"`

	// Enable eager input streaming for this tool.
	EagerInputStreaming *bool `json:"eager_input_streaming,omitempty"`

	// Examples of valid inputs for the tool.
	InputExamples []map[string]interface{} `json:"input_examples,omitempty"`

	// The type of the tool. Set to "custom" for user-defined tools, null for built-in.
	Type string `json:"type,omitempty"`
}

// AnthropicToolInputSchema is the JSON Schema for a tool's input.
type AnthropicToolInputSchema struct {
	// The type of the schema, always "object".
	Type string `json:"type"`
	// The properties of the object.
	Properties map[string]interface{} `json:"properties,omitempty"`
	// The required fields.
	Required []string `json:"required,omitempty"`
}

// AnthropicCacheControlEphemeral represents an ephemeral cache control breakpoint.
type AnthropicCacheControlEphemeral struct {
	// Always "ephemeral".
	Type string `json:"type"`
	// The TTL for the cache entry. "5m" (default) or "1h".
	TTL string `json:"ttl,omitempty"`
}

// AnthropicMetadata represents metadata about the request.
type AnthropicMetadata struct {
	// An external identifier for the user who is associated with the request.
	UserID string `json:"user_id,omitempty"`
}

// AnthropicOutputConfig configures structured outputs from the model.
type AnthropicOutputConfig struct {
	// All possible effort levels: "low", "medium", "high", "xhigh", "max".
	Effort string `json:"effort,omitempty"`
	// A schema to specify Claude's output format. See structured outputs docs.
	Format *AnthropicJSONOutputFormat `json:"format,omitempty"`
}

// AnthropicJSONOutputFormat defines a JSON Schema for structured outputs.
type AnthropicJSONOutputFormat struct {
	// The JSON schema of the format.
	Schema map[string]interface{} `json:"schema"`
	// Always "json_schema".
	Type string `json:"type"`
}

// -- Response types --

// AnthropicMessage represents a message response from the Claude API.
// Maps to SDK's Message interface.
type AnthropicMessage struct {
	// Unique object identifier. Format and length may change over time.
	ID string `json:"id"`
	// Object type. For Messages, this is always "message".
	Type string `json:"type"`
	// Conversational role of the generated message. Always "assistant".
	Role string `json:"role"`
	// The model that handled the request.
	Model string `json:"model"`
	// Content generated by the model. An array of content blocks.
	Content []AnthropicContentBlock `json:"content"`
	// The reason that we stopped generating.
	// "end_turn", "max_tokens", "stop_sequence", "tool_use", "pause_turn", "refusal".
	StopReason *string `json:"stop_reason"`
	// Which custom stop sequence was generated, if any.
	StopSequence *string `json:"stop_sequence"`
	// Structured information about a refusal.
	StopDetails *AnthropicRefusalStopDetails `json:"stop_details,omitempty"`
	// Information about the container used in the request (for code execution tool).
	Container *AnthropicContainer `json:"container,omitempty"`
	// Billing and rate-limit usage.
	Usage *AnthropicUsage `json:"usage"`
}

// AnthropicRefusalStopDetails provides structured information about a refusal.
type AnthropicRefusalStopDetails struct {
	// The policy category that triggered the refusal. "cyber", "bio", or null.
	Category *string `json:"category"`
	// Human-readable explanation of the refusal.
	Explanation *string `json:"explanation"`
	// Always "refusal".
	Type string `json:"type"`
}

// AnthropicContainer contains information about the container used for code execution.
type AnthropicContainer struct {
	// Identifier for the container used in this request.
	ID string `json:"id"`
	// The time at which the container will expire.
	ExpiresAt string `json:"expires_at"`
}

// AnthropicContentBlock represents a single content block in the response.
// The type field determines which fields are populated.
// Types: "text", "thinking", "redacted_thinking", "tool_use", "server_tool_use",
// "web_search_result", "web_fetch_result", "code_execution_result", "bash_code_execution_result",
// "text_editor_code_execution_result", "tool_search_result", "container_upload".
// Maps to SDK's ContentBlock union type.
type AnthropicContentBlock struct {
	// The type of content block.
	Type string `json:"type"`

	// Text content (for "text" blocks).
	Text string `json:"text,omitempty"`

	// Citations supporting the text block (for "text" blocks).
	Citations []AnthropicTextCitation `json:"citations,omitempty"`

	// Claude's internal thinking content (for "thinking" blocks).
	Thinking string `json:"thinking,omitempty"`

	// Cryptographic signature for the thinking block (for "thinking" blocks).
	// Required for multi-turn continuity with extended thinking.
	Signature string `json:"signature,omitempty"`

	// Redacted thinking data (for "redacted_thinking" blocks).
	Data string `json:"data,omitempty"`

	// Unique identifier for the tool use block (for "tool_use" blocks).
	ID string `json:"id,omitempty"`

	// The name of the tool being called (for "tool_use" or "server_tool_use" blocks).
	Name string `json:"name,omitempty"`

	// The input parameters for the tool call (for "tool_use" blocks).
	Input interface{} `json:"input,omitempty"`

	// Info about who invoked the tool (for "server_tool_use" blocks).
	Caller interface{} `json:"caller,omitempty"`

	// The tool_use_id this result responds to (for tool result blocks in input).
	ToolUseID string `json:"tool_use_id,omitempty"`

	// Whether the tool result contains an error (for tool result blocks).
	IsError *bool `json:"is_error,omitempty"`

	// The content of the tool result (for tool result blocks).
	Content interface{} `json:"content,omitempty"`

	// Source for image blocks (for "image" blocks in input).
	Source interface{} `json:"source,omitempty"`

	// File ID for container upload blocks.
	FileID string `json:"file_id,omitempty"`

	// Return code for code execution result blocks.
	ReturnCode int `json:"return_code,omitempty"`

	// Stdout for code execution result blocks.
	Stdout string `json:"stdout,omitempty"`

	// Stderr for code execution result blocks.
	Stderr string `json:"stderr,omitempty"`

	// Encrypted stdout (for encrypted code execution results).
	EncryptedStdout string `json:"encrypted_stdout,omitempty"`

	// Error code for code execution errors.
	ErrorCode string `json:"error_code,omitempty"`

	// Error message for code execution errors.
	ErrorMessage *string `json:"error_message,omitempty"`

	// File type for text editor results.
	FileType string `json:"file_type,omitempty"`

	// Number of lines for text editor view results.
	NumLines *int `json:"num_lines,omitempty"`

	// Start line for text editor view results.
	StartLine *int `json:"start_line,omitempty"`

	// Total lines for text editor view results.
	TotalLines *int `json:"total_lines,omitempty"`

	// Whether the text editor result is a file update.
	IsFileUpdate *bool `json:"is_file_update,omitempty"`

	// Lines for text editor str_replace results.
	Lines []string `json:"lines,omitempty"`

	// Old line count for text editor results.
	OldLines *int `json:"old_lines,omitempty"`

	// New line count for text editor results.
	NewLines *int `json:"new_lines,omitempty"`

	// Old start line for text editor results.
	OldStart *int `json:"old_start,omitempty"`

	// New start line for text editor results.
	NewStart *int `json:"new_start,omitempty"`

	// Tool references for tool search results.
	ToolReferences []AnthropicToolReferenceBlock `json:"tool_references,omitempty"`

	// The encrypted content for compaction items.
	EncryptedContent string `json:"encrypted_content,omitempty"`

	// Title for document blocks.
	Title string `json:"title,omitempty"`

	// Document citation configuration.
	DocumentCitations *AnthropicCitationsConfig `json:"citations_config,omitempty"`
}

// AnthropicTextCitation represents a citation within a text block.
type AnthropicTextCitation struct {
	// The type of citation location.
	Type string `json:"type"`
	// The cited text.
	CitedText string `json:"cited_text"`
	// The document index.
	DocumentIndex int `json:"document_index"`
	// The document title.
	DocumentTitle *string `json:"document_title"`
	// The file ID if cited from an uploaded file.
	FileID *string `json:"file_id"`
	// End char index (for "char_location" type).
	EndCharIndex int `json:"end_char_index,omitempty"`
	// Start char index (for "char_location" type).
	StartCharIndex int `json:"start_char_index,omitempty"`
	// End page number (for "page_location" type).
	EndPageNumber int `json:"end_page_number,omitempty"`
	// Start page number (for "page_location" type).
	StartPageNumber int `json:"start_page_number,omitempty"`
	// End block index (for "content_block_location" type).
	EndBlockIndex int `json:"end_block_index,omitempty"`
	// Start block index (for "content_block_location" type).
	StartBlockIndex int `json:"start_block_index,omitempty"`
	// URL for web search citations.
	URL string `json:"url,omitempty"`
	// Encrypted index for web search citations.
	EncryptedIndex string `json:"encrypted_index,omitempty"`
	// Source for search result citations.
	Source string `json:"source,omitempty"`
	// Search result index.
	SearchResultIndex int `json:"search_result_index,omitempty"`
}

// AnthropicCitationsConfig configures citations for a document.
type AnthropicCitationsConfig struct {
	// Whether citations are enabled.
	Enabled bool `json:"enabled"`
}

// AnthropicToolReferenceBlock references a tool in tool search results.
type AnthropicToolReferenceBlock struct {
	// The name of the tool being referenced.
	ToolName string `json:"tool_name"`
	// Always "tool_reference".
	Type string `json:"type"`
}

// AnthropicUsage represents billing and rate-limit usage.
// Maps to SDK's Usage interface on the Message type.
type AnthropicUsage struct {
	// The number of input tokens used.
	InputTokens int `json:"input_tokens"`
	// The number of output tokens used.
	OutputTokens int `json:"output_tokens"`
	// The number of input tokens used to create the cache entry.
	CacheCreationInputTokens int `json:"cache_creation_input_tokens,omitempty"`
	// The number of input tokens read from the cache.
	CacheReadInputTokens int `json:"cache_read_input_tokens,omitempty"`
	// The number of server tool requests (web_search, web_fetch).
	ServerToolUse *AnthropicServerToolUsage `json:"server_tool_use,omitempty"`
}

// AnthropicServerToolUsage tracks server-side tool usage.
type AnthropicServerToolUsage struct {
	// The number of web fetch tool requests.
	WebFetchRequests int `json:"web_fetch_requests"`
	// The number of web search tool requests.
	WebSearchRequests int `json:"web_search_requests"`
}

// -- SSE (streaming) types --

// AnthropicRawMessageStreamEvent represents a streaming SSE event from the Messages API.
// Maps to SDK's RawMessageStreamEvent union type.
type AnthropicRawMessageStreamEvent struct {
	// The type of stream event.
	// "message_start", "message_delta", "message_stop",
	// "content_block_start", "content_block_delta", "content_block_stop".
	Type string `json:"type"`

	// The index of the content block (for content_block_* events).
	Index *int `json:"index,omitempty"`

	// The message payload (for "message_start" event).
	Message *AnthropicMessage `json:"message,omitempty"`

	// The content block payload (for "content_block_start" event).
	ContentBlock *AnthropicContentBlock `json:"content_block,omitempty"`

	// The delta payload (for "content_block_delta" and "message_delta" events).
	Delta *AnthropicRawContentBlockDelta `json:"delta,omitempty"`

	// Usage information (for "message_delta" event).
	Usage *AnthropicMessageDeltaUsage `json:"usage,omitempty"`
}

// AnthropicRawContentBlockDelta represents a delta update for a content block.
// Maps to SDK's RawContentBlockDelta union type.
type AnthropicRawContentBlockDelta struct {
	// The type of delta. "text_delta", "input_json_delta", "citations_delta",
	// "thinking_delta", "signature_delta".
	Type string `json:"type,omitempty"`

	// Text content (for "text_delta").
	Text string `json:"text,omitempty"`

	// Partial JSON (for "input_json_delta").
	PartialJSON string `json:"partial_json,omitempty"`

	// Thinking content (for "thinking_delta").
	Thinking string `json:"thinking,omitempty"`

	// Signature (for "signature_delta").
	Signature string `json:"signature,omitempty"`

	// Citation info (for "citations_delta").
	Citation interface{} `json:"citation,omitempty"`

	// Stop reason (for message_delta).
	StopReason *string `json:"stop_reason,omitempty"`

	// Stop sequence (for message_delta).
	StopSequence *string `json:"stop_sequence,omitempty"`

	// Structured refusal info (for message_delta).
	StopDetails *AnthropicRefusalStopDetails `json:"stop_details,omitempty"`

	// Container info (for message_delta).
	Container *AnthropicContainer `json:"container,omitempty"`
}

// AnthropicMessageDeltaUsage represents cumulative token usage in a message_delta event.
// Maps to SDK's MessageDeltaUsage.
type AnthropicMessageDeltaUsage struct {
	// Cumulative cache creation input tokens.
	CacheCreationInputTokens *int `json:"cache_creation_input_tokens"`
	// Cumulative cache read input tokens.
	CacheReadInputTokens *int `json:"cache_read_input_tokens"`
	// Cumulative input tokens.
	InputTokens *int `json:"input_tokens"`
	// Cumulative output tokens.
	OutputTokens int `json:"output_tokens"`
	// The number of server tool requests.
	ServerToolUse *AnthropicServerToolUsage `json:"server_tool_use,omitempty"`
}

// -- Built-in Anthropic tool type definitions --

// AnthropicToolBash20250124 represents the bash tool (type "bash_20250124").
type AnthropicToolBash20250124 struct {
	Name        string                        `json:"name"`
	Type        string                        `json:"type"`
	CacheControl *AnthropicCacheControlEphemeral `json:"cache_control,omitempty"`
	DeferLoading *bool                         `json:"defer_loading,omitempty"`
	Strict      *bool                          `json:"strict,omitempty"`
	InputExamples []map[string]interface{}     `json:"input_examples,omitempty"`
	AllowedCallers []string                    `json:"allowed_callers,omitempty"`
}

// AnthropicToolTextEditor20250124 represents the text editor tool (type "text_editor_20250124").
type AnthropicToolTextEditor20250124 struct {
	Name        string                        `json:"name"`
	Type        string                        `json:"type"`
	CacheControl *AnthropicCacheControlEphemeral `json:"cache_control,omitempty"`
	DeferLoading *bool                         `json:"defer_loading,omitempty"`
	Strict      *bool                          `json:"strict,omitempty"`
	InputExamples []map[string]interface{}     `json:"input_examples,omitempty"`
	AllowedCallers []string                    `json:"allowed_callers,omitempty"`
}

// AnthropicWebSearchTool20250305 represents the web search tool.
type AnthropicWebSearchTool20250305 struct {
	Name        string                        `json:"name"`
	Type        string                        `json:"type"`
	CacheControl *AnthropicCacheControlEphemeral `json:"cache_control,omitempty"`
	DeferLoading *bool                         `json:"defer_loading,omitempty"`
	Strict      *bool                          `json:"strict,omitempty"`
	AllowedCallers []string                    `json:"allowed_callers,omitempty"`
}

// AnthropicWebFetchTool20250910 represents the web fetch tool.
type AnthropicWebFetchTool20250910 struct {
	Name        string                        `json:"name"`
	Type        string                        `json:"type"`
	CacheControl *AnthropicCacheControlEphemeral `json:"cache_control,omitempty"`
	DeferLoading *bool                         `json:"defer_loading,omitempty"`
	Strict      *bool                          `json:"strict,omitempty"`
	AllowedCallers []string                    `json:"allowed_callers,omitempty"`
}

// AnthropicCodeExecutionTool20260120 represents the code execution tool with daemon mode.
type AnthropicCodeExecutionTool20260120 struct {
	Name        string                        `json:"name"`
	Type        string                        `json:"type"`
	CacheControl *AnthropicCacheControlEphemeral `json:"cache_control,omitempty"`
	DeferLoading *bool                         `json:"defer_loading,omitempty"`
	Strict      *bool                          `json:"strict,omitempty"`
	AllowedCallers []string                    `json:"allowed_callers,omitempty"`
}

// -- Constants --

// Anthropic SSE event name constants.
const (
	AnthropicSSEMessageStartEvent      = "message_start"
	AnthropicSSEContentBlockStartEvent = "content_block_start"
	AnthropicSSEContentBlockDeltaEvent = "content_block_delta"
	AnthropicSSEContentBlockStopEvent  = "content_block_stop"
	AnthropicSSEMessageDeltaEvent      = "message_delta"
	AnthropicSSEMessageStopEvent       = "message_stop"
	AnthropicSSEPingEvent              = "ping"
)

// Anthropic stop reason constants for Response.StopReason.
const (
	AnthropicStopReasonEndTurn      = "end_turn"
	AnthropicStopReasonMaxTokens    = "max_tokens"
	AnthropicStopReasonStopSequence = "stop_sequence"
	AnthropicStopReasonToolUse      = "tool_use"
	AnthropicStopReasonPauseTurn    = "pause_turn"
	AnthropicStopReasonRefusal      = "refusal"
)

// Anthropic content / delta type constants.
const (
	AnthropicContentTypeText            = "text"
	AnthropicContentTypeThinking        = "thinking"
	AnthropicContentTypeRedacted        = "redacted_thinking"
	AnthropicContentTypeToolUse         = "tool_use"
	AnthropicContentTypeServerToolUse   = "server_tool_use"
	AnthropicContentTypeToolResult      = "tool_result"
	AnthropicContentTypeImage           = "image"
	AnthropicContentTypeDocument        = "document"
	AnthropicContentTypeWebSearchResult = "web_search_result"
	AnthropicContentTypeWebFetchResult  = "web_fetch_result"
	AnthropicContentTypeContainerUpload = "container_upload"
	AnthropicDeltaTypeTextDelta         = "text_delta"
	AnthropicDeltaTypeThinkingDelta     = "thinking_delta"
	AnthropicDeltaTypeInputJSONDelta    = "input_json_delta"
	AnthropicDeltaTypeSignatureDelta    = "signature_delta"
	AnthropicDeltaTypeCitationsDelta    = "citations_delta"
)

// Anthropic object / role constants.
const (
	AnthropicObjectMessage = "message"
	AnthropicRoleUser      = "user"
	AnthropicRoleAssistant = "assistant"
)

// Anthropic tool type constants.
const (
	AnthropicToolTypeCustom              = "custom"
	AnthropicToolTypeBash20250124        = "bash_20250124"
	AnthropicToolTypeTextEditor20250124  = "text_editor_20250124"
	AnthropicToolTypeTextEditor20250429  = "text_editor_20250429"
	AnthropicToolTypeTextEditor20250728  = "text_editor_20250728"
	AnthropicToolTypeCodeExec20250522    = "code_execution_20250522"
	AnthropicToolTypeCodeExec20250825    = "code_execution_20250825"
	AnthropicToolTypeCodeExec20260120    = "code_execution_20260120"
	AnthropicToolTypeMemory20250818      = "memory_20250818"
	AnthropicToolTypeWebSearch20250305   = "web_search_20250305"
	AnthropicToolTypeWebFetch20250910    = "web_fetch_20250910"
	AnthropicToolTypeWebSearch20260209   = "web_search_20260209"
	AnthropicToolTypeWebFetch20260209    = "web_fetch_20260209"
	AnthropicToolTypeWebFetch20260309    = "web_fetch_20260309"
	AnthropicToolTypeToolSearchBm25      = "tool_search_tool_bm25_20251119"
	AnthropicToolTypeToolSearchRegex     = "tool_search_tool_regex_20251119"
)

// Anthropic thinking configuration constants.
const (
	AnthropicThinkingEnabled  = "enabled"
	AnthropicThinkingDisabled = "disabled"
	AnthropicThinkingAdaptive = "adaptive"
)

// Anthropic cache control TTL constants.
const (
	AnthropicCacheTTL5m = "5m"
	AnthropicCacheTL1h  = "1h"
)

// -- Backward-compatible aliases for legacy type names. --

// Deprecated: use AnthropicMessageRequest.
type AnthropicRequest = AnthropicMessageRequest

// Deprecated: use AnthropicMessageParam.
// NOTE: AnthropicMessage is the new response type name (SDK-aligned).
// The old request-message type is now AnthropicMessageParam.
// type AnthropicMessage = AnthropicMessageParam  -- conflicts with the response type

// Deprecated: use AnthropicMessage.
type AnthropicResponse = AnthropicMessage

// Deprecated: use AnthropicRawMessageStreamEvent.
type AnthropicSSEEvent = AnthropicRawMessageStreamEvent

// Deprecated: use AnthropicRawContentBlockDelta.
type AnthropicSSEDelta = AnthropicRawContentBlockDelta

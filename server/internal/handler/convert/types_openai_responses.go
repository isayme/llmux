package convert

//
// OpenAI Responses API types.
// Source: https://github.com/openai/openai-node (npm: openai)
// Names aligned with SDK's Response/* interfaces, prefixed with OpenAI.
//

// OpenAIResponsesRequest represents the body for POST /v1/responses.
// Maps to SDK's ResponseCreateParamsBase.
type OpenAIResponsesRequest struct {
	// Model ID used to generate the response, like "gpt-4o" or "o3".
	Model string `json:"model"`

	// Text input or structured input items for the model.
	Input interface{} `json:"input"`

	// System-level instructions for the model, equivalent to a "system" message in Chat Completions.
	Instructions string `json:"instructions,omitempty"`

	// An upper bound for the number of tokens that can be generated for a response.
	MaxOutputTokens *int `json:"max_output_tokens,omitempty"`

	// What sampling temperature to use, between 0 and 2.
	Temperature *float64 `json:"temperature,omitempty"`

	// An alternative to sampling with temperature, called nucleus sampling.
	TopP *float64 `json:"top_p,omitempty"`

	// Whether to stream back partial progress via SSE events.
	Stream *bool `json:"stream,omitempty"`

	// Up to 4 sequences where the API will stop generating further tokens.
	Stop []string `json:"stop,omitempty"`

	// Whether to store the response for later retrieval.
	Store *bool `json:"store,omitempty"`

	// Set of 16 key-value pairs that can be attached to an object.
	Metadata map[string]string `json:"metadata,omitempty"`

	// How the model should select which tool to use.
	ToolChoice *OpenAIResponsesToolChoice `json:"tool_choice,omitempty"`

	// An array of tools the model may call while generating a response.
	Tools []OpenAIResponsesTool `json:"tools,omitempty"`

	// Whether to allow the model to run tool calls in parallel.
	ParallelToolCalls *bool `json:"parallel_tool_calls,omitempty"`

	// The unique ID of the previous response for multi-turn conversations.
	PreviousResponseID string `json:"previous_response_id,omitempty"`

	// Configuration for reasoning models (gpt-5 and o-series only).
	Reasoning *OpenAIReasoning `json:"reasoning,omitempty"`

	// Configuration options for a text response. Can be plain text or structured JSON.
	Text *OpenAIResponsesTextConfig `json:"text,omitempty"`

	// The truncation strategy. "auto" or "disabled".
	Truncation string `json:"truncation,omitempty"`

	// Used by OpenAI to cache responses for similar requests.
	PromptCacheKey string `json:"prompt_cache_key,omitempty"`

	// The retention policy for the prompt cache.
	PromptCacheRetention string `json:"prompt_cache_retention,omitempty"`

	// An integer between 0 and 20 for log probabilities at each token position.
	TopLogprobs *int `json:"top_logprobs,omitempty"`

	// Whether to run the model response in the background.
	Background *bool `json:"background,omitempty"`

	// A stable identifier used to help detect users that may be violating usage policies.
	SafetyIdentifier string `json:"safety_identifier,omitempty"`

	// Reference to a prompt template and its variables.
	Prompt *OpenAIResponsesPrompt `json:"prompt,omitempty"`

	// Specifies the latency tier. "auto", "default", "flex", "scale", "priority".
	ServiceTier string `json:"service_tier,omitempty"`

	// Number between -2.0 and 2.0 for frequency penalty.
	FrequencyPenalty *float64 `json:"frequency_penalty,omitempty"`

	// Number between -2.0 and 2.0 for presence penalty.
	PresencePenalty *float64 `json:"presence_penalty,omitempty"`

	// Deprecated: A stable identifier for your end-users.
	// Deprecated: use SafetyIdentifier or PromptCacheKey instead.
	User string `json:"user,omitempty"`
}

// OpenAIResponsesToolChoice controls which tool the model selects.
type OpenAIResponsesToolChoice struct {
	// The type of tool choice. "auto", "required", "none", "function", "custom", etc.
	Type string `json:"type,omitempty"`
	// Function name (when type is "function").
	Name string `json:"name,omitempty"`
}

// OpenAIResponsesTool represents a tool available to the model.
type OpenAIResponsesTool struct {
	// The type of the tool.
	// "function", "custom", "web_search", "file_search", "computer", "computer_use_preview",
	// "shell", "apply_patch", "namespace", "mcp".
	Type string `json:"type"`

	// Function definition (when type is "function").
	Name string `json:"name,omitempty"`
	Parameters map[string]interface{} `json:"parameters,omitempty"`
	Strict *bool `json:"strict,omitempty"`
	Description string `json:"description,omitempty"`
	DeferLoading *bool `json:"defer_loading,omitempty"`

	// File search tool specific fields.
	VectorStoreIDs []string `json:"vector_store_ids,omitempty"`
	MaxNumResults *int `json:"max_num_results,omitempty"`

	// Computer use tool specific fields.
	DisplayHeight int `json:"display_height,omitempty"`
	DisplayWidth int `json:"display_width,omitempty"`
	Environment string `json:"environment,omitempty"`

	// Custom tool specific fields.
	Format *OpenAICustomToolInputFormat `json:"format,omitempty"`

	// MCP tool specific fields.
	McpID string `json:"mcp_id,omitempty"`

	// Shell tool specific fields.
	ShellEnvironment interface{} `json:"environment,omitempty"`

	// Namespace tool specific fields.
	NamespaceTools []OpenAIResponsesTool `json:"tools,omitempty"`
}

// OpenAIResponsesTextConfig configures text or structured JSON response.
type OpenAIResponsesTextConfig struct {
	// The format type. "text" or "json_schema".
	Format *OpenAIResponsesTextFormat `json:"format,omitempty"`
}

// OpenAIResponsesTextFormat specifies the response format for text output.
type OpenAIResponsesTextFormat struct {
	// The type: "text", "json_schema", or "json_object".
	Type string `json:"type"`
	// JSON schema (when type is "json_schema").
	JSONSchema *OpenAIResponseFormatJSONSchema `json:"json_schema,omitempty"`
}

// OpenAIReasoning configures reasoning model behavior.
type OpenAIReasoning struct {
	// Constrains effort on reasoning: "none", "minimal", "low", "medium", "high", "xhigh".
	Effort string `json:"effort,omitempty"`
	// Deprecated: use Summary instead.
	GenerateSummary string `json:"generate_summary,omitempty"`
	// A summary of the reasoning. "auto", "concise", "detailed".
	Summary string `json:"summary,omitempty"`
}

// OpenAIResponsesPrompt references a reusable prompt template.
type OpenAIResponsesPrompt struct {
	// The ID of the prompt template.
	ID string `json:"id"`
	// Variables to fill in the prompt template.
	Variables map[string]string `json:"variables,omitempty"`
}

// -- Input item types --

// OpenAIResponsesInputItem represents a single item in the input array.
type OpenAIResponsesInputItem struct {
	// The type of input item. "message", "function_call_output", "file_search_call", etc.
	Type string `json:"type"`

	// The role (for "message" type). "user", "assistant", "system", "developer".
	Role string `json:"role,omitempty"`

	// The content (for "message" type).
	Content interface{} `json:"content,omitempty"`

	// The call_id (for "function_call_output" type).
	CallID string `json:"call_id,omitempty"`

	// The output (for "function_call_output" type).
	Output string `json:"output,omitempty"`

	// The status of the item.
	Status string `json:"status,omitempty"`
}

// -- Response types --

// OpenAIResponsesResponse represents a model response from the Responses API.
// Maps to SDK's Response interface.
type OpenAIResponsesResponse struct {
	// Unique identifier for this Response.
	ID string `json:"id"`
	// The object type of this resource - always "response".
	Object string `json:"object"`
	// Unix timestamp (in seconds) of when this Response was created.
	CreatedAt int64 `json:"created_at"`
	// The status of the response generation. "completed", "failed", "in_progress",
	// "cancelled", "queued", or "incomplete".
	Status string `json:"status,omitempty"`
	// A concatenation of all text outputs from the response.
	OutputText string `json:"output_text"`
	// Model ID used to generate the response.
	Model string `json:"model"`
	// An array of content items generated by the model.
	Output []OpenAIResponsesOutputItem `json:"output"`
	// An error object returned when the model fails.
	Error *OpenAIResponsesError `json:"error"`
	// Details about why the response is incomplete.
	IncompleteDetails *OpenAIResponsesIncompleteDetails `json:"incomplete_details"`
	// System-level instructions used for this response.
	Instructions *string `json:"instructions"`
	// Set of 16 key-value pairs attached to the object.
	Metadata map[string]string `json:"metadata"`
	// Whether to allow parallel tool calls.
	ParallelToolCalls bool `json:"parallel_tool_calls"`
	// What sampling temperature was used.
	Temperature *float64 `json:"temperature"`
	// How the model selected which tool to use.
	ToolChoice interface{} `json:"tool_choice"`
	// An array of tools the model may call.
	Tools []OpenAIResponsesTool `json:"tools"`
	// What top_p was used.
	TopP *float64 `json:"top_p"`
	// Represents token usage details.
	Usage *OpenAIResponsesUsage `json:"usage,omitempty"`

	// Whether the response runs in the background.
	Background *bool `json:"background,omitempty"`
	// Unix timestamp when the response was completed.
	CompletedAt *int64 `json:"completed_at,omitempty"`
	// The conversation this response belonged to.
	Conversation *OpenAIResponsesConversation `json:"conversation,omitempty"`
	// An upper bound for tokens generated.
	MaxOutputTokens *int `json:"max_output_tokens,omitempty"`
	// The unique ID of the previous response.
	PreviousResponseID string `json:"previous_response_id,omitempty"`
	// Reference to a prompt template.
	Prompt *OpenAIResponsesPrompt `json:"prompt,omitempty"`
	// Prompt cache key.
	PromptCacheKey string `json:"prompt_cache_key,omitempty"`
	// Prompt cache retention policy.
	PromptCacheRetention string `json:"prompt_cache_retention,omitempty"`
	// Reasoning configuration used.
	Reasoning *OpenAIReasoning `json:"reasoning,omitempty"`
	// Safety identifier.
	SafetyIdentifier string `json:"safety_identifier,omitempty"`
	// Service tier used.
	ServiceTier string `json:"service_tier,omitempty"`
	// Text configuration used.
	Text *OpenAIResponsesTextConfig `json:"text,omitempty"`
	// Top logprobs setting.
	TopLogprobs *int `json:"top_logprobs,omitempty"`
	// Truncation strategy used.
	Truncation string `json:"truncation,omitempty"`
	// Deprecated: user identifier.
	User string `json:"user,omitempty"`
}

// OpenAIResponsesError represents an error returned when the model fails.
type OpenAIResponsesError struct {
	// The error code.
	Code *string `json:"code"`
	// The error message.
	Message string `json:"message"`
	// The parameter that caused the error.
	Param *string `json:"param"`
	// The type of error.
	Type string `json:"type"`
}

// OpenAIResponsesIncompleteDetails explains why the response is incomplete.
type OpenAIResponsesIncompleteDetails struct {
	// The reason: "max_output_tokens" or "content_filter".
	Reason string `json:"reason,omitempty"`
}

// OpenAIResponsesConversation identifies the conversation associated with the response.
type OpenAIResponsesConversation struct {
	// The unique ID of the conversation.
	ID string `json:"id"`
}

// -- Output item types --

// OpenAIResponsesOutputItem is a content item generated by the model.
// The type discriminator determines which fields are populated.
// Types: "message", "function_call", "custom_tool_call", "reasoning",
// "computer_call", "file_search_call", "web_search_call", "code_interpreter_call",
// "compaction", "apply_patch_call", "shell_call", etc.
type OpenAIResponsesOutputItem struct {
	// The type of output item.
	Type string `json:"type"`
	// Unique identifier for this output item.
	ID string `json:"id,omitempty"`
	// The status of the item. "in_progress", "completed", "incomplete", "failed".
	Status string `json:"status,omitempty"`
	// The role ("assistant") for "message" type items.
	Role string `json:"role,omitempty"`
	// Content blocks for "message" type items.
	Content []OpenAIResponsesContentBlock `json:"content,omitempty"`

	// The call_id for tool call items.
	CallID string `json:"call_id,omitempty"`
	// The name of the function/tool called.
	Name string `json:"name,omitempty"`
	// The arguments (JSON string) for function calls.
	Arguments string `json:"arguments,omitempty"`
	// The input for custom tool calls.
	Input interface{} `json:"input,omitempty"`

	// For "function_call" items.
	FunctionName string `json:"function_name,omitempty"`
	// For "function_call" items.
	FunctionArguments string `json:"function_arguments,omitempty"`

	// Summary blocks for "reasoning" type items.
	Summary []OpenAIResponsesSummaryBlock `json:"summary,omitempty"`

	// The namespace for namespace tool calls.
	Namespace string `json:"namespace,omitempty"`

	// The output for tool call output items.
	Output string `json:"output,omitempty"`

	// The ID of the entity that created this item.
	CreatedBy string `json:"created_by,omitempty"`

	// The encrypted content for compaction items.
	EncryptedContent string `json:"encrypted_content,omitempty"`

	// Code for code_interpreter_call items.
	Code string `json:"code,omitempty"`
	// Container ID for code_interpreter_call items.
	ContainerID string `json:"container_id,omitempty"`
	// Outputs from code interpreter.
	Outputs interface{} `json:"outputs,omitempty"`

	// Pending safety checks for computer_call items.
	PendingSafetyChecks []OpenAIResponsesPendingSafetyCheck `json:"pending_safety_checks,omitempty"`
	// Computer action for computer_call items.
	Action interface{} `json:"action,omitempty"`
	// Batched actions for computer_use.
	Actions []interface{} `json:"actions,omitempty"`

	// Operation for apply_patch_call items.
	Operation interface{} `json:"operation,omitempty"`
}

// OpenAIResponsesPendingSafetyCheck represents a pending safety check.
type OpenAIResponsesPendingSafetyCheck struct {
	// The ID of the pending safety check.
	ID string `json:"id"`
	// The code/type of the safety check.
	Code string `json:"code,omitempty"`
	// Details about the pending safety check.
	Message string `json:"message,omitempty"`
}

// OpenAIResponsesSummaryBlock represents a reasoning summary.
type OpenAIResponsesSummaryBlock struct {
	// The type: "summary_text".
	Type string `json:"type"`
	// The summary text.
	Text string `json:"text"`
}

// OpenAIResponsesContentBlock represents a content block within a message output item.
// Types: "output_text", "refusal".
type OpenAIResponsesContentBlock struct {
	// The type of content. "output_text" or "refusal".
	Type string `json:"type"`
	// The text content (for "output_text" type).
	Text string `json:"text,omitempty"`
	// The refusal message (for "refusal" type).
	Refusal string `json:"refusal,omitempty"`
	// Annotations for the content block.
	Annotations []interface{} `json:"annotations,omitempty"`
}

// -- Legacy Usage types (Chat Completions compatibility) --

// Usage represents token usage for Chat Completions (legacy format).
// Kept for backward compatibility with existing converter code.
type Usage struct {
	PromptTokens     int `json:"prompt_tokens,omitempty"`
	CompletionTokens int `json:"completion_tokens,omitempty"`
	TotalTokens      int `json:"total_tokens,omitempty"`
}

//  -- Responses API Usage type --

// OpenAIResponsesUsage represents token usage for the Responses API.
// Maps to SDK's ResponseUsage.
type OpenAIResponsesUsage struct {
	// Number of tokens in the input.
	InputTokens int `json:"input_tokens"`
	// Number of tokens in the output.
	OutputTokens int `json:"output_tokens"`
	// Total number of tokens used.
	TotalTokens int `json:"total_tokens"`
	// Breakdown of output tokens.
	OutputTokensDetails *OpenAIResponsesOutputTokensDetails `json:"output_tokens_details,omitempty"`
	// Breakdown of input tokens.
	InputTokensDetails *OpenAIResponsesInputTokensDetails `json:"input_tokens_details,omitempty"`
}

// OpenAIResponsesOutputTokensDetails provides breakdown of output tokens.
type OpenAIResponsesOutputTokensDetails struct {
	// Tokens generated by the model for reasoning.
	ReasoningTokens int `json:"reasoning_tokens,omitempty"`
	// Audio tokens generated by the model.
	AudioTokens int `json:"audio_tokens,omitempty"`
	// Accepted prediction tokens.
	AcceptedPredictionTokens int `json:"accepted_prediction_tokens,omitempty"`
	// Rejected prediction tokens.
	RejectedPredictionTokens int `json:"rejected_prediction_tokens,omitempty"`
	// Text tokens in the output.
	TextTokens int `json:"text_tokens,omitempty"`
}

// OpenAIResponsesInputTokensDetails provides breakdown of input tokens.
type OpenAIResponsesInputTokensDetails struct {
	// Cached tokens in the input.
	CachedTokens int `json:"cached_tokens,omitempty"`
	// Audio tokens in the input.
	AudioTokens int `json:"audio_tokens,omitempty"`
	// Text tokens in the input.
	TextTokens int `json:"text_tokens,omitempty"`
	// Image tokens in the input.
	ImageTokens int `json:"image_tokens,omitempty"`
}

// -- SSE (streaming) event types --

// OpenAIResponsesStreamEvent represents a streaming SSE event from the Responses API.
// Maps to SDK's ResponseStreamEvent union type.
type OpenAIResponsesStreamEvent struct {
	// The type of stream event.
	// "response.created", "response.in_progress", "response.completed",
	// "response.failed", "response.incomplete", "response.output_item.added",
	// "response.output_item.done", "response.content_part.added",
	// "response.output_text.delta", "response.output_text.done",
	// "response.refusal.delta", "response.refusal.done",
	// "response.code_interpreter_call.*", "response.audio.*",
	// "response.function_call_arguments.delta", "response.function_call_arguments.done",
	// "response.file_search_call.*", "error".
	Type string `json:"type"`

	// Delta text for streaming text events.
	Delta string `json:"delta,omitempty"`
	// The partial text for text delta events.
	Text string `json:"text,omitempty"`
	// The refusal content for refusal delta events.
	Refusal string `json:"refusal,omitempty"`

	// The output item for item added/done events.
	Item *OpenAIResponsesOutputItem `json:"item,omitempty"`
	// The content part for content part events.
	Part *OpenAIResponsesContentBlock `json:"part,omitempty"`

	// The response summary for response.* events.
	Response *OpenAIResponsesStreamSummary `json:"response,omitempty"`

	// The item_id for code interpreter call events.
	ItemID string `json:"item_id,omitempty"`
	// The output_index for code interpreter call events.
	OutputIndex int `json:"output_index,omitempty"`
	// The sequence number for ordering streaming events.
	SequenceNumber int `json:"sequence_number,omitempty"`

	// The code delta for code interpreter call code events.
	Code string `json:"code,omitempty"`

	// Audio data for audio streaming events.
	AudioDelta string `json:"audio_delta,omitempty"`

	// The error details for error events.
	Error *OpenAIResponsesError `json:"error,omitempty"`
}

// OpenAIResponsesStreamSummary contains summary properties for response lifecycle events.
type OpenAIResponsesStreamSummary struct {
	// The unique identifier of the response.
	ID string `json:"id,omitempty"`
	// The model used.
	Model string `json:"model,omitempty"`
	// Unix timestamp of creation.
	CreatedAt int64 `json:"created_at,omitempty"`
	// Usage statistics (included in response.completed event).
	Usage *Usage `json:"usage,omitempty"`
	// The status of the response.
	Status string `json:"status,omitempty"`
}

// -- Compacted response type --

// OpenAIResponsesCompactedResponse represents a compacted conversation response.
// Maps to SDK's CompactedResponse.
type OpenAIResponsesCompactedResponse struct {
	// The unique identifier for the compacted response.
	ID string `json:"id"`
	// Unix timestamp (in seconds) when compacted.
	CreatedAt int64 `json:"created_at"`
	// The object type. Always "response.compaction".
	Object string `json:"object"`
	// The compacted list of output items.
	Output []OpenAIResponsesOutputItem `json:"output"`
	// Token accounting for the compaction pass.
	Usage *OpenAIResponsesUsage `json:"usage"`
}

// -- Constants --

// OpenAI Responses API object type constants.
const (
	OpenAIResponsesObject           = "response"
	OpenAIResponsesCompactionObject = "response.compaction"
)

// OpenAI Responses API status constants.
const (
	OpenAIResponsesStatusCompleted   = "completed"
	OpenAIResponsesStatusFailed      = "failed"
	OpenAIResponsesStatusInProgress  = "in_progress"
	OpenAIResponsesStatusCancelled   = "cancelled"
	OpenAIResponsesStatusQueued      = "queued"
	OpenAIResponsesStatusIncomplete  = "incomplete"
)

// OpenAI Responses API output item type constants.
const (
	OpenAIResponsesItemTypeMessage         = "message"
	OpenAIResponsesItemTypeFunctionCall    = "function_call"
	OpenAIResponsesItemTypeFunctionCallOutput = "function_call_output"
	OpenAIResponsesItemTypeCustomToolCall  = "custom_tool_call"
	OpenAIResponsesItemTypeReasoning       = "reasoning"
	OpenAIResponsesItemTypeComputerCall    = "computer_call"
	OpenAIResponsesItemTypeCodeInterpreter = "code_interpreter_call"
	OpenAIResponsesItemTypeFileSearch      = "file_search_call"
	OpenAIResponsesItemTypeWebSearch       = "web_search_call"
	OpenAIResponsesItemTypeCompaction      = "compaction"
	OpenAIResponsesItemTypeApplyPatch      = "apply_patch_call"
	OpenAIResponsesItemTypeShellCall       = "shell_call"
	OpenAIResponsesItemTypeMcpCall         = "mcp_call"
	OpenAIResponsesItemTypeMcpListTools    = "mcp_list_tools"
	OpenAIResponsesItemTypeImageGeneration = "image_generation_call"
)

// OpenAI Responses API content type constants.
const (
	OpenAIResponsesContentTypeText   = "output_text"
	OpenAIResponsesContentTypeRefusal = "refusal"
)

// OpenAI Responses API SSE event constants.
const (
	OpenAIResponsesSSECreated                    = "response.created"
	OpenAIResponsesSSEInProgress                 = "response.in_progress"
	OpenAIResponsesSSECompleted                  = "response.completed"
	OpenAIResponsesSSEFailed                     = "response.failed"
	OpenAIResponsesSSEIncomplete                 = "response.incomplete"
	OpenAIResponsesSSEOutputItemAdded            = "response.output_item.added"
	OpenAIResponsesSSEOutputItemDone             = "response.output_item.done"
	OpenAIResponsesSSEContentPartAdded            = "response.content_part.added"
	OpenAIResponsesSSEContentPartDone            = "response.content_part.done"
	OpenAIResponsesSSEOutputTextDelta            = "response.output_text.delta"
	OpenAIResponsesSSEOutputTextDone             = "response.output_text.done"
	OpenAIResponsesSSERefusalDelta               = "response.refusal.delta"
	OpenAIResponsesSSERefusalDone                = "response.refusal.done"
	OpenAIResponsesSSEFunctionCallArgsDelta      = "response.function_call_arguments.delta"
	OpenAIResponsesSSEFunctionCallArgsDone       = "response.function_call_arguments.done"
	OpenAIResponsesSSECodeInterpreterCodeDelta   = "response.code_interpreter_call_code.delta"
	OpenAIResponsesSSECodeInterpreterCodeDone    = "response.code_interpreter_call_code.done"
	OpenAIResponsesSSECodeInterpreterCompleted   = "response.code_interpreter_call.completed"
	OpenAIResponsesSSEAudioDelta                 = "response.audio.delta"
	OpenAIResponsesSSEAudioDone                  = "response.audio.done"
	OpenAIResponsesSSEAudioTranscriptDelta       = "response.audio.transcript.delta"
	OpenAIResponsesSSEAudioTranscriptDone        = "response.audio.transcript.done"
	OpenAIResponsesSSEError                      = "error"
)

// OpenAI Responses API truncation constants.
const (
	OpenAIResponsesTruncationAuto     = "auto"
	OpenAIResponsesTruncationDisabled = "disabled"
)

// OpenAI Responses API tool choice constants.
const (
	OpenAIResponsesToolChoiceAuto     = "auto"
	OpenAIResponsesToolChoiceRequired = "required"
	OpenAIResponsesToolChoiceNone     = "none"
)

// OpenAI Responses API incomplete details reason constants.
const (
	OpenAIResponsesIncompleteMaxTokens    = "max_output_tokens"
	OpenAIResponsesIncompleteContentFilter = "content_filter"
)

// -- Backward-compatible aliases for legacy constants. --

// Deprecated: use OpenAIResponsesObject.
const ResponsesObject = OpenAIResponsesObject

// Deprecated: use OpenAIResponsesItemTypeMessage.
const ResponsesItemTypeMessage = OpenAIResponsesItemTypeMessage

// Deprecated: use OpenAIResponsesContentTypeText.
const ResponsesContentTypeText = OpenAIResponsesContentTypeText

// Deprecated: use OpenAIResponsesSSECreated.
const ResponsesSSECreated = OpenAIResponsesSSECreated

// Deprecated: use OpenAIResponsesSSEOutputItemAdded.
const ResponsesSSEOutputItemAdded = OpenAIResponsesSSEOutputItemAdded

// Deprecated: use OpenAIResponsesSSEOutputItemDone.
const ResponsesSSEOutputItemDone = OpenAIResponsesSSEOutputItemDone

// Deprecated: use OpenAIResponsesSSEContentPartAdded.
const ResponsesSSEContentPartAdded = OpenAIResponsesSSEContentPartAdded

// Deprecated: use OpenAIResponsesSSEOutputTextDelta.
const ResponsesSSETextDelta = "response.text.delta"

// Deprecated: use OpenAIResponsesSSEOutputTextDone.
const ResponsesSSETextDone = "response.text.done"

// Deprecated: use OpenAIResponsesSSECompleted.
const ResponsesSSEDone = "response.done"

// Deprecated: use OpenAIResponsesSSEError.
const ResponsesSSEError = OpenAIResponsesSSEError

// -- Backward-compatible aliases for legacy type names. --

// Deprecated: use OpenAIResponsesInputItem.
type ResponsesInputItem = OpenAIResponsesInputItem

// Deprecated: use OpenAIResponsesOutputItem.
type ResponsesOutputItem = OpenAIResponsesOutputItem

// Deprecated: use OpenAIResponsesContentBlock.
type ResponsesContentBlock = OpenAIResponsesContentBlock

// Deprecated: use OpenAIResponsesUsage.
type ResponsesUsage = OpenAIResponsesUsage

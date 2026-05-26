package convert

//
// OpenAI Chat Completions API types.
// Source: https://github.com/openai/openai-node (npm: openai)
// Names aligned with SDK's ChatCompletion* interfaces, prefixed with OpenAI.
//

// OpenAIChatCompletionRequest represents the body for POST /chat/completions.
// Maps to SDK's ChatCompletionCreateParamsBase.
type OpenAIChatCompletionRequest struct {
	// Model ID used to generate the response, like "gpt-4o" or "o3".
	Model string `json:"model"`

	// A list of messages comprising the conversation so far.
	Messages []OpenAIChatCompletionMessageParam `json:"messages"`

	// Number between -2.0 and 2.0. Positive values penalize new tokens based on
	// their existing frequency in the text so far.
	FrequencyPenalty *float64 `json:"frequency_penalty,omitempty"`

	// Deprecated in favor of tool_choice. Controls which (if any) function is called.
	// Deprecated: use ToolChoice instead.
	FunctionCall *OpenAIChatCompletionFunctionCallOption `json:"function_call,omitempty"`

	// Deprecated in favor of tools. A list of functions the model may generate JSON inputs for.
	// Deprecated: use Tools instead.
	Functions []OpenAIChatCompletionCreateParamsFunction `json:"functions,omitempty"`

	// Modify the likelihood of specified tokens appearing in the completion.
	// Accepts a JSON object mapping token IDs to bias values from -100 to 100.
	LogitBias map[string]float64 `json:"logit_bias,omitempty"`

	// Whether to return log probabilities of the output tokens.
	Logprobs *bool `json:"logprobs,omitempty"`

	// An upper bound for the number of tokens that can be generated for a completion,
	// including visible output tokens and reasoning tokens.
	MaxCompletionTokens *int `json:"max_completion_tokens,omitempty"`

	// Deprecated: The maximum number of tokens that can be generated in the chat completion.
	// Deprecated: use MaxCompletionTokens instead.
	MaxTokens *int `json:"max_tokens,omitempty"`

	// Set of 16 key-value pairs that can be attached to an object.
	Metadata map[string]string `json:"metadata,omitempty"`

	// Output types that you would like the model to generate. Most models are
	// capable of generating text, which is the default: ["text"].
	Modalities []string `json:"modalities,omitempty"`

	// How many chat completion choices to generate for each input message.
	N *int `json:"n,omitempty"`

	// Whether to enable parallel function calling during tool use.
	ParallelToolCalls *bool `json:"parallel_tool_calls,omitempty"`

	// Static predicted output content, such as the content of a text file.
	Prediction *OpenAIChatCompletionPredictionContent `json:"prediction,omitempty"`

	// Number between -2.0 and 2.0. Positive values penalize new tokens based on
	// whether they appear in the text so far.
	PresencePenalty *float64 `json:"presence_penalty,omitempty"`

	// Used by OpenAI to cache responses for similar requests.
	PromptCacheKey string `json:"prompt_cache_key,omitempty"`

	// The retention policy for the prompt cache.
	PromptCacheRetention string `json:"prompt_cache_retention,omitempty"`

	// Constrains effort on reasoning for reasoning models.
	// Supported values: "none", "minimal", "low", "medium", "high", "xhigh".
	ReasoningEffort *string `json:"reasoning_effort,omitempty"`

	// Specifies the format that the model must output.
	ResponseFormat *OpenAIResponseFormat `json:"response_format,omitempty"`

	// If specified, our system will make a best effort to sample deterministically.
	Seed *int `json:"seed,omitempty"`

	// Specifies the latency tier to use for processing the request.
	// "auto", "default", "flex", "scale", "priority".
	ServiceTier *string `json:"service_tier,omitempty"`

	// Up to 4 sequences where the API will stop generating further tokens.
	Stop *OpenAIChatCompletionStop `json:"stop,omitempty"`

	// Whether to store the output for later retrieval via the List and Retrieve APIs.
	Store *bool `json:"store,omitempty"`

	// If set, an additional chunk will be streamed before the data: [DONE] message.
	// The usage field on this chunk shows the token usage statistics for the entire request.
	StreamOptions *OpenAIChatCompletionStreamOptions `json:"stream_options,omitempty"`

	// Whether to stream back partial progress. If set, tokens will be sent as
	// data-only server-sent events as they become available.
	Stream *bool `json:"stream,omitempty"`

	// What sampling temperature to use, between 0 and 2.
	Temperature *float64 `json:"temperature,omitempty"`

	// Controls which (if any) tool is called by the model.
	// "none", "auto", "required", or a specific tool choice.
	ToolChoice *OpenAIChatCompletionToolChoiceOption `json:"tool_choice,omitempty"`

	// A list of tools the model may call. Currently, only functions are supported as tools.
	Tools []OpenAIChatCompletionTool `json:"tools,omitempty"`

	// An integer between 0 and 20 specifying the maximum number of most likely tokens
	// to return at each token position, each with an associated log probability.
	TopLogprobs *int `json:"top_logprobs,omitempty"`

	// An alternative to sampling with temperature, called nucleus sampling.
	TopP *float64 `json:"top_p,omitempty"`

	// Deprecated: A unique identifier representing your end-user.
	// Deprecated: use PromptCacheKey or SafetyIdentifier instead.
	User string `json:"user,omitempty"`
}

// OpenAIChatCompletionMessageParam is a union of all possible message types.
type OpenAIChatCompletionMessageParam struct {
	// Role of the message author. One of: "developer", "system", "user", "assistant", "tool", "function".
	Role string `json:"role"`

	// The contents of the message. Can be a string or an array of content parts.
	// Required unless tool_calls or function_call is specified (for assistant messages).
	Content interface{} `json:"content,omitempty"`

	// An optional name for the participant.
	Name string `json:"name,omitempty"`

	// The refusal message by the assistant.
	Refusal *string `json:"refusal,omitempty"`

	// The tool calls generated by the model (only for assistant role).
	ToolCalls []OpenAIChatCompletionMessageToolCall `json:"tool_calls,omitempty"`

	// Tool call that this message is responding to (only for tool role).
	ToolCallID string `json:"tool_call_id,omitempty"`

	// Deprecated: The name and arguments of a function that should be called.
	// Deprecated: use ToolCalls instead.
	FunctionCall *OpenAIChatCompletionAssistantMessageParamFunctionCall `json:"function_call,omitempty"`

	// Data about a previous audio response from the model.
	Audio *OpenAIChatCompletionAssistantMessageParamAudio `json:"audio,omitempty"`
}

// OpenAIChatCompletionAssistantMessageParamAudio represents data about a previous audio response.
type OpenAIChatCompletionAssistantMessageParamAudio struct {
	// Unique identifier for a previous audio response from the model.
	ID string `json:"id"`
}

// OpenAIChatCompletionAssistantMessageParamFunctionCall represents a deprecated function call.
// Deprecated: replaced by ToolCalls.
type OpenAIChatCompletionAssistantMessageParamFunctionCall struct {
	// The arguments to call the function with, as generated by the model in JSON format.
	Arguments string `json:"arguments"`
	// The name of the function to call.
	Name string `json:"name"`
}

// OpenAIChatCompletionFunctionCallOption specifies a particular function to call.
type OpenAIChatCompletionFunctionCallOption struct {
	// The name of the function to call.
	Name string `json:"name"`
}

// OpenAIChatCompletionCreateParamsFunction is a deprecated function definition.
// Deprecated: use OpenAIChatCompletionTool instead.
type OpenAIChatCompletionCreateParamsFunction struct {
	// The name of the function to be called.
	Name string `json:"name"`
	// A description of what the function does.
	Description string `json:"description,omitempty"`
	// The parameters the functions accepts, described as a JSON Schema object.
	Parameters map[string]interface{} `json:"parameters,omitempty"`
	// Whether to enable strict schema adherence.
	Strict *bool `json:"strict,omitempty"`
}

// OpenAIChatCompletionPredictionContent represents static predicted output content.
type OpenAIChatCompletionPredictionContent struct {
	// The content that should be matched when generating a model response.
	Content interface{} `json:"content"`
	// The type of the predicted content. Currently always "content".
	Type string `json:"type"`
}

// OpenAIResponseFormat specifies the format that the model must output.
type OpenAIResponseFormat struct {
	// The type of response format. One of: "text", "json_object", "json_schema", "grammar", "python".
	Type string `json:"type"`

	// JSON Schema configuration (used when type is "json_schema").
	JSONSchema *OpenAIResponseFormatJSONSchema `json:"json_schema,omitempty"`

	// The custom grammar for the model to follow (used when type is "grammar").
	Grammar string `json:"grammar,omitempty"`
}

// OpenAIResponseFormatJSONSchema represents JSON Schema response format configuration.
type OpenAIResponseFormatJSONSchema struct {
	// The name of the response format.
	Name string `json:"name"`
	// A description of what the response format is for.
	Description string `json:"description,omitempty"`
	// The schema for the response format, described as a JSON Schema object.
	Schema map[string]interface{} `json:"schema,omitempty"`
	// Whether to enable strict schema adherence.
	Strict *bool `json:"strict,omitempty"`
}

// OpenAIChatCompletionStreamOptions configures streaming response behavior.
type OpenAIChatCompletionStreamOptions struct {
	// If set, an additional chunk will be streamed before data: [DONE] with usage info.
	IncludeUsage *bool `json:"include_usage,omitempty"`
	// When true, stream obfuscation will be enabled.
	IncludeObfuscation *bool `json:"include_obfuscation,omitempty"`
}

// OpenAIChatCompletionToolChoiceOption controls which tool is called by the model.
type OpenAIChatCompletionToolChoiceOption struct {
	// One of: "none", "auto", "required", or the type override.
	Type string `json:"type,omitempty"`

	// Function name (when type is "function").
	Function *OpenAIChatCompletionNamedToolChoiceFunction `json:"function,omitempty"`

	// Custom tool name (when type is "custom").
	Custom *OpenAIChatCompletionNamedToolChoiceCustom `json:"custom,omitempty"`

	// Allowed tools constraint.
	AllowedTools *OpenAIChatCompletionAllowedTools `json:"allowed_tools,omitempty"`
}

// OpenAIChatCompletionNamedToolChoiceFunction specifies a function tool to use.
type OpenAIChatCompletionNamedToolChoiceFunction struct {
	// The name of the function to call.
	Name string `json:"name"`
}

// OpenAIChatCompletionNamedToolChoiceCustom specifies a custom tool to use.
type OpenAIChatCompletionNamedToolChoiceCustom struct {
	// The name of the custom tool to call.
	Name string `json:"name"`
}

// OpenAIChatCompletionAllowedTools constrains the tools available to the model.
type OpenAIChatCompletionAllowedTools struct {
	// "auto" or "required".
	Mode string `json:"mode"`
	// A list of tool definitions that the model should be allowed to call.
	Tools []map[string]interface{} `json:"tools"`
}

// OpenAIChatCompletionTool is a tool the model may call.
// Currently supports: function tools and custom tools.
type OpenAIChatCompletionTool struct {
	// The type of the tool. "function" or "custom".
	Type string `json:"type"`

	// Function definition (when type is "function").
	Function *OpenAIFunctionDefinition `json:"function,omitempty"`

	// Custom tool definition (when type is "custom").
	Custom *OpenAIChatCompletionCustomTool `json:"custom,omitempty"`
}

// OpenAIFunctionDefinition defines a function tool.
type OpenAIFunctionDefinition struct {
	// The name of the function to be called. Must be a-z, A-Z, 0-9, underscores and dashes.
	Name string `json:"name"`
	// A description of what the function does.
	Description string `json:"description,omitempty"`
	// The parameters the functions accepts, described as a JSON Schema object.
	Parameters map[string]interface{} `json:"parameters,omitempty"`
	// Whether to enable strict schema adherence when generating the function call.
	Strict *bool `json:"strict,omitempty"`
}

// OpenAIChatCompletionCustomTool represents a custom tool with input format.
type OpenAIChatCompletionCustomTool struct {
	// The name of the custom tool.
	Name string `json:"name"`
	// Optional description of the custom tool.
	Description string `json:"description,omitempty"`
	// The input format for the custom tool. Default is unconstrained text.
	Format *OpenAICustomToolInputFormat `json:"format,omitempty"`
}

// OpenAICustomToolInputFormat specifies the input format for a custom tool.
type OpenAICustomToolInputFormat struct {
	// The type: "text" or "grammar".
	Type string `json:"type"`
	// Grammar definition (when type is "grammar").
	Definition string `json:"definition,omitempty"`
	// The syntax of the grammar definition. "lark" or "regex".
	Syntax string `json:"syntax,omitempty"`
}

// OpenAIChatCompletionStop represents stop sequences. Can be a single string or an array of strings.
type OpenAIChatCompletionStop struct {
	// Stop sequences. If single, stored as first element.
	Values []string `json:"values,omitempty"`
}

// -- Response types --

// OpenAIChatCompletion represents a chat completion response returned by model.
// Maps to SDK's ChatCompletion interface.
type OpenAIChatCompletion struct {
	// A unique identifier for the chat completion.
	ID string `json:"id"`
	// A list of chat completion choices. Can be more than one if n is greater than 1.
	Choices []OpenAIChatCompletionChoice `json:"choices"`
	// The Unix timestamp (in seconds) of when the chat completion was created.
	Created int64 `json:"created"`
	// The model used for the chat completion.
	Model string `json:"model"`
	// The object type, which is always "chat.completion".
	Object string `json:"object"`
	// The service tier used to serve the request.
	ServiceTier *string `json:"service_tier,omitempty"`
	// Deprecated: This fingerprint represents the backend configuration that the model runs with.
	// Deprecated: use Seed for determinism tracking.
	SystemFingerprint string `json:"system_fingerprint,omitempty"`
	// Usage statistics for the completion request.
	Usage *OpenAICompletionUsage `json:"usage,omitempty"`
}

// OpenAIChatCompletionChoice represents a single choice in the completion.
type OpenAIChatCompletionChoice struct {
	// The reason the model stopped generating tokens.
	// "stop", "length", "tool_calls", "content_filter", or "function_call" (deprecated).
	FinishReason string `json:"finish_reason"`
	// The index of the choice in the list of choices.
	Index int `json:"index"`
	// Log probability information for the choice.
	Logprobs *OpenAIChatCompletionChoiceLogprobs `json:"logprobs"`
	// A chat completion message generated by the model.
	Message OpenAIChatCompletionMessage `json:"message"`
}

// OpenAIChatCompletionChoiceLogprobs contains log probability information for a choice.
type OpenAIChatCompletionChoiceLogprobs struct {
	// A list of message content tokens with log probability information.
	Content []OpenAIChatCompletionTokenLogprob `json:"content"`
	// A list of message refusal tokens with log probability information.
	Refusal []OpenAIChatCompletionTokenLogprob `json:"refusal"`
}

// OpenAIChatCompletionTokenLogprob represents log probability data for a token.
type OpenAIChatCompletionTokenLogprob struct {
	// The token.
	Token string `json:"token"`
	// A list of integers representing the UTF-8 bytes representation of the token.
	// Can be null if there is no bytes representation.
	Bytes []int `json:"bytes"`
	// The log probability of this token.
	Logprob float64 `json:"logprob"`
	// List of the most likely tokens and their log probability at this token position.
	TopLogprobs []OpenAIChatCompletionTokenLogprobTopLogprob `json:"top_logprobs"`
}

// OpenAIChatCompletionTokenLogprobTopLogprob represents a top candidate token and its logprob.
type OpenAIChatCompletionTokenLogprobTopLogprob struct {
	// The token.
	Token string `json:"token"`
	// A list of integers representing the UTF-8 bytes representation of the token.
	Bytes []int `json:"bytes"`
	// The log probability of this token.
	Logprob float64 `json:"logprob"`
}

// OpenAIChatCompletionMessage represents a chat completion message generated by the model.
type OpenAIChatCompletionMessage struct {
	// The contents of the message.
	Content *string `json:"content"`
	// The reasoning content generated by the model (non-standard, e.g., DeepSeek).
	ReasoningContent *string `json:"reasoning_content,omitempty"`
	// The refusal message generated by the model.
	Refusal *string `json:"refusal"`
	// The role of the author of this message. Always "assistant" for response messages.
	Role string `json:"role"`
	// Annotations for the message, when applicable (e.g., web search tool citations).
	Annotations []OpenAIChatCompletionMessageAnnotation `json:"annotations,omitempty"`
	// If the audio output modality is requested, this contains data about the audio response.
	Audio *OpenAIChatCompletionAudio `json:"audio,omitempty"`
	// Deprecated: The name and arguments of a function that should be called.
	// Deprecated: use ToolCalls instead.
	FunctionCall *OpenAIChatCompletionMessageFunctionCall `json:"function_call,omitempty"`
	// The tool calls generated by the model, such as function calls.
	ToolCalls []OpenAIChatCompletionMessageToolCall `json:"tool_calls,omitempty"`
}

// OpenAIChatCompletionMessageAnnotation represents an annotation on a message.
type OpenAIChatCompletionMessageAnnotation struct {
	// The type of the annotation. Always "url_citation" for web search.
	Type string `json:"type"`
	// A URL citation when using web search.
	URLCitation *OpenAIChatCompletionMessageAnnotationURLCitation `json:"url_citation,omitempty"`
}

// OpenAIChatCompletionMessageAnnotationURLCitation represents a URL citation.
type OpenAIChatCompletionMessageAnnotationURLCitation struct {
	// The index of the last character of the URL citation in the message.
	EndIndex int `json:"end_index"`
	// The index of the first character of the URL citation in the message.
	StartIndex int `json:"start_index"`
	// The title of the web resource.
	Title string `json:"title"`
	// The URL of the web resource.
	URL string `json:"url"`
}

// OpenAIChatCompletionAudio contains data about the audio response from the model.
type OpenAIChatCompletionAudio struct {
	// Unique identifier for this audio response.
	ID string `json:"id"`
	// Base64 encoded audio bytes generated by the model.
	Data string `json:"data"`
	// The Unix timestamp (in seconds) for when this audio response will no longer be accessible.
	ExpiresAt int64 `json:"expires_at"`
	// Transcript of the audio generated by the model.
	Transcript string `json:"transcript"`
}

// OpenAIChatCompletionMessageFunctionCall represents a deprecated function call in a message.
// Deprecated: replaced by ToolCalls.
type OpenAIChatCompletionMessageFunctionCall struct {
	// The arguments to call the function with, as generated by the model in JSON format.
	Arguments string `json:"arguments"`
	// The name of the function to call.
	Name string `json:"name"`
}

// OpenAIChatCompletionMessageToolCall represents a tool call generated by the model.
// Can be a function tool call or a custom tool call.
type OpenAIChatCompletionMessageToolCall struct {
	// The ID of the tool call.
	ID string `json:"id"`
	// The type of the tool. "function" or "custom".
	Type string `json:"type"`
	// The function that the model called (when type is "function").
	Function *OpenAIChatCompletionMessageFunctionToolCallFunction `json:"function,omitempty"`
	// The custom tool that the model called (when type is "custom").
	Custom *OpenAIChatCompletionMessageCustomToolCallCustom `json:"custom,omitempty"`
}

// OpenAIChatCompletionMessageFunctionToolCallFunction represents the function called by the model.
type OpenAIChatCompletionMessageFunctionToolCallFunction struct {
	// The arguments to call the function with, as generated by the model in JSON format.
	Arguments string `json:"arguments"`
	// The name of the function to call.
	Name string `json:"name"`
}

// OpenAIChatCompletionMessageCustomToolCallCustom represents a custom tool called by the model.
type OpenAIChatCompletionMessageCustomToolCallCustom struct {
	// The input for the custom tool call generated by the model.
	Input string `json:"input"`
	// The name of the custom tool to call.
	Name string `json:"name"`
}

// OpenAICompletionUsage represents usage statistics for a completion request.
// Shared between Chat Completions and Legacy Completions.
type OpenAICompletionUsage struct {
	// Number of tokens in the generated completion.
	CompletionTokens int `json:"completion_tokens"`
	// Number of tokens in the prompt.
	PromptTokens int `json:"prompt_tokens"`
	// Total number of tokens used in the request (prompt + completion).
	TotalTokens int `json:"total_tokens"`
	// Breakdown of tokens used in a completion.
	CompletionTokensDetails *OpenAICompletionTokensDetails `json:"completion_tokens_details,omitempty"`
	// Breakdown of tokens used in the prompt.
	PromptTokensDetails *OpenAIPromptTokensDetails `json:"prompt_tokens_details,omitempty"`
}

// OpenAICompletionTokensDetails provides a breakdown of completion tokens.
type OpenAICompletionTokensDetails struct {
	// When using Predicted Outputs, tokens in the prediction that appeared in the completion.
	AcceptedPredictionTokens int `json:"accepted_prediction_tokens,omitempty"`
	// Audio input tokens generated by the model.
	AudioTokens int `json:"audio_tokens,omitempty"`
	// Tokens generated by the model for reasoning.
	ReasoningTokens int `json:"reasoning_tokens,omitempty"`
	// When using Predicted Outputs, tokens in the prediction that did not appear.
	RejectedPredictionTokens int `json:"rejected_prediction_tokens,omitempty"`
}

// OpenAIPromptTokensDetails provides a breakdown of prompt tokens.
type OpenAIPromptTokensDetails struct {
	// Audio input tokens present in the prompt.
	AudioTokens int `json:"audio_tokens,omitempty"`
	// Cached tokens present in the prompt.
	CachedTokens int `json:"cached_tokens,omitempty"`
}

// -- Stream (chunk) types --

// OpenAIChatCompletionChunk represents a streamed chunk of a chat completion response.
// Maps to SDK's ChatCompletionChunk interface.
type OpenAIChatCompletionChunk struct {
	// A unique identifier for the chat completion. Each chunk has the same ID.
	ID string `json:"id"`
	// A list of chat completion choices. Can contain more than one element if n > 1.
	Choices []OpenAIChatCompletionChunkChoice `json:"choices"`
	// The Unix timestamp (in seconds) of when the chat completion was created.
	Created int64 `json:"created"`
	// The model to generate the completion.
	Model string `json:"model"`
	// The object type, which is always "chat.completion.chunk".
	Object string `json:"object"`
	// The service tier used to serve the request.
	ServiceTier *string `json:"service_tier,omitempty"`
	// Deprecated: This fingerprint represents the backend configuration that the model runs with.
	// Deprecated: use Seed for determinism tracking.
	SystemFingerprint string `json:"system_fingerprint,omitempty"`
	// An optional field that will only be present when you set stream_options: {"include_usage": true}.
	// Contains null except for the last chunk which has the total token usage.
	Usage *OpenAICompletionUsage `json:"usage,omitempty"`
}

// OpenAIChatCompletionChunkChoice represents a single choice in a streamed chunk.
type OpenAIChatCompletionChunkChoice struct {
	// A chat completion delta generated by streamed model responses.
	Delta OpenAIChatCompletionChunkChoiceDelta `json:"delta"`
	// The reason the model stopped generating tokens (null for intermediate chunks).
	FinishReason *string `json:"finish_reason"`
	// The index of the choice in the list of choices.
	Index int `json:"index"`
	// Log probability information for the choice.
	Logprobs *OpenAIChatCompletionChoiceLogprobs `json:"logprobs,omitempty"`
}

// OpenAIChatCompletionChunkChoiceDelta represents a delta in a streamed chunk.
type OpenAIChatCompletionChunkChoiceDelta struct {
	// The contents of the chunk message.
	Content *string `json:"content,omitempty"`
	// Deprecated: The name and arguments of a function that should be called.
	// Deprecated: use ToolCalls instead.
	FunctionCall *OpenAIChatCompletionChunkChoiceDeltaFunctionCall `json:"function_call,omitempty"`
	// The reasoning content generated by the model (non-standard, e.g., DeepSeek).
	ReasoningContent *string `json:"reasoning_content,omitempty"`
	// The refusal message generated by the model.
	Refusal *string `json:"refusal,omitempty"`
	// The role of the author of this message.
	Role string `json:"role,omitempty"`
	// The tool calls generated by the model.
	ToolCalls []OpenAIChatCompletionChunkChoiceDeltaToolCall `json:"tool_calls,omitempty"`
}

// OpenAIChatCompletionChunkChoiceDeltaFunctionCall represents a deprecated function call delta.
// Deprecated: replaced by ToolCalls.
type OpenAIChatCompletionChunkChoiceDeltaFunctionCall struct {
	// The arguments to call the function with, as generated by the model in JSON format.
	Arguments string `json:"arguments,omitempty"`
	// The name of the function to call.
	Name string `json:"name,omitempty"`
}

// OpenAIChatCompletionChunkChoiceDeltaToolCall represents a tool call delta in a streamed chunk.
type OpenAIChatCompletionChunkChoiceDeltaToolCall struct {
	// The index of the tool call in the list of tool calls (used for ordering in stream).
	Index int `json:"index"`
	// The ID of the tool call.
	ID string `json:"id,omitempty"`
	// The function being called.
	Function *OpenAIChatCompletionChunkChoiceDeltaToolCallFunction `json:"function,omitempty"`
	// The type of the tool. Currently, only "function" is supported.
	Type string `json:"type,omitempty"`
}

// OpenAIChatCompletionChunkChoiceDeltaToolCallFunction represents function details in a tool call delta.
type OpenAIChatCompletionChunkChoiceDeltaToolCallFunction struct {
	// The arguments to call the function with, as generated by the model in JSON format.
	Arguments string `json:"arguments,omitempty"`
	// The name of the function to call.
	Name string `json:"name,omitempty"`
}

// -- Shared constants --

// OpenAI Chat Completion object type constants.
const (
	OpenAIChatCompletionObject      = "chat.completion"
	OpenAIChatCompletionChunkObject = "chat.completion.chunk"
)

// OpenAI Chat Completion role constants.
const (
	OpenAIRoleDeveloper = "developer"
	OpenAIRoleSystem    = "system"
	OpenAIRoleUser      = "user"
	OpenAIRoleAssistant = "assistant"
	OpenAIRoleTool      = "tool"
	OpenAIRoleFunction  = "function"
)

// OpenAI Chat Completion finish reason constants.
const (
	OpenAIFinishReasonStop          = "stop"
	OpenAIFinishReasonLength        = "length"
	OpenAIFinishReasonToolCalls     = "tool_calls"
	OpenAIFinishReasonContentFilter = "content_filter"
	// Deprecated: use ToolCalls instead.
	OpenAIFinishReasonFunctionCall = "function_call"
)

// OpenAI service tier constants.
const (
	OpenAIServiceTierAuto     = "auto"
	OpenAIServiceTierDefault  = "default"
	OpenAIServiceTierFlex     = "flex"
	OpenAIServiceTierScale    = "scale"
	OpenAIServiceTierPriority = "priority"
)

// OpenAI tool choice constants.
const (
	OpenAIToolChoiceNone     = "none"
	OpenAIToolChoiceAuto     = "auto"
	OpenAIToolChoiceRequired = "required"
)

// OpenAI response format type constants.
const (
	OpenAIResponseFormatText         = "text"
	OpenAIResponseFormatJSONObject   = "json_object"
	OpenAIResponseFormatJSONSchemaType = "json_schema"
	OpenAIResponseFormatGrammar      = "grammar"
	OpenAIResponseFormatPython       = "python"
)

// OpenAI reasoning effort constants.
const (
	OpenAIReasoningEffortNone    = "none"
	OpenAIReasoningEffortMinimal = "minimal"
	OpenAIReasoningEffortLow     = "low"
	OpenAIReasoningEffortMedium  = "medium"
	OpenAIReasoningEffortHigh    = "high"
	OpenAIReasoningEffortXHigh   = "xhigh"
)

// -- Backward-compatible aliases for legacy type names. --

// Deprecated: use OpenAIChatCompletionRequest.
type OpenAIChatRequest = OpenAIChatCompletionRequest

// Deprecated: use OpenAIChatCompletionMessageParam.
type OpenAIChatMessage = OpenAIChatCompletionMessageParam

// Deprecated: use OpenAIChatCompletion.
type OpenAIChatResponse = OpenAIChatCompletion

// Deprecated: use OpenAIChatCompletionChunk.
type OpenAIChatStreamChunk = OpenAIChatCompletionChunk

// Deprecated: use OpenAIChatCompletionChoice.
type OpenAIChatChoice = OpenAIChatCompletionChoice

// Deprecated: use OpenAIChatCompletionChunkChoiceDelta.
type OpenAIChatDelta = OpenAIChatCompletionChunkChoiceDelta

// Deprecated: use OpenAIChatCompletionMessageToolCall.
type OpenAIChatToolCall = OpenAIChatCompletionMessageToolCall

// Deprecated: use OpenAIChatCompletionMessageFunctionCall.
type OpenAIChatFunctionCall = OpenAIChatCompletionMessageFunctionCall

// Deprecated: use OpenAIChatCompletionChunkObject.
const OpenAIChatStreamChunkObject = OpenAIChatCompletionChunkObject

// Deprecated: use OpenAIChatCompletionChunkChoice.
type OpenAIChatStreamChoice = OpenAIChatCompletionChunkChoice

// Deprecated: use OpenAIChatCompletionObject.
const OpenAIChatObject = OpenAIChatCompletionObject

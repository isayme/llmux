package convert

type OpenAIResponsesRequest struct {
	Model           string      `json:"model"`
	Input           interface{} `json:"input"`
	Instructions    string      `json:"instructions,omitempty"`
	MaxOutputTokens *int        `json:"max_output_tokens,omitempty"`
}

type ResponsesInputItem struct {
	Type    string      `json:"type"`
	Role    string      `json:"role,omitempty"`
	Content interface{} `json:"content,omitempty"`
}

type OpenAIResponsesResponse struct {
	ID        string                 `json:"id"`
	Object    string                 `json:"object"`
	CreatedAt int64                  `json:"created_at"`
	Model     string                 `json:"model"`
	Output    []ResponsesOutputItem  `json:"output"`
	Usage     *ResponsesUsage        `json:"usage,omitempty"`
}

type ResponsesOutputItem struct {
	Type    string                   `json:"type"`
	ID      string                   `json:"id,omitempty"`
	Role    string                   `json:"role,omitempty"`
	Content []ResponsesContentBlock  `json:"content,omitempty"`
}

type ResponsesContentBlock struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

type Usage struct {
	PromptTokens     int `json:"prompt_tokens,omitempty"`
	CompletionTokens int `json:"completion_tokens,omitempty"`
	TotalTokens      int `json:"total_tokens,omitempty"`
}

type OpenAIResponsesStreamEvent struct {
	Type     string                        `json:"type"`
	Delta    string                        `json:"delta,omitempty"`
	Text     string                        `json:"text,omitempty"`
	Item     *ResponsesOutputItem          `json:"item,omitempty"`
	Part     *ResponsesContentBlock        `json:"part,omitempty"`
	Response *OpenAIResponsesStreamSummary `json:"response,omitempty"`
}

type ResponsesUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
	TotalTokens  int `json:"total_tokens"`
}

type OpenAIResponsesStreamSummary struct {
	ID        string           `json:"id,omitempty"`
	Model     string           `json:"model,omitempty"`
	CreatedAt int64            `json:"created_at,omitempty"`
	Usage     *Usage           `json:"usage,omitempty"`
}

// OpenAI Responses API object / item / content type constants.
const (
	ResponsesObject          = "response"
	ResponsesItemTypeMessage = "message"
	ResponsesContentTypeText = "output_text"
)

// Responses API SSE event name constants, used in the "event:" line and event data "type" field.
const (
	ResponsesSSECreated            = "response.created"
	ResponsesSSEOutputItemAdded    = "response.output_item.added"
	ResponsesSSEOutputItemDone     = "response.output_item.done"
	ResponsesSSEContentPartAdded   = "response.content_part.added"
	ResponsesSSETextDelta          = "response.text.delta"
	ResponsesSSETextDone           = "response.text.done"
	ResponsesSSEDone               = "response.done"
	ResponsesSSEError              = "error"
)

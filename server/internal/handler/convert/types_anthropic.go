package convert

type AnthropicRequest struct {
	Model         string              `json:"model"`
	Messages      []AnthropicMessage  `json:"messages"`
	System        string              `json:"system,omitempty"`
	MaxTokens     int                 `json:"max_tokens"`
	StopSequences []string            `json:"stop_sequences,omitempty"`
	Stream        *bool               `json:"stream,omitempty"`
	Temperature   *float64            `json:"temperature,omitempty"`
	TopP          *float64            `json:"top_p,omitempty"`
}

type AnthropicMessage struct {
	Role    string      `json:"role"`
	Content interface{} `json:"content"`
}

type AnthropicResponse struct {
	ID         string                  `json:"id"`
	Type       string                  `json:"type"`
	Role       string                  `json:"role"`
	Model      string                  `json:"model"`
	Content    []AnthropicContentBlock `json:"content"`
	StopReason string                  `json:"stop_reason"`
	Usage      *AnthropicUsage         `json:"usage,omitempty"`
}

type AnthropicContentBlock struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

type AnthropicUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

type AnthropicSSEEvent struct {
	Type         string                `json:"type"`
	Index        *int                  `json:"index,omitempty"`
	Delta        *AnthropicSSEDelta    `json:"delta,omitempty"`
	Message      *AnthropicResponse    `json:"message,omitempty"`
	ContentBlock *AnthropicContentBlock `json:"content_block,omitempty"`
	Usage        *AnthropicUsage       `json:"usage,omitempty"`
}

type AnthropicSSEDelta struct {
	Type       string `json:"type,omitempty"`
	Text       string `json:"text,omitempty"`
	StopReason string `json:"stop_reason,omitempty"`
}

// Anthropic SSE event name constants, used in the "event:" line and inside event data "type" field.
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
)

// Anthropic content / delta type constants, used in ContentBlock.Type and SSEDelta.Type.
const (
	AnthropicContentTypeText    = "text"
	AnthropicDeltaTypeTextDelta = "text_delta"
)

// Anthropic object / role constants, used in Response.Type and Message.Role.
const (
	AnthropicObjectMessage = "message"
	AnthropicRoleUser      = "user"
	AnthropicRoleAssistant = "assistant"
)

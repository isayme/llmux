package convert

type OpenAIChatRequest struct {
	Model       string              `json:"model"`
	Messages    []OpenAIChatMessage `json:"messages"`
	MaxTokens   *int                `json:"max_tokens,omitempty"`
	Temperature *float64            `json:"temperature,omitempty"`
	TopP        *float64            `json:"top_p,omitempty"`
	Stream      *bool               `json:"stream,omitempty"`
	Stop        []string            `json:"stop,omitempty"`
}

type OpenAIChatMessage struct {
	Role    string      `json:"role"`
	Content interface{} `json:"content"`
}

type OpenAIChatResponse struct {
	ID      string            `json:"id"`
	Object  string            `json:"object"`
	Created int64             `json:"created"`
	Model   string            `json:"model"`
	Choices []OpenAIChatChoice `json:"choices"`
	Usage   *Usage            `json:"usage,omitempty"`
}

type OpenAIChatChoice struct {
	Index        int              `json:"index"`
	Message      OpenAIChatMessage `json:"message"`
	FinishReason string           `json:"finish_reason"`
}

type OpenAIChatStreamChunk struct {
	ID      string                    `json:"id,omitempty"`
	Object  string                    `json:"object"`
	Created int64                     `json:"created,omitempty"`
	Model   string                    `json:"model,omitempty"`
	Choices []OpenAIChatStreamChoice  `json:"choices"`
}

type OpenAIChatStreamChoice struct {
	Index        int               `json:"index"`
	Delta        OpenAIChatDelta   `json:"delta"`
	FinishReason string            `json:"finish_reason,omitempty"`
}

type OpenAIChatDelta struct {
	Role    string `json:"role,omitempty"`
	Content string `json:"content,omitempty"`
}

// OpenAI Chat API object type constants, used in the "object" field of non-streaming/streaming responses.
const (
	OpenAIChatObject           = "chat.completion"
	OpenAIChatStreamChunkObject = "chat.completion.chunk"
)

// OpenAI Chat message role constants for Message.Role and Delta.Role.
const (
	OpenAIRoleSystem    = "system"
	OpenAIRoleUser      = "user"
	OpenAIRoleAssistant = "assistant"
)

// OpenAI Chat finish reason constants for Choice.FinishReason.
const (
	OpenAIFinishReasonStop   = "stop"
	OpenAIFinishReasonLength = "length"
)

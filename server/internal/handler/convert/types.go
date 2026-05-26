package convert

import (
	"io"
)

type UsedAIProtocol int

// Protocol identifiers, used to select converter via GetConverter().
const (
	ProtocolOpenAI UsedAIProtocol = iota
	ProtocolAnthropic
	ProtocolOpenAIResponses
)

func (p UsedAIProtocol) String() string {
	switch p {
	case ProtocolOpenAI:
		return "openai"
	case ProtocolAnthropic:
		return "anthropic"
	case ProtocolOpenAIResponses:
		return "openai_responses"
	default:
		return "unknown"
	}
}

type SSEEvent struct {
	Event string
	Data  string
}

type ProtocolConverter interface {
	ConvertRequest(req any) (any, error)
	ConvertResponse(body []byte) ([]byte, error)
	ConvertSSE(body io.ReadCloser) io.ReadCloser
}

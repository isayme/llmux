package convert

import (
	"io"

	"llmux/internal/constant"
)

type UsedAIProtocol int

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

// ProtocolConverter transforms requests and responses between AI protocols.
// Each conversion direction (e.g. OpenAI ↔ Anthropic) is a separate
// implementation; adding a new protocol only requires a new struct.
type ProtocolConverter interface {
	ConvertRequest(req map[string]interface{}) map[string]interface{}
	ConvertResponse(body []byte) ([]byte, error)
	ConvertSSE(body io.ReadCloser) io.ReadCloser
}

type converterKey struct {
	src UsedAIProtocol
	dst string
}

var converters = map[converterKey]func() ProtocolConverter{
	{ProtocolOpenAI, constant.ProviderTypeAnthropic}:                func() ProtocolConverter { return &openaiToAnthropicConverter{} },
	{ProtocolAnthropic, constant.ProviderTypeOpenAI}:                func() ProtocolConverter { return &anthropicToOpenAIConverter{} },
	{ProtocolOpenAIResponses, constant.ProviderTypeOpenAI}:          func() ProtocolConverter { return &responsesToOpenAIConverter{} },
	{ProtocolOpenAI, constant.ProviderTypeOpenAIResponses}:          func() ProtocolConverter { return &openaiToResponsesConverter{} },
	{ProtocolOpenAIResponses, constant.ProviderTypeAnthropic}:       func() ProtocolConverter { return &responsesToAnthropicConverter{} },
	{ProtocolAnthropic, constant.ProviderTypeOpenAIResponses}:       func() ProtocolConverter { return &anthropicToResponsesConverter{} },
}

// GetConverter returns the ProtocolConverter for the given client protocol and
// provider type. Returns a no-op passthrough converter when protocols match.
func GetConverter(usedProtocol UsedAIProtocol, providerType string) ProtocolConverter {
	if fn, ok := converters[converterKey{usedProtocol, providerType}]; ok {
		return fn()
	}
	return &noopConverter{}
}

// noopConverter passes data through unchanged when no protocol conversion
// is needed (client and provider use the same protocol).
type noopConverter struct{}

func (c *noopConverter) ConvertRequest(req map[string]interface{}) map[string]interface{} { return req }
func (c *noopConverter) ConvertResponse(body []byte) ([]byte, error)                      { return body, nil }
func (c *noopConverter) ConvertSSE(r io.ReadCloser) io.ReadCloser                          { return r }

// copyMap creates a shallow copy of src, omitting the given keys.
func copyMap(src map[string]interface{}, omit ...string) map[string]interface{} {
	out := make(map[string]interface{})
	skip := make(map[string]bool, len(omit))
	for _, k := range omit {
		skip[k] = true
	}
	for k, v := range src {
		if skip[k] {
			continue
		}
		out[k] = v
	}
	return out
}

// extractString walks a nested map to extract a string value at the given key
// path. Used when reading SSE event data.
func extractString(m map[string]interface{}, keys ...string) string {
	for i, key := range keys {
		v, ok := m[key]
		if !ok {
			return ""
		}
		if i == len(keys)-1 {
			s, _ := v.(string)
			return s
		}
		m, ok = v.(map[string]interface{})
		if !ok {
			return ""
		}
	}
	return ""
}

func toFloat64(v interface{}) float64 {
	switch n := v.(type) {
	case float64:
		return n
	case int:
		return float64(n)
	case int64:
		return float64(n)
	}
	return 0
}

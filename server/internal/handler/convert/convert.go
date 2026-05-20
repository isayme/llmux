package convert

import (
	"io"

	"llmux/internal/constant"
)

type UsedAIProtocol int

const (
	ProtocolOpenAI UsedAIProtocol = iota
	ProtocolAnthropic
)

// ProtocolConverter transforms requests and responses between AI protocols.
// Each conversion direction (e.g. OpenAI ↔ Anthropic) is a separate
// implementation; adding a new protocol only requires a new struct.
type ProtocolConverter interface {
	ConvertRequest(req map[string]interface{}) map[string]interface{}
	ConvertResponse(body []byte) ([]byte, error)
	ConvertSSE(body io.ReadCloser) io.ReadCloser
}

// GetConverter returns the ProtocolConverter for the given client protocol and
// provider type. Returns a no-op passthrough converter when protocols match.
func GetConverter(usedProtocol UsedAIProtocol, providerType string) ProtocolConverter {
	if usedProtocol == ProtocolOpenAI && providerType == constant.ProviderTypeAnthropic {
		return &openaiToAnthropicConverter{}
	}
	if usedProtocol == ProtocolAnthropic && providerType == constant.ProviderTypeOpenAI {
		return &anthropicToOpenAIConverter{}
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

package convert

import (
	"io"

	"llmux/internal/constant"
)

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

func GetConverter(usedProtocol UsedAIProtocol, providerType string) ProtocolConverter {
	if fn, ok := converters[converterKey{usedProtocol, providerType}]; ok {
		return fn()
	}
	return &noopConverter{}
}

type noopConverter struct{}

func (c *noopConverter) ConvertRequest(req any) (any, error) { return req, nil }
func (c *noopConverter) ConvertResponse(body []byte) ([]byte, error) { return body, nil }
func (c *noopConverter) ConvertSSE(r io.ReadCloser) io.ReadCloser { return r }

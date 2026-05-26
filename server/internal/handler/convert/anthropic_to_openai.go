package convert

import (
	"encoding/json"
	"errors"
	"io"
)

type anthropicToOpenAIConverter struct{}

func (c *anthropicToOpenAIConverter) ConvertRequest(req any) (any, error) {
	anthReq, ok := req.(*AnthropicRequest)
	if !ok {
		return nil, errors.New("expected *AnthropicRequest")
	}

	var messages []OpenAIChatMessage

	if anthReq.System != "" {
		messages = append(messages, OpenAIChatMessage{
			Role:    OpenAIRoleSystem,
			Content: anthReq.System,
		})
	}

	for _, m := range anthReq.Messages {
		messages = append(messages, OpenAIChatMessage{
			Role:    m.Role,
			Content: m.Content,
		})
	}

	oaiReq := &OpenAIChatRequest{
		Model:     anthReq.Model,
		Messages:  messages,
		MaxTokens: intPtr(anthReq.MaxTokens),
		Stream:    anthReq.Stream,
	}

	if anthReq.Temperature != nil {
		oaiReq.Temperature = anthReq.Temperature
	}
	if anthReq.TopP != nil {
		oaiReq.TopP = anthReq.TopP
	}
	if len(anthReq.StopSequences) > 0 {
		oaiReq.Stop = anthReq.StopSequences
	}

	return oaiReq, nil
}

func intPtr(v int) *int {
	return &v
}

func (c *anthropicToOpenAIConverter) ConvertResponse(body []byte) ([]byte, error) {
	var oaiResp OpenAIChatResponse
	if err := json.Unmarshal(body, &oaiResp); err != nil {
		return nil, err
	}

	anthResp := AnthropicResponse{
		ID:    oaiResp.ID,
		Type:  AnthropicObjectMessage,
		Role:  AnthropicRoleAssistant,
		Model: oaiResp.Model,
		Content: []AnthropicContentBlock{
			{
				Type: AnthropicContentTypeText,
				Text: extractChatContent(oaiResp.Choices),
			},
		},
		StopReason: mapOpenAIFinishReason(extractFinishReason(oaiResp.Choices)),
	}

	if oaiResp.Usage != nil {
		anthResp.Usage = &AnthropicUsage{
			InputTokens:  oaiResp.Usage.PromptTokens,
			OutputTokens: oaiResp.Usage.CompletionTokens,
		}
	}

	return json.Marshal(anthResp)
}

func extractChatContent(choices []OpenAIChatChoice) string {
	if len(choices) == 0 {
		return ""
	}
	content, _ := choices[0].Message.Content.(string)
	return content
}

func extractFinishReason(choices []OpenAIChatChoice) string {
	if len(choices) == 0 {
		return ""
	}
	return choices[0].FinishReason
}

func mapOpenAIFinishReason(finishReason string) string {
	switch finishReason {
	case OpenAIFinishReasonStop:
		return AnthropicStopReasonEndTurn
	case OpenAIFinishReasonLength:
		return AnthropicStopReasonMaxTokens
	default:
		return AnthropicStopReasonEndTurn
	}
}

func (c *anthropicToOpenAIConverter) ConvertSSE(r io.ReadCloser) io.ReadCloser {
	pr, pw := io.Pipe()

	go func() {
		defer pw.Close()

		events, errs := ParseSSE(r)

		var messageID string
		var model string
		var started bool
		var outputTokens int

		for {
			select {
			case event, ok := <-events:
				if !ok {
					return
				}

				if event.Data == "" {
					continue
				}

				if event.Data == SSEDoneMarker {
					if !started {
						return
					}
					writeSSEJSON(pw, AnthropicSSEMessageStopEvent, AnthropicSSEEvent{
						Type: AnthropicSSEMessageStopEvent,
					})
					return
				}

				var chunk OpenAIChatStreamChunk
				if err := json.Unmarshal([]byte(event.Data), &chunk); err != nil {
					continue
				}

				if len(chunk.Choices) == 0 {
					continue
				}

				if !started {
					started = true
					messageID = chunk.ID
					if messageID == "" {
						messageID = "msg_unknown"
					}
					model = chunk.Model

					writeSSEJSON(pw, AnthropicSSEMessageStartEvent, AnthropicSSEEvent{
						Type: AnthropicSSEMessageStartEvent,
						Message: &AnthropicResponse{
							ID:    messageID,
							Type:  AnthropicObjectMessage,
							Role:  AnthropicRoleAssistant,
							Model: model,
							Usage: &AnthropicUsage{},
						},
					})

					writeSSEJSON(pw, AnthropicSSEContentBlockStartEvent, AnthropicSSEEvent{
						Type:  AnthropicSSEContentBlockStartEvent,
						Index: intPtr(0),
						ContentBlock: &AnthropicContentBlock{
							Type: AnthropicContentTypeText,
							Text: "",
						},
					})
				}

				delta := chunk.Choices[0].Delta
				if delta.Content != "" {
					writeSSEJSON(pw, AnthropicSSEContentBlockDeltaEvent, AnthropicSSEEvent{
						Type:  AnthropicSSEContentBlockDeltaEvent,
						Index: intPtr(0),
						Delta: &AnthropicSSEDelta{
							Type: AnthropicDeltaTypeTextDelta,
							Text: delta.Content,
						},
					})
					outputTokens++
				}

				finishReason := chunk.Choices[0].FinishReason
				if finishReason != "" {
					writeSSEJSON(pw, AnthropicSSEContentBlockStopEvent, AnthropicSSEEvent{
						Type:  AnthropicSSEContentBlockStopEvent,
						Index: intPtr(0),
					})

					writeSSEJSON(pw, AnthropicSSEMessageDeltaEvent, AnthropicSSEEvent{
						Type: AnthropicSSEMessageDeltaEvent,
						Delta: &AnthropicSSEDelta{
							StopReason: mapOpenAIFinishReason(finishReason),
						},
						Usage: &AnthropicUsage{
							OutputTokens: outputTokens,
						},
					})

					writeSSEJSON(pw, AnthropicSSEMessageStopEvent, AnthropicSSEEvent{
						Type: AnthropicSSEMessageStopEvent,
					})
				}

			case err, ok := <-errs:
				if ok && err != nil {
					pw.CloseWithError(err)
					return
				}
				return
			}
		}
	}()

	return pr
}

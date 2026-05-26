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
		oaiReq.Stop = &OpenAIChatCompletionStop{Values: anthReq.StopSequences}
	}

	if anthReq.Thinking != nil {
		switch anthReq.Thinking.Type {
		case AnthropicThinkingDisabled:
			oaiReq.ReasoningEffort = stringPtr(OpenAIReasoningEffortNone)
		default:
			oaiReq.ReasoningEffort = stringPtr(OpenAIReasoningEffortHigh)
		}
	}
	if anthReq.OutputConfig != nil && anthReq.OutputConfig.Effort != "" {
		oaiReq.ReasoningEffort = stringPtr(mapAnthropicEffortToReasoning(anthReq.OutputConfig.Effort))
	}

	// Unmapped Anthropic fields that have OpenAI equivalents (not yet implemented):
	//   Tools/ToolChoice → OpenAI Tools/ToolChoice
	//   Metadata → OpenAI Metadata
	//   OutputConfig → OpenAI ResponseFormat (only Format part)
	// Unmapped — no OpenAI equivalent:
	//   TopK

	return oaiReq, nil
}

func intPtr(v int) *int {
	return &v
}

func mapAnthropicEffortToReasoning(effort string) string {
	switch effort {
	case "low":
		return OpenAIReasoningEffortLow
	case "medium":
		return OpenAIReasoningEffortMedium
	case "high":
		return OpenAIReasoningEffortHigh
	case "xhigh", "max":
		return OpenAIReasoningEffortXHigh
	default:
		return OpenAIReasoningEffortHigh
	}
}

func (c *anthropicToOpenAIConverter) ConvertResponse(body []byte) ([]byte, error) {
	var oaiResp OpenAIChatResponse
	if err := json.Unmarshal(body, &oaiResp); err != nil {
		return nil, err
	}

	content := buildAnthropicContent(oaiResp.Choices)
	anthResp := AnthropicResponse{
		ID:    oaiResp.ID,
		Type:  AnthropicObjectMessage,
		Role:  AnthropicRoleAssistant,
		Model: oaiResp.Model,
		Content:    content,
		StopReason: stringPtr(mapOpenAIFinishReason(extractFinishReason(oaiResp.Choices))),
	}

	if oaiResp.Usage != nil {
		anthResp.Usage = &AnthropicUsage{
			InputTokens:  oaiResp.Usage.PromptTokens,
			OutputTokens: oaiResp.Usage.CompletionTokens,
		}
	}

	// Unmapped OpenAI Chat response fields:
	//   Choices[].Message.ToolCalls → not converted to Anthropic tool_use blocks
	//   Choices[].Message.Refusal → not converted
	//   Choices[].Message.Audio → no Anthropic equivalent
	//   Choices[].Message.FunctionCall → deprecated
	//   ServiceTier, SystemFingerprint → no Anthropic equivalent

	return json.Marshal(anthResp)
}

func stringPtr(s string) *string {
	return &s
}

func buildAnthropicContent(choices []OpenAIChatChoice) []AnthropicContentBlock {
	if len(choices) == 0 {
		return nil
	}
	msg := choices[0].Message
	var blocks []AnthropicContentBlock
	if msg.ReasoningContent != nil && *msg.ReasoningContent != "" {
		blocks = append(blocks, AnthropicContentBlock{
			Type:     AnthropicContentTypeThinking,
			Thinking: *msg.ReasoningContent,
		})
	}
	if msg.Content != nil {
		blocks = append(blocks, AnthropicContentBlock{
			Type: AnthropicContentTypeText,
			Text: *msg.Content,
		})
	}
	return blocks
}

func extractChatContent(choices []OpenAIChatChoice) string {
	if len(choices) == 0 || choices[0].Message.Content == nil {
		return ""
	}
	return *choices[0].Message.Content
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
		var inThinking bool
		var textStarted bool
		var blockIndex int
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
				}

				delta := chunk.Choices[0].Delta

				if delta.ReasoningContent != nil && *delta.ReasoningContent != "" {
					if !inThinking {
						writeSSEJSON(pw, AnthropicSSEContentBlockStartEvent, AnthropicSSEEvent{
							Type:  AnthropicSSEContentBlockStartEvent,
							Index: intPtr(blockIndex),
							ContentBlock: &AnthropicContentBlock{
								Type: AnthropicContentTypeThinking,
							},
						})
						inThinking = true
					}

					writeSSEJSON(pw, AnthropicSSEContentBlockDeltaEvent, AnthropicSSEEvent{
						Type:  AnthropicSSEContentBlockDeltaEvent,
						Index: intPtr(blockIndex),
						Delta: &AnthropicSSEDelta{
							Type:     AnthropicDeltaTypeThinkingDelta,
							Thinking: *delta.ReasoningContent,
						},
					})
				}

				if delta.Content != nil && *delta.Content != "" {
					if inThinking {
						writeSSEJSON(pw, AnthropicSSEContentBlockStopEvent, AnthropicSSEEvent{
							Type:  AnthropicSSEContentBlockStopEvent,
							Index: intPtr(blockIndex),
						})
						blockIndex++
						inThinking = false
					}
					if !textStarted {
						writeSSEJSON(pw, AnthropicSSEContentBlockStartEvent, AnthropicSSEEvent{
							Type:  AnthropicSSEContentBlockStartEvent,
							Index: intPtr(blockIndex),
							ContentBlock: &AnthropicContentBlock{
								Type: AnthropicContentTypeText,
								Text: "",
							},
						})
						textStarted = true
					}

					writeSSEJSON(pw, AnthropicSSEContentBlockDeltaEvent, AnthropicSSEEvent{
						Type:  AnthropicSSEContentBlockDeltaEvent,
						Index: intPtr(blockIndex),
						Delta: &AnthropicSSEDelta{
							Type: AnthropicDeltaTypeTextDelta,
							Text: *delta.Content,
						},
					})
					outputTokens++
				}

				finishReason := chunk.Choices[0].FinishReason
				if finishReason != nil && *finishReason != "" {
					if inThinking {
						writeSSEJSON(pw, AnthropicSSEContentBlockStopEvent, AnthropicSSEEvent{
							Type:  AnthropicSSEContentBlockStopEvent,
							Index: intPtr(blockIndex),
						})
						blockIndex++
						inThinking = false
					}
					if textStarted {
						writeSSEJSON(pw, AnthropicSSEContentBlockStopEvent, AnthropicSSEEvent{
							Type:  AnthropicSSEContentBlockStopEvent,
							Index: intPtr(blockIndex),
						})
					}

					writeSSEJSON(pw, AnthropicSSEMessageDeltaEvent, AnthropicSSEEvent{
						Type: AnthropicSSEMessageDeltaEvent,
						Delta: &AnthropicSSEDelta{
							StopReason: stringPtr(mapOpenAIFinishReason(*finishReason)),
						},
						Usage: &AnthropicMessageDeltaUsage{
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

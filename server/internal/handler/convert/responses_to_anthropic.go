package convert

import (
	"encoding/json"
	"errors"
	"io"
)

type responsesToAnthropicConverter struct{}

func (c *responsesToAnthropicConverter) ConvertRequest(req any) (any, error) {
	respReq, ok := req.(*OpenAIResponsesRequest)
	if !ok {
		return nil, errors.New("expected *OpenAIResponsesRequest")
	}

	anthReq := &AnthropicRequest{
		Model: respReq.Model,
	}

	if respReq.MaxOutputTokens != nil {
		anthReq.MaxTokens = *respReq.MaxOutputTokens
	} else {
		anthReq.MaxTokens = 4096
	}

	if respReq.Instructions != "" {
		anthReq.System = respReq.Instructions
	}

	if respReq.Reasoning != nil && respReq.Reasoning.Effort != "" {
		effort := respReq.Reasoning.Effort
		if effort == OpenAIReasoningEffortNone {
			anthReq.Thinking = &AnthropicThinkingConfigParam{
				Type: AnthropicThinkingDisabled,
			}
		} else {
			anthReq.Thinking = &AnthropicThinkingConfigParam{
				Type:         AnthropicThinkingEnabled,
				BudgetTokens: 4096,
			}
			anthReq.OutputConfig = &AnthropicOutputConfig{
				Effort: mapReasoningEffortToAnthropic(effort),
			}
		}
	}

	switch input := respReq.Input.(type) {
	case string:
		anthReq.Messages = []AnthropicMessageParam{
			{
				Role:    AnthropicRoleUser,
				Content: input,
			},
		}
	case []interface{}:
		var messages []AnthropicMessageParam
		for _, item := range input {
			if m, ok := item.(map[string]interface{}); ok {
				if m["type"] == "message" {
					messages = append(messages, AnthropicMessageParam{
						Role:    m["role"].(string),
						Content: m["content"],
					})
				}
			}
		}
		anthReq.Messages = messages
	}

	// Unmapped OpenAI Responses fields that have Anthropic equivalents (not yet implemented):
	//   Temperature → AnthropicMessageRequest.Temperature
	//   TopP → AnthropicMessageRequest.TopP
	//   Stream → AnthropicMessageRequest.Stream
	//   Stop → AnthropicMessageRequest.StopSequences (different format)
	//   Tools/ToolChoice → AnthropicMessageRequest.Tools/ToolChoice
	//   Metadata → AnthropicMessageRequest.Metadata (different type)
	//   Text → AnthropicMessageRequest.OutputConfig (only Format part)
	// Unmapped — no Anthropic equivalent:
	//   FrequencyPenalty, PresencePenalty, TopLogprobs, ParallelToolCalls,
	//   ServiceTier, Truncation, Background, PreviousResponseID, Prompt,
	//   PromptCacheKey/PromptCacheRetention (Anthropic uses cache control on blocks)

	return anthReq, nil
}

func (c *responsesToAnthropicConverter) ConvertResponse(body []byte) ([]byte, error) {
	var respResp OpenAIResponsesResponse
	if err := json.Unmarshal(body, &respResp); err != nil {
		return nil, err
	}

	anthResp := AnthropicResponse{
		ID:         respResp.ID,
		Type:       AnthropicObjectMessage,
		Role:       AnthropicRoleAssistant,
		Model:      respResp.Model,
							StopReason: stringPtr(AnthropicStopReasonEndTurn),
		Content: []AnthropicContentBlock{
			{
				Type: AnthropicContentTypeText,
				Text: extractResponsesText(respResp.Output),
			},
		},
	}

	if respResp.Usage != nil {
		anthResp.Usage = &AnthropicUsage{
			InputTokens:  respResp.Usage.InputTokens,
			OutputTokens: respResp.Usage.OutputTokens,
		}
	}

	return json.Marshal(anthResp)
}

func (c *responsesToAnthropicConverter) ConvertSSE(r io.ReadCloser) io.ReadCloser {
	pr, pw := io.Pipe()

	go func() {
		defer pw.Close()

		events, errs := ParseSSE(r)

		var messageID string
		var started bool

		for {
			select {
			case event, ok := <-events:
				if !ok {
					return
				}

				switch event.Event {
				case ResponsesSSECreated:
					var evt OpenAIResponsesStreamEvent
					if err := json.Unmarshal([]byte(event.Data), &evt); err != nil {
						continue
					}
					if evt.Response != nil {
						messageID = evt.Response.ID
					}

					writeSSEJSON(pw, AnthropicSSEMessageStartEvent, AnthropicSSEEvent{
						Type: AnthropicSSEMessageStartEvent,
						Message: &AnthropicResponse{
							ID:      messageID,
							Type:    AnthropicObjectMessage,
							Role:    AnthropicRoleAssistant,
							Content: []AnthropicContentBlock{},
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
					started = true

				case ResponsesSSETextDelta:
					if !started {
						continue
					}
					var evt OpenAIResponsesStreamEvent
					if err := json.Unmarshal([]byte(event.Data), &evt); err != nil {
						continue
					}

					writeSSEJSON(pw, AnthropicSSEContentBlockDeltaEvent, AnthropicSSEEvent{
						Type:  AnthropicSSEContentBlockDeltaEvent,
						Index: intPtr(0),
						Delta: &AnthropicSSEDelta{
							Type: AnthropicDeltaTypeTextDelta,
							Text: evt.Delta,
						},
					})

				case ResponsesSSEDone:
					if started {
						writeSSEJSON(pw, AnthropicSSEContentBlockStopEvent, AnthropicSSEEvent{
							Type:  AnthropicSSEContentBlockStopEvent,
							Index: intPtr(0),
						})
						writeSSEJSON(pw, AnthropicSSEMessageDeltaEvent, AnthropicSSEEvent{
							Type: AnthropicSSEMessageDeltaEvent,
							Delta: &AnthropicSSEDelta{
		StopReason: stringPtr(AnthropicStopReasonEndTurn),
							},
						})
						writeSSEJSON(pw, AnthropicSSEMessageStopEvent, AnthropicSSEEvent{
							Type: AnthropicSSEMessageStopEvent,
						})
					}
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

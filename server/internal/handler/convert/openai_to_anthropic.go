package convert

import (
	"encoding/json"
	"errors"
	"io"
	"strings"
	"time"
)

type openaiToAnthropicConverter struct{}

func (c *openaiToAnthropicConverter) ConvertRequest(req any) (any, error) {
	oaiReq, ok := req.(*OpenAIChatRequest)
	if !ok {
		return nil, errors.New("expected *OpenAIChatRequest")
	}

	nonSystem, systemText := splitSystemMessages(oaiReq.Messages)

	anthReq := &AnthropicRequest{
		Model:     oaiReq.Model,
		Messages:  toAnthropicMessages(nonSystem),
		System:    systemText,
		MaxTokens: defaultInt(oaiReq.MaxTokens, 4096),
		Stream:    oaiReq.Stream,
	}

	if oaiReq.Temperature != nil {
		anthReq.Temperature = oaiReq.Temperature
	}
	if oaiReq.TopP != nil {
		anthReq.TopP = oaiReq.TopP
	}
	if oaiReq.Stop != nil && len(oaiReq.Stop.Values) > 0 {
		anthReq.StopSequences = oaiReq.Stop.Values
	}

	return anthReq, nil
}

func splitSystemMessages(messages []OpenAIChatMessage) (nonSystem []OpenAIChatMessage, systemText string) {
	var texts []string
	for _, m := range messages {
		if m.Role == OpenAIRoleSystem {
			switch c := m.Content.(type) {
			case string:
				if c != "" {
					texts = append(texts, c)
				}
			case []interface{}:
				for _, block := range c {
					if bm, ok := block.(map[string]interface{}); ok {
						if text, ok := bm["text"].(string); ok && text != "" {
							texts = append(texts, text)
						}
					}
				}
			}
		} else {
			nonSystem = append(nonSystem, m)
		}
	}
	if len(texts) > 0 {
		systemText = strings.Join(texts, "\n\n")
	}
	return
}

func toAnthropicMessages(messages []OpenAIChatMessage) []AnthropicMessageParam {
	anth := make([]AnthropicMessageParam, 0, len(messages))
	for _, m := range messages {
		anth = append(anth, AnthropicMessageParam{
			Role:    m.Role,
			Content: m.Content,
		})
	}
	return anth
}

func defaultInt(v *int, def int) int {
	if v != nil {
		return *v
	}
	return def
}

func (c *openaiToAnthropicConverter) ConvertResponse(body []byte) ([]byte, error) {
	var anthResp AnthropicResponse
	if err := json.Unmarshal(body, &anthResp); err != nil {
		return nil, err
	}

	text := extractText(anthResp.Content)
	oaiResp := OpenAIChatResponse{
		ID:      anthResp.ID,
		Object:  OpenAIChatObject,
		Created: time.Now().Unix(),
		Model:   anthResp.Model,
		Choices: []OpenAIChatChoice{
			{
				Index: 0,
				Message: OpenAIChatCompletionMessage{
					Role:    OpenAIRoleAssistant,
					Content: stringPtr(text),
				},
				FinishReason: mapAnthropicStopReason(defaultString(anthResp.StopReason, "")),
			},
		},
	}

	if anthResp.Usage != nil {
		oaiResp.Usage = &OpenAICompletionUsage{
			PromptTokens:     anthResp.Usage.InputTokens,
			CompletionTokens: anthResp.Usage.OutputTokens,
			TotalTokens:      anthResp.Usage.InputTokens + anthResp.Usage.OutputTokens,
		}
	}

	return json.Marshal(oaiResp)
}

func defaultString(v *string, def string) string {
	if v != nil {
		return *v
	}
	return def
}

func extractText(blocks []AnthropicContentBlock) string {
	if len(blocks) == 0 {
		return ""
	}
	return blocks[0].Text
}

func mapAnthropicStopReason(stopReason string) string {
	switch stopReason {
	case AnthropicStopReasonEndTurn:
		return OpenAIFinishReasonStop
	case AnthropicStopReasonMaxTokens:
		return OpenAIFinishReasonLength
	case AnthropicStopReasonStopSequence:
		return OpenAIFinishReasonStop
	default:
		return OpenAIFinishReasonStop
	}
}

func (c *openaiToAnthropicConverter) ConvertSSE(r io.ReadCloser) io.ReadCloser {
	pr, pw := io.Pipe()

	go func() {
		defer pw.Close()

		events, errs := ParseSSE(r)
		var model string

		for {
			select {
			case event, ok := <-events:
				if !ok {
					return
				}

				switch event.Event {
				case AnthropicSSEMessageStartEvent:
					var evt AnthropicSSEEvent
					if err := json.Unmarshal([]byte(event.Data), &evt); err != nil {
						continue
					}
					if evt.Message != nil {
						model = evt.Message.Model
					}

					chunk := OpenAIChatStreamChunk{
						ID:     extractSSEID(event.Data),
						Object: OpenAIChatStreamChunkObject,
						Model:  model,
						Choices: []OpenAIChatStreamChoice{
							{
								Index: 0,
								Delta: OpenAIChatDelta{Role: OpenAIRoleAssistant},
							},
						},
					}
					writeSSEJSON(pw, "", chunk)

				case AnthropicSSEContentBlockDeltaEvent:
					var evt AnthropicSSEEvent
					if err := json.Unmarshal([]byte(event.Data), &evt); err != nil {
						continue
					}
					if evt.Delta == nil || evt.Delta.Type != AnthropicDeltaTypeTextDelta {
						continue
					}

					chunk := OpenAIChatStreamChunk{
						Object: OpenAIChatStreamChunkObject,
						Choices: []OpenAIChatStreamChoice{
							{
								Index: 0,
								Delta: OpenAIChatDelta{Content: stringPtr(evt.Delta.Text)},
							},
						},
					}
					writeSSEJSON(pw, "", chunk)

				case AnthropicSSEMessageDeltaEvent:
					var evt AnthropicSSEEvent
					if err := json.Unmarshal([]byte(event.Data), &evt); err != nil {
						continue
					}
					stopReason := ""
					if evt.Delta != nil {
						stopReason = defaultString(evt.Delta.StopReason, "")
					}

					chunk := OpenAIChatStreamChunk{
						Object: OpenAIChatStreamChunkObject,
						Choices: []OpenAIChatStreamChoice{
							{
								Index:         0,
								Delta:         OpenAIChatDelta{},
								FinishReason:  stringPtr(mapAnthropicStopReason(stopReason)),
							},
						},
					}
					writeSSEJSON(pw, "", chunk)

				case AnthropicSSEMessageStopEvent:
					WriteSSE(pw, SSEEvent{Data: SSEDoneMarker})

				case AnthropicSSEPingEvent:
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

func extractSSEID(data string) string {
	var raw map[string]interface{}
	if err := json.Unmarshal([]byte(data), &raw); err != nil {
		return ""
	}
	msg, ok := raw["message"].(map[string]interface{})
	if !ok {
		return ""
	}
	id, _ := msg["id"].(string)
	return id
}

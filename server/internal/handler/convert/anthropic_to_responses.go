package convert

import (
	"encoding/json"
	"errors"
	"io"
	"strings"
)

type anthropicToResponsesConverter struct{}

func (c *anthropicToResponsesConverter) ConvertRequest(req any) (any, error) {
	anthReq, ok := req.(*AnthropicRequest)
	if !ok {
		return nil, errors.New("expected *AnthropicRequest")
	}

	respReq := &OpenAIResponsesRequest{
		Model: anthReq.Model,
	}

	if anthReq.MaxTokens > 0 {
		respReq.MaxOutputTokens = &anthReq.MaxTokens
	}

	var input []interface{}
	for _, m := range anthReq.Messages {
		input = append(input, ResponsesInputItem{
			Type:    ResponsesItemTypeMessage,
			Role:    m.Role,
			Content: m.Content,
		})
	}
	respReq.Input = input

	if anthReq.System != "" {
		respReq.Instructions = anthReq.System
	} else if len(anthReq.Messages) > 0 && anthReq.Messages[0].Role == OpenAIRoleSystem {
		var texts []string
		for _, m := range anthReq.Messages {
			if m.Role == OpenAIRoleSystem {
				switch c := m.Content.(type) {
				case string:
					if c != "" {
						texts = append(texts, c)
					}
				}
			}
		}
		if len(texts) > 0 {
			respReq.Instructions = strings.Join(texts, "\n\n")
		}
	}

	return respReq, nil
}

func (c *anthropicToResponsesConverter) ConvertResponse(body []byte) ([]byte, error) {
	var anthResp AnthropicResponse
	if err := json.Unmarshal(body, &anthResp); err != nil {
		return nil, err
	}

	text := extractText(anthResp.Content)

	respResp := OpenAIResponsesResponse{
		ID:     anthResp.ID,
		Object: ResponsesObject,
		Model:  anthResp.Model,
		Output: []ResponsesOutputItem{
			{
				Type: ResponsesItemTypeMessage,
				ID:   anthResp.ID,
				Role: anthResp.Role,
				Content: []ResponsesContentBlock{
					{
						Type: ResponsesContentTypeText,
						Text: text,
					},
				},
			},
		},
	}

	if anthResp.Usage != nil {
		respResp.Usage = &ResponsesUsage{
			InputTokens:  anthResp.Usage.InputTokens,
			OutputTokens: anthResp.Usage.OutputTokens,
			TotalTokens:  anthResp.Usage.InputTokens + anthResp.Usage.OutputTokens,
		}
	}

	return json.Marshal(respResp)
}

func (c *anthropicToResponsesConverter) ConvertSSE(r io.ReadCloser) io.ReadCloser {
	pr, pw := io.Pipe()

	go func() {
		defer pw.Close()

		events, errs := ParseSSE(r)

		var responseID string
		var started bool

		for {
			select {
			case event, ok := <-events:
				if !ok {
					return
				}

				if event.Data == "" {
					continue
				}

				var evt AnthropicSSEEvent
				if err := json.Unmarshal([]byte(event.Data), &evt); err != nil {
					continue
				}

				switch event.Event {
				case AnthropicSSEMessageStartEvent:
					if evt.Message != nil {
						responseID = evt.Message.ID
					}
					writeSSEJSON(pw, ResponsesSSECreated, OpenAIResponsesStreamEvent{
						Type: ResponsesSSECreated,
						Response: &OpenAIResponsesStreamSummary{
							ID: responseID,
						},
					})
					writeSSEJSON(pw, ResponsesSSEOutputItemAdded, OpenAIResponsesStreamEvent{
						Type: ResponsesSSEOutputItemAdded,
						Item: &ResponsesOutputItem{
							Type: ResponsesItemTypeMessage,
							Role: AnthropicRoleAssistant,
						},
					})
					writeSSEJSON(pw, ResponsesSSEContentPartAdded, OpenAIResponsesStreamEvent{
						Type: ResponsesSSEContentPartAdded,
						Part: &ResponsesContentBlock{
							Type: AnthropicContentTypeText,
						},
					})
					started = true

				case AnthropicSSEContentBlockDeltaEvent:
					if !started {
						continue
					}
					if evt.Delta == nil || evt.Delta.Type != AnthropicDeltaTypeTextDelta {
						continue
					}

					writeSSEJSON(pw, ResponsesSSETextDelta, OpenAIResponsesStreamEvent{
						Type:  ResponsesSSETextDelta,
						Delta: evt.Delta.Text,
					})

				case AnthropicSSEMessageStopEvent:
					if started {
						writeSSEJSON(pw, ResponsesSSETextDone, OpenAIResponsesStreamEvent{
							Type: ResponsesSSETextDone,
						})
						writeSSEJSON(pw, ResponsesSSEOutputItemDone, OpenAIResponsesStreamEvent{
							Type: ResponsesSSEOutputItemDone,
						})
						writeSSEJSON(pw, ResponsesSSEDone, OpenAIResponsesStreamEvent{
							Type: ResponsesSSEDone,
							Response: &OpenAIResponsesStreamSummary{
								ID: responseID,
							},
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

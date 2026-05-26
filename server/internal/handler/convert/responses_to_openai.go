package convert

import (
	"encoding/json"
	"errors"
	"io"
)

type responsesToOpenAIConverter struct{}

func (c *responsesToOpenAIConverter) ConvertRequest(req any) (any, error) {
	respReq, ok := req.(*OpenAIResponsesRequest)
	if !ok {
		return nil, errors.New("expected *OpenAIResponsesRequest")
	}

	var messages []OpenAIChatMessage

	if respReq.Instructions != "" {
		messages = append(messages, OpenAIChatMessage{
			Role:    OpenAIRoleSystem,
			Content: respReq.Instructions,
		})
	}

	switch input := respReq.Input.(type) {
	case string:
		messages = append(messages, OpenAIChatMessage{
			Role:    OpenAIRoleUser,
			Content: input,
		})
	case []interface{}:
		for _, item := range input {
			if m, ok := item.(map[string]interface{}); ok {
				if m["type"] == "message" {
					messages = append(messages, OpenAIChatMessage{
						Role:    m["role"].(string),
						Content: m["content"],
					})
				}
			}
		}
	}

	oaiReq := &OpenAIChatRequest{
		Model:    respReq.Model,
		Messages: messages,
	}

	if respReq.MaxOutputTokens != nil {
		oaiReq.MaxTokens = respReq.MaxOutputTokens
	}

	return oaiReq, nil
}

func (c *responsesToOpenAIConverter) ConvertResponse(body []byte) ([]byte, error) {
	var respResp OpenAIResponsesResponse
	if err := json.Unmarshal(body, &respResp); err != nil {
		return nil, err
	}

	oaiResp := OpenAIChatResponse{
		ID:      respResp.ID,
		Object:  OpenAIChatObject,
		Created: respResp.CreatedAt,
		Model:   respResp.Model,
		Choices: []OpenAIChatChoice{
			{
				Index: 0,
				FinishReason: OpenAIFinishReasonStop,
				Message: OpenAIChatCompletionMessage{
					Role:    OpenAIRoleAssistant,
					Content: stringPtr(extractResponsesText(respResp.Output)),
				},
			},
		},
	}

	if respResp.Usage != nil {
		oaiResp.Usage = &OpenAICompletionUsage{
			PromptTokens:     respResp.Usage.InputTokens,
			CompletionTokens: respResp.Usage.OutputTokens,
			TotalTokens:      respResp.Usage.TotalTokens,
		}
	}

	return json.Marshal(oaiResp)
}

func extractResponsesText(output []ResponsesOutputItem) string {
	if len(output) == 0 {
		return ""
	}
	if len(output[0].Content) == 0 {
		return ""
	}
	return output[0].Content[0].Text
}

func (c *responsesToOpenAIConverter) ConvertSSE(r io.ReadCloser) io.ReadCloser {
	pr, pw := io.Pipe()

	go func() {
		defer pw.Close()

		events, errs := ParseSSE(r)

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
					chunk := OpenAIChatStreamChunk{
						ID:     extractStringFromMap(event.Data, "response", "id"),
						Object: OpenAIChatStreamChunkObject,
						Model:  extractStringFromMap(event.Data, "response", "model"),
						Choices: []OpenAIChatStreamChoice{
							{
								Index: 0,
								Delta: OpenAIChatDelta{Role: OpenAIRoleAssistant},
							},
						},
					}
					if evt.Response != nil {
						chunk.Created = evt.Response.CreatedAt
					}
					writeSSEJSON(pw, "", chunk)

				case ResponsesSSETextDelta:
					var evt OpenAIResponsesStreamEvent
					if err := json.Unmarshal([]byte(event.Data), &evt); err != nil {
						continue
					}
					chunk := OpenAIChatStreamChunk{
						Object: OpenAIChatStreamChunkObject,
						Choices: []OpenAIChatStreamChoice{
							{
								Index: 0,
								Delta: OpenAIChatDelta{Content: stringPtr(evt.Delta)},
							},
						},
					}
					writeSSEJSON(pw, "", chunk)

				case ResponsesSSEDone:
					var evt OpenAIResponsesStreamEvent
					if err := json.Unmarshal([]byte(event.Data), &evt); err != nil {
						continue
					}
					if evt.Response != nil {
						chunk := OpenAIChatStreamChunk{
							Object: OpenAIChatStreamChunkObject,
							Choices: []OpenAIChatStreamChoice{
								{
									Index:         0,
									Delta:         OpenAIChatDelta{},
									FinishReason: stringPtr(OpenAIFinishReasonStop),
								},
							},
						}
						if evt.Response.Usage != nil {
							chunk.ID = evt.Response.ID
						}
						writeSSEJSON(pw, "", chunk)
					}
					WriteSSE(pw, SSEEvent{Data: SSEDoneMarker})

				case ResponsesSSEError:
					writeSSEJSON(pw, "", map[string]interface{}{
						"error": map[string]interface{}{
							"message": event.Data,
						},
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

func extractStringFromMap(data string, keys ...string) string {
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(data), &m); err != nil {
		return ""
	}
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

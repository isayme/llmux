package convert

import (
	"encoding/json"
	"errors"
	"io"
	"strings"
)

type openaiToResponsesConverter struct{}

func (c *openaiToResponsesConverter) ConvertRequest(req any) (any, error) {
	oaiReq, ok := req.(*OpenAIChatRequest)
	if !ok {
		return nil, errors.New("expected *OpenAIChatRequest")
	}

	respReq := &OpenAIResponsesRequest{
		Model: oaiReq.Model,
	}

	if oaiReq.MaxTokens != nil {
		respReq.MaxOutputTokens = oaiReq.MaxTokens
	}

	var input []interface{}
	var instructions []string

	for _, m := range oaiReq.Messages {
		if m.Role == OpenAIRoleSystem {
			switch c := m.Content.(type) {
			case string:
				if c != "" {
					instructions = append(instructions, c)
				}
			}
			continue
		}
		input = append(input, ResponsesInputItem{
			Type:    ResponsesItemTypeMessage,
			Role:    m.Role,
			Content: m.Content,
		})
	}

	respReq.Input = input
	if len(instructions) > 0 {
		respReq.Instructions = strings.Join(instructions, "\n\n")
	}

	// Unmapped OpenAI Chat fields that have Responses equivalents (not yet implemented):
	//   Temperature → OpenAIResponsesRequest.Temperature
	//   TopP → OpenAIResponsesRequest.TopP
	//   Stream → OpenAIResponsesRequest.Stream
	//   Stop → OpenAIResponsesRequest.Stop
	//   Store → OpenAIResponsesRequest.Store
	//   Metadata → OpenAIResponsesRequest.Metadata
	//   Tools/ToolChoice → OpenAIResponsesRequest.Tools/ToolChoice
	//   ReasoningEffort → OpenAIResponsesRequest.Reasoning
	//   ResponseFormat → OpenAIResponsesRequest.Text
	//   FrequencyPenalty, PresencePenalty → both have equivalents
	//   ServiceTier → OpenAIResponsesRequest.ServiceTier
	//   ParallelToolCalls → OpenAIResponsesRequest.ParallelToolCalls
	//   TopLogprobs, PromptCacheKey, PromptCacheRetention → both have equivalents
	// Unmapped — no Responses equivalent:
	//   LogitBias, Seed, Prediction, Modalities, N, User (deprecated)

	return respReq, nil
}

func (c *openaiToResponsesConverter) ConvertResponse(body []byte) ([]byte, error) {
	var oaiResp OpenAIChatResponse
	if err := json.Unmarshal(body, &oaiResp); err != nil {
		return nil, err
	}

	respResp := OpenAIResponsesResponse{
		ID:     oaiResp.ID,
		Object: ResponsesObject,
		Model:  oaiResp.Model,
	}

	respResp.CreatedAt = oaiResp.Created

	content := extractChatContent(oaiResp.Choices)
	respResp.Output = []ResponsesOutputItem{
		{
			Type: ResponsesItemTypeMessage,
			ID:   oaiResp.ID,
			Role: OpenAIRoleAssistant,
			Content: []ResponsesContentBlock{
				{
					Type: ResponsesContentTypeText,
					Text: content,
				},
			},
		},
	}

	if oaiResp.Usage != nil {
		respResp.Usage = &ResponsesUsage{
			InputTokens:  oaiResp.Usage.PromptTokens,
			OutputTokens: oaiResp.Usage.CompletionTokens,
			TotalTokens:  oaiResp.Usage.TotalTokens,
		}
	}

	// Unmapped OpenAI Chat response fields:
	//   Choices[].Message.ToolCalls → not converted to Responses function_call/custom items
	//   Choices[].Message.Refusal → no Responses equivalent
	//   ServiceTier, SystemFingerprint → no Responses equivalent

	return json.Marshal(respResp)
}

func (c *openaiToResponsesConverter) ConvertSSE(r io.ReadCloser) io.ReadCloser {
	pr, pw := io.Pipe()

	go func() {
		defer pw.Close()

		events, errs := ParseSSE(r)

		var responseID string
		var responseModel string
		var started bool
		var hadContent bool

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
					if hadContent {
						writeSSEJSON(pw, ResponsesSSETextDone, OpenAIResponsesStreamEvent{
							Type: ResponsesSSETextDone,
						})
						writeSSEJSON(pw, ResponsesSSEOutputItemDone, OpenAIResponsesStreamEvent{
							Type: ResponsesSSEOutputItemDone,
						})
					}
					writeSSEJSON(pw, ResponsesSSEDone, OpenAIResponsesStreamEvent{
						Type: ResponsesSSEDone,
						Response: &OpenAIResponsesStreamSummary{
							ID:    responseID,
							Model: responseModel,
						},
					})
					continue
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
					responseID = chunk.ID
					responseModel = chunk.Model

					writeSSEJSON(pw, ResponsesSSECreated, OpenAIResponsesStreamEvent{
						Type: ResponsesSSECreated,
						Response: &OpenAIResponsesStreamSummary{
							ID:    responseID,
							Model: responseModel,
						},
					})
					writeSSEJSON(pw, ResponsesSSEOutputItemAdded, OpenAIResponsesStreamEvent{
						Type: ResponsesSSEOutputItemAdded,
						Item: &ResponsesOutputItem{
							Type: ResponsesItemTypeMessage,
							Role: OpenAIRoleAssistant,
						},
					})
					writeSSEJSON(pw, ResponsesSSEContentPartAdded, OpenAIResponsesStreamEvent{
						Type: ResponsesSSEContentPartAdded,
						Part: &ResponsesContentBlock{
							Type: AnthropicContentTypeText,
						},
					})
				}

				delta := chunk.Choices[0].Delta
				if delta.Content != nil && *delta.Content != "" {
					hadContent = true
					writeSSEJSON(pw, ResponsesSSETextDelta, OpenAIResponsesStreamEvent{
						Type:  ResponsesSSETextDelta,
						Delta: *delta.Content,
					})
				}

				finishReason := chunk.Choices[0].FinishReason
				if finishReason != nil && *finishReason != "" {
					if hadContent {
						writeSSEJSON(pw, ResponsesSSETextDone, OpenAIResponsesStreamEvent{
							Type: ResponsesSSETextDone,
						})
					}
					writeSSEJSON(pw, ResponsesSSEOutputItemDone, OpenAIResponsesStreamEvent{
						Type: ResponsesSSEOutputItemDone,
					})
					writeSSEJSON(pw, ResponsesSSEDone, OpenAIResponsesStreamEvent{
						Type: ResponsesSSEDone,
						Response: &OpenAIResponsesStreamSummary{
							ID:    responseID,
							Model: responseModel,
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

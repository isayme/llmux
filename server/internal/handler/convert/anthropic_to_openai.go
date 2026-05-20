package convert

import (
	"encoding/json"
	"io"
)

// anthropicToOpenAIConverter converts Anthropic-format requests and OpenAI-format
// responses so that an Anthropic client can communicate with an OpenAI provider.
type anthropicToOpenAIConverter struct{}

// --- Request: Anthropic → OpenAI ---

func (c *anthropicToOpenAIConverter) ConvertRequest(req map[string]interface{}) map[string]interface{} {
	out := copyMap(req,
		"system",         // moved into messages array
		"stop_sequences", // renamed to stop
	)

	out = c.injectSystemIntoMessages(out, req)
	out = c.convertStopSequencesToStop(out, req)
	return out
}

// injectSystemIntoMessages moves Anthropic's top-level "system" field into the
// OpenAI messages array as a message with role="system".
// Anthropic uses a dedicated "system" field; OpenAI expects it inline.
func (c *anthropicToOpenAIConverter) injectSystemIntoMessages(out, req map[string]interface{}) map[string]interface{} {
	system, ok := req["system"]
	if !ok {
		return out
	}

	var systemMsg map[string]interface{}

	switch s := system.(type) {
	case string:
		if s != "" {
			systemMsg = map[string]interface{}{"role": "system", "content": s}
		}
	case []interface{}:
		// Anthropic system can be an array of text content blocks.
		var texts []string
		for _, block := range s {
			if bm, ok := block.(map[string]interface{}); ok {
				if text, _ := bm["text"].(string); text != "" {
					texts = append(texts, text)
				}
			}
		}
		if len(texts) > 0 {
			content := texts[0]
			for i := 1; i < len(texts); i++ {
				content += "\n\n" + texts[i]
			}
			systemMsg = map[string]interface{}{"role": "system", "content": content}
		}
	}

	if systemMsg != nil {
		messages, _ := req["messages"].([]interface{})
		out["messages"] = append([]interface{}{systemMsg}, messages...)
	}
	return out
}

// convertStopSequencesToStop renames Anthropic's "stop_sequences" ([]string)
// to OpenAI's "stop" field.
func (c *anthropicToOpenAIConverter) convertStopSequencesToStop(out, req map[string]interface{}) map[string]interface{} {
	seqs, ok := req["stop_sequences"].([]interface{})
	if !ok {
		return out
	}
	strs := make([]string, 0, len(seqs))
	for _, s := range seqs {
		if str, ok := s.(string); ok {
			strs = append(strs, str)
		}
	}
	out["stop"] = strs
	return out
}

// --- Response: OpenAI → Anthropic ---

func (c *anthropicToOpenAIConverter) ConvertResponse(body []byte) ([]byte, error) {
	var o map[string]interface{}
	if err := json.Unmarshal(body, &o); err != nil {
		return nil, err
	}

	out := copyMap(o,
		"object",  // replaced with type: "message"
		"choices", // restructured into content[]
		"usage",   // field names differ
	)

	out["type"] = "message"
	out["role"] = "assistant"
	content, finishReason := c.extractContentAndFinishReason(o)
	stopReason := c.mapFinishReason(finishReason)
	usage := c.remapUsageToAnthropic(o)

	out["content"] = []map[string]interface{}{
		{"type": "text", "text": content},
	}
	out["stop_reason"] = stopReason
	out["usage"] = usage

	return json.Marshal(out)
}

// extractContentAndFinishReason pulls the assistant message content and
// finish_reason from OpenAI's choices[0].
// OpenAI nests these in choices[0].message and choices[0].finish_reason.
func (c *anthropicToOpenAIConverter) extractContentAndFinishReason(o map[string]interface{}) (content string, finishReason string) {
	choices, _ := o["choices"].([]interface{})
	if len(choices) == 0 {
		return "", ""
	}
	choice, ok := choices[0].(map[string]interface{})
	if !ok {
		return "", ""
	}
	if msg, ok := choice["message"].(map[string]interface{}); ok {
		content, _ = msg["content"].(string)
	}
	finishReason, _ = choice["finish_reason"].(string)
	return
}

// mapFinishReason converts OpenAI's finish_reason to Anthropic's stop_reason.
// stop → end_turn, length → max_tokens.
func (c *anthropicToOpenAIConverter) mapFinishReason(finishReason string) string {
	switch finishReason {
	case "stop":
		return "end_turn"
	case "length":
		return "max_tokens"
	default:
		return "end_turn"
	}
}

// remapUsageToAnthropic converts OpenAI usage (prompt_tokens, completion_tokens)
// to Anthropic usage (input_tokens, output_tokens).
func (c *anthropicToOpenAIConverter) remapUsageToAnthropic(o map[string]interface{}) map[string]interface{} {
	u, ok := o["usage"].(map[string]interface{})
	if !ok {
		return nil
	}
	prompt, _ := u["prompt_tokens"].(float64)
	completion, _ := u["completion_tokens"].(float64)
	return map[string]interface{}{
		"input_tokens":  int(prompt),
		"output_tokens": int(completion),
	}
}

// --- SSE: OpenAI → Anthropic ---

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

				if event.Data == "[DONE]" {
					if !started {
						return
					}
					writeSSEJSON(pw, AnthropicSSEMessageStopEvent, map[string]interface{}{
						"type": "message_stop",
					})
					return
				}

				var chunk map[string]interface{}
				if err := json.Unmarshal([]byte(event.Data), &chunk); err != nil {
					continue
				}

				choices, _ := chunk["choices"].([]interface{})
				if len(choices) == 0 {
					continue
				}
				choice, ok := choices[0].(map[string]interface{})
				if !ok {
					continue
				}

				if !started {
					started = true
					if id, ok := chunk["id"].(string); ok {
						messageID = id
					}
					if messageID == "" {
						messageID = "msg_unknown"
					}
					if m, ok := chunk["model"].(string); ok {
						model = m
					}

					writeSSEJSON(pw, AnthropicSSEMessageStartEvent, map[string]interface{}{
						"type": "message_start",
						"message": map[string]interface{}{
							"id":    messageID,
							"type":  "message",
							"role":  "assistant",
							"model": model,
							"usage": map[string]interface{}{
								"input_tokens": 0,
							},
						},
					})

					writeSSEJSON(pw, AnthropicSSEContentBlockStartEvent, map[string]interface{}{
						"type":  "content_block_start",
						"index": 0,
						"content_block": map[string]interface{}{
							"type": "text",
							"text": "",
						},
					})
				}

				delta, _ := choice["delta"].(map[string]interface{})
				if delta != nil {
					if content, ok := delta["content"].(string); ok && content != "" {
						writeSSEJSON(pw, AnthropicSSEContentBlockDeltaEvent, map[string]interface{}{
							"type":  "content_block_delta",
							"index": 0,
							"delta": map[string]interface{}{
								"type": "text_delta",
								"text": content,
							},
						})
					}
				}

				finishReason, _ := choice["finish_reason"].(string)
				if finishReason != "" {
					writeSSEJSON(pw, AnthropicSSEContentBlockStopEvent, map[string]interface{}{
						"type":  "content_block_stop",
						"index": 0,
					})

					writeSSEJSON(pw, AnthropicSSEMessageDeltaEvent, map[string]interface{}{
						"type": "message_delta",
						"delta": map[string]interface{}{
							"stop_reason": c.mapFinishReason(finishReason),
						},
						"usage": map[string]interface{}{
							"output_tokens": outputTokens,
						},
					})

					writeSSEJSON(pw, AnthropicSSEMessageStopEvent, map[string]interface{}{
						"type": "message_stop",
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

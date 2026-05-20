package convert

import (
	"encoding/json"
	"io"
	"time"
)

// openaiToAnthropicConverter converts OpenAI-format requests and Anthropic-format
// responses so that an OpenAI client can communicate with an Anthropic provider.
type openaiToAnthropicConverter struct{}

// --- Request: OpenAI → Anthropic ---

func (c *openaiToAnthropicConverter) ConvertRequest(req map[string]interface{}) map[string]interface{} {
	out := copyMap(req,
		"messages",   // extracted, system role removed
		"stop",       // renamed to stop_sequences
		"max_tokens", // required by Anthropic, defaults to 4096
	)

	out = c.extractSystemFromMessages(out, req)
	out = c.convertStopToStopSequences(out, req)
	out = c.ensureMaxTokens(out, req)
	return out
}

// extractSystemFromMessages removes system-role messages from the messages array
// and places their content into the Anthropic top-level "system" field.
// OpenAI represents system prompts as messages with role="system"; Anthropic uses
// a dedicated "system" field (string or content block array).
func (c *openaiToAnthropicConverter) extractSystemFromMessages(out, req map[string]interface{}) map[string]interface{} {
	messages, ok := req["messages"].([]interface{})
	if !ok {
		return out
	}

	var nonSystem []interface{}
	var systemContents []string

	for _, m := range messages {
		msg, ok := m.(map[string]interface{})
		if !ok {
			nonSystem = append(nonSystem, m)
			continue
		}
		if role, _ := msg["role"].(string); role != "system" {
			nonSystem = append(nonSystem, m)
			continue
		}
		// Collect system message text from both string and content-block formats.
		if content, _ := msg["content"].(string); content != "" {
			systemContents = append(systemContents, content)
		}
		if content, ok := msg["content"].([]interface{}); ok {
			for _, c := range content {
				if cm, ok := c.(map[string]interface{}); ok {
					if text, _ := cm["text"].(string); text != "" {
						systemContents = append(systemContents, text)
					}
				}
			}
		}
	}

	out["messages"] = nonSystem

	if len(systemContents) > 0 {
		systemStr := systemContents[0]
		for i := 1; i < len(systemContents); i++ {
			systemStr += "\n\n" + systemContents[i]
		}
		out["system"] = systemStr
	}
	return out
}

// convertStopToStopSequences renames OpenAI's "stop" field to Anthropic's
// "stop_sequences". OpenAI accepts both a single string or an array;
// Anthropic requires an array.
func (c *openaiToAnthropicConverter) convertStopToStopSequences(out, req map[string]interface{}) map[string]interface{} {
	stopVal, ok := req["stop"]
	if !ok {
		return out
	}
	switch v := stopVal.(type) {
	case string:
		out["stop_sequences"] = []string{v}
	case []interface{}:
		seqs := make([]string, 0, len(v))
		for _, s := range v {
			if str, ok := s.(string); ok {
				seqs = append(seqs, str)
			}
		}
		out["stop_sequences"] = seqs
	}
	return out
}

// ensureMaxTokens guarantees "max_tokens" is present.
// Anthropic requires this field; OpenAI does not. Default to 4096 when missing.
func (c *openaiToAnthropicConverter) ensureMaxTokens(out, req map[string]interface{}) map[string]interface{} {
	if req["max_tokens"] != nil {
		out["max_tokens"] = req["max_tokens"]
	} else {
		out["max_tokens"] = 4096
	}
	return out
}

// --- Response: Anthropic → OpenAI ---

func (c *openaiToAnthropicConverter) ConvertResponse(body []byte) ([]byte, error) {
	var a map[string]interface{}
	if err := json.Unmarshal(body, &a); err != nil {
		return nil, err
	}

	out := copyMap(a,
		"type",        // replaced with object: "chat.completion"
		"content",     // restructured into choices[0].message
		"stop_reason", // renamed to choices[0].finish_reason
		"usage",       // field names differ
		"role",        // added explicitly below
	)

	out["object"] = "chat.completion"
	content := c.extractTextFromContentBlocks(a)
	sr, _ := a["stop_reason"].(string)
	finishReason := c.mapStopReason(sr)
	usage := c.remapUsageToOpenAI(a)

	out["choices"] = []map[string]interface{}{
		{
			"index": 0,
			"message": map[string]interface{}{
				"role":    "assistant",
				"content": content,
			},
			"finish_reason": finishReason,
		},
	}
	out["usage"] = usage

	return json.Marshal(out)
}

// extractTextFromContentBlocks extracts the text from Anthropic's content[0] block.
// Anthropic returns content as [{type: "text", text: "..."}];
// OpenAI uses choices[0].message.content as a plain string.
func (c *openaiToAnthropicConverter) extractTextFromContentBlocks(a map[string]interface{}) string {
	contentArr, _ := a["content"].([]interface{})
	if len(contentArr) == 0 {
		return ""
	}
	block, ok := contentArr[0].(map[string]interface{})
	if !ok {
		return ""
	}
	text, _ := block["text"].(string)
	return text
}

// mapStopReason converts Anthropic's stop_reason to OpenAI's finish_reason.
// end_turn → stop, max_tokens → length, stop_sequence → stop.
func (c *openaiToAnthropicConverter) mapStopReason(stopReason string) string {
	switch stopReason {
	case "end_turn":
		return "stop"
	case "max_tokens":
		return "length"
	case "stop_sequence":
		return "stop"
	default:
		return "stop"
	}
}

// remapUsageToOpenAI converts Anthropic usage (input_tokens, output_tokens) to
// OpenAI usage (prompt_tokens, completion_tokens, total_tokens).
func (c *openaiToAnthropicConverter) remapUsageToOpenAI(a map[string]interface{}) map[string]interface{} {
	u, ok := a["usage"].(map[string]interface{})
	if !ok {
		return nil
	}
	input, _ := u["input_tokens"].(float64)
	output, _ := u["output_tokens"].(float64)
	return map[string]interface{}{
		"prompt_tokens":     int(input),
		"completion_tokens": int(output),
		"total_tokens":      int(input) + int(output),
	}
}

// --- SSE: Anthropic → OpenAI ---

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
					var data map[string]interface{}
					if err := json.Unmarshal([]byte(event.Data), &data); err != nil {
						continue
					}
					if msg, ok := data["message"].(map[string]interface{}); ok {
						if m, ok := msg["model"].(string); ok {
							model = m
						}
					}

					chunk := map[string]interface{}{
						"id":      extractString(data, "message", "id"),
						"object":  "chat.completion.chunk",
						"created": time.Now().Unix(),
						"model":   model,
						"choices": []map[string]interface{}{
							{
								"index": 0,
								"delta": map[string]interface{}{
									"role": "assistant",
								},
							},
						},
					}
					writeSSEJSON(pw, "", chunk)

				case AnthropicSSEContentBlockDeltaEvent:
					var data map[string]interface{}
					if err := json.Unmarshal([]byte(event.Data), &data); err != nil {
						continue
					}
					delta, _ := data["delta"].(map[string]interface{})
					if delta == nil {
						continue
					}
					deltaType, _ := delta["type"].(string)
					if deltaType != "text_delta" {
						continue
					}
					text, _ := delta["text"].(string)

					chunk := map[string]interface{}{
						"object":  "chat.completion.chunk",
						"created": time.Now().Unix(),
						"model":   model,
						"choices": []map[string]interface{}{
							{
								"index": 0,
								"delta": map[string]interface{}{
									"content": text,
								},
							},
						},
					}
					writeSSEJSON(pw, "", chunk)

				case AnthropicSSEMessageDeltaEvent:
					var data map[string]interface{}
					if err := json.Unmarshal([]byte(event.Data), &data); err != nil {
						continue
					}
					delta, _ := data["delta"].(map[string]interface{})
					if delta == nil {
						continue
					}
					stopReason, _ := delta["stop_reason"].(string)
					finish := c.mapStopReason(stopReason)

					chunk := map[string]interface{}{
						"object":  "chat.completion.chunk",
						"created": time.Now().Unix(),
						"model":   model,
						"choices": []map[string]interface{}{
							{
								"index":         0,
								"delta":         map[string]interface{}{},
								"finish_reason": finish,
							},
						},
					}
					writeSSEJSON(pw, "", chunk)

				case AnthropicSSEMessageStopEvent:
					WriteSSE(pw, SSEEvent{Data: "[DONE]"})

				case AnthropicSSEPingEvent:
					// ignore pings
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

package convert

import (
	"encoding/json"
	"io"
)

type responsesToAnthropicConverter struct{}

func (c *responsesToAnthropicConverter) ConvertRequest(req map[string]interface{}) map[string]interface{} {
	out := copyMap(req,
		"input",
		"instructions",
		"max_output_tokens",
	)

	if v, ok := req["max_output_tokens"]; ok {
		out["max_tokens"] = v
	}

	if instructions, ok := req["instructions"]; ok {
		if s, ok := instructions.(string); ok && s != "" {
			out["system"] = s
		}
	}

	if input, ok := req["input"]; ok {
		switch v := input.(type) {
		case string:
			out["messages"] = []interface{}{
				map[string]interface{}{
					"role":    "user",
					"content": v,
				},
			}
		case []interface{}:
			var messages []interface{}
			for _, item := range v {
				if m, ok := item.(map[string]interface{}); ok {
					if m["type"] == "message" {
						messages = append(messages, map[string]interface{}{
							"role":    m["role"],
							"content": m["content"],
						})
					}
				}
			}
			out["messages"] = messages
		}
	}

	return out
}

func (c *responsesToAnthropicConverter) ConvertResponse(body []byte) ([]byte, error) {
	var resp map[string]interface{}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}

	out := copyMap(resp,
		"output",
		"object",
	)

	out["type"] = "message"
	out["role"] = "assistant"

	text := extractResponsesContent(resp)

	out["content"] = []map[string]interface{}{
		{"type": "text", "text": text},
	}

	if u, ok := resp["usage"].(map[string]interface{}); ok {
		out["usage"] = map[string]interface{}{
			"input_tokens":  toFloat64(u["input_tokens"]),
			"output_tokens": toFloat64(u["output_tokens"]),
		}
	}

	return json.Marshal(out)
}

func (c *responsesToAnthropicConverter) ConvertSSE(r io.ReadCloser) io.ReadCloser {
	pr, pw := io.Pipe()

	go func() {
		defer pw.Close()
		defer r.Close()

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
				case "response.created":
					var data map[string]interface{}
					if err := json.Unmarshal([]byte(event.Data), &data); err != nil {
						continue
					}
					messageID = extractString(data, "response", "id")

					writeSSEJSON(pw, AnthropicSSEMessageStartEvent, map[string]interface{}{
						"type": "message_start",
						"message": map[string]interface{}{
							"id":      messageID,
							"type":    "message",
							"role":    "assistant",
							"content": []interface{}{},
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
					started = true

				case "response.text.delta":
					if !started {
						continue
					}
					var data map[string]interface{}
					if err := json.Unmarshal([]byte(event.Data), &data); err != nil {
						continue
					}
					writeSSEJSON(pw, AnthropicSSEContentBlockDeltaEvent, map[string]interface{}{
						"type":  "content_block_delta",
						"index": 0,
						"delta": map[string]interface{}{
							"type": "text_delta",
							"text": data["delta"],
						},
					})

				case "response.done":
					if started {
						writeSSEJSON(pw, AnthropicSSEContentBlockStopEvent, map[string]interface{}{
							"type":  "content_block_stop",
							"index": 0,
						})
						writeSSEJSON(pw, AnthropicSSEMessageDeltaEvent, map[string]interface{}{
							"type": "message_delta",
							"delta": map[string]interface{}{
								"stop_reason": "end_turn",
							},
						})
						writeSSEJSON(pw, AnthropicSSEMessageStopEvent, map[string]interface{}{
							"type": "message_stop",
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

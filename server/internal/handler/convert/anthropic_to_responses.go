package convert

import (
	"encoding/json"
	"io"
)

type anthropicToResponsesConverter struct{}

func (c *anthropicToResponsesConverter) ConvertRequest(req map[string]interface{}) map[string]interface{} {
	out := copyMap(req,
		"messages",
		"system",
		"max_tokens",
	)

	if v, ok := req["max_tokens"]; ok {
		out["max_output_tokens"] = v
	}

	messages, _ := req["messages"].([]interface{})
	var input []interface{}
	for _, m := range messages {
		if msg, ok := m.(map[string]interface{}); ok {
			input = append(input, map[string]interface{}{
				"type":    "message",
				"role":    msg["role"],
				"content": msg["content"],
			})
		}
	}
	out["input"] = input

	if system, ok := req["system"]; ok {
		switch s := system.(type) {
		case string:
			if s != "" {
				out["instructions"] = s
			}
		case []interface{}:
			var texts []string
			for _, block := range s {
				if bm, ok := block.(map[string]interface{}); ok {
					if text, ok := bm["text"].(string); ok && text != "" {
						texts = append(texts, text)
					}
				}
			}
			if len(texts) > 0 {
				content := texts[0]
				for i := 1; i < len(texts); i++ {
					content += "\n\n" + texts[i]
				}
				out["instructions"] = content
			}
		}
	}

	return out
}

func (c *anthropicToResponsesConverter) ConvertResponse(body []byte) ([]byte, error) {
	var resp map[string]interface{}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}

	out := copyMap(resp,
		"type",
		"content",
		"role",
	)

	out["object"] = "response"

	role, _ := resp["role"].(string)
	contentArr, _ := resp["content"].([]interface{})

	var text string
	if len(contentArr) > 0 {
		if block, ok := contentArr[0].(map[string]interface{}); ok {
			text, _ = block["text"].(string)
		}
	}

	output := []interface{}{
		map[string]interface{}{
			"type": "message",
			"id":   resp["id"],
			"role": role,
			"content": []interface{}{
				map[string]interface{}{
					"type": "output_text",
					"text": text,
				},
			},
		},
	}
	out["output"] = output

	if u, ok := resp["usage"].(map[string]interface{}); ok {
		out["usage"] = map[string]interface{}{
			"input_tokens":  toFloat64(u["input_tokens"]),
			"output_tokens": toFloat64(u["output_tokens"]),
			"total_tokens":  toFloat64(u["input_tokens"]) + toFloat64(u["output_tokens"]),
		}
	}

	return json.Marshal(out)
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

				var data map[string]interface{}
				if err := json.Unmarshal([]byte(event.Data), &data); err != nil {
					continue
				}

				switch event.Event {
				case AnthropicSSEMessageStartEvent:
					msg, _ := data["message"].(map[string]interface{})
					if msg != nil {
						responseID, _ = msg["id"].(string)
					}
					writeSSEJSON(pw, "response.created", map[string]interface{}{
						"type": "response.created",
						"response": map[string]interface{}{
							"id": responseID,
						},
					})
					writeSSEJSON(pw, "response.output_item.added", map[string]interface{}{
						"type": "response.output_item.added",
						"item": map[string]interface{}{
							"type": "message",
							"role": "assistant",
						},
					})
					writeSSEJSON(pw, "response.content_part.added", map[string]interface{}{
						"type": "response.content_part.added",
						"part": map[string]interface{}{
							"type": "text",
						},
					})
					started = true

				case AnthropicSSEContentBlockDeltaEvent:
					if !started {
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

					writeSSEJSON(pw, "response.text.delta", map[string]interface{}{
						"type":  "response.text.delta",
						"delta": text,
					})

				case AnthropicSSEMessageStopEvent:
					if started {
						writeSSEJSON(pw, "response.text.done", map[string]interface{}{
							"type": "response.text.done",
							"text": "",
						})
						writeSSEJSON(pw, "response.output_item.done", map[string]interface{}{
							"type": "response.output_item.done",
						})
						writeSSEJSON(pw, "response.done", map[string]interface{}{
							"type": "response.done",
							"response": map[string]interface{}{
								"id": responseID,
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

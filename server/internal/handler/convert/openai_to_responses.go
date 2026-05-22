package convert

import (
	"encoding/json"
	"io"
	"strings"
)

type openaiToResponsesConverter struct{}

func (c *openaiToResponsesConverter) ConvertRequest(req map[string]interface{}) map[string]interface{} {
	out := copyMap(req,
		"messages",
		"max_tokens",
	)

	if v, ok := req["max_tokens"]; ok {
		out["max_output_tokens"] = v
	}

	messages, _ := req["messages"].([]interface{})

	var input []interface{}
	var instructions []string

	for _, m := range messages {
		msg, ok := m.(map[string]interface{})
		if !ok {
			continue
		}
		role, _ := msg["role"].(string)
		if role == "system" {
			if content, ok := msg["content"].(string); ok && content != "" {
				instructions = append(instructions, content)
			}
			continue
		}
		input = append(input, map[string]interface{}{
			"type":    "message",
			"role":    role,
			"content": msg["content"],
		})
	}

	out["input"] = input
	if len(instructions) > 0 {
		out["instructions"] = strings.Join(instructions, "\n\n")
	}

	return out
}

func (c *openaiToResponsesConverter) ConvertResponse(body []byte) ([]byte, error) {
	var resp map[string]interface{}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}

	out := copyMap(resp,
		"object",
		"choices",
		"created",
		"usage",
	)

	out["object"] = "response"

	if created, ok := resp["created"]; ok {
		out["created_at"] = created
	}

	choices, _ := resp["choices"].([]interface{})
	if len(choices) > 0 {
		choice, ok := choices[0].(map[string]interface{})
		if !ok {
			return json.Marshal(out)
		}

		msg, _ := choice["message"].(map[string]interface{})
		if msg == nil {
			return json.Marshal(out)
		}

		role, _ := msg["role"].(string)
		content, _ := msg["content"].(string)

		output := []interface{}{
			map[string]interface{}{
				"type": "message",
				"id":   resp["id"],
				"role": role,
				"content": []interface{}{
					map[string]interface{}{
						"type": "output_text",
						"text": content,
					},
				},
			},
		}
		out["output"] = output
	}

	if u, ok := resp["usage"].(map[string]interface{}); ok {
		out["usage"] = map[string]interface{}{
			"input_tokens":  toFloat64(u["prompt_tokens"]),
			"output_tokens": toFloat64(u["completion_tokens"]),
			"total_tokens":  toFloat64(u["total_tokens"]),
		}
	}

	return json.Marshal(out)
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

				if event.Data == "[DONE]" {
					if hadContent {
						writeSSEJSON(pw, "response.text.done", map[string]interface{}{
							"type": "response.text.done",
							"text": "",
						})
						writeSSEJSON(pw, "response.output_item.done", map[string]interface{}{
							"type": "response.output_item.done",
						})
					}
					writeSSEJSON(pw, "response.done", map[string]interface{}{
						"type": "response.done",
						"response": map[string]interface{}{
							"id":    responseID,
							"model": responseModel,
						},
					})
					continue
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
					responseID, _ = chunk["id"].(string)
					responseModel, _ = chunk["model"].(string)

					writeSSEJSON(pw, "response.created", map[string]interface{}{
						"type": "response.created",
						"response": map[string]interface{}{
							"id":    responseID,
							"model": responseModel,
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
				}

				delta, _ := choice["delta"].(map[string]interface{})
				if delta != nil {
					if content, ok := delta["content"].(string); ok && content != "" {
						hadContent = true
						writeSSEJSON(pw, "response.text.delta", map[string]interface{}{
							"type":  "response.text.delta",
							"delta": content,
						})
					}
				}

				finishReason, _ := choice["finish_reason"].(string)
				if finishReason != "" {
					if hadContent {
						writeSSEJSON(pw, "response.text.done", map[string]interface{}{
							"type": "response.text.done",
							"text": "",
						})
					}
					writeSSEJSON(pw, "response.output_item.done", map[string]interface{}{
						"type": "response.output_item.done",
					})
					writeSSEJSON(pw, "response.done", map[string]interface{}{
						"type": "response.done",
						"response": map[string]interface{}{
							"id":    responseID,
							"model": responseModel,
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

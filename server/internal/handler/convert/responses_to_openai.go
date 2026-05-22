package convert

import (
	"encoding/json"
	"io"
)

type responsesToOpenAIConverter struct{}

func (c *responsesToOpenAIConverter) ConvertRequest(req map[string]interface{}) map[string]interface{} {
	out := copyMap(req,
		"input",
		"instructions",
		"max_output_tokens",
	)

	if v, ok := req["max_output_tokens"]; ok {
		out["max_tokens"] = v
	}

	var messages []interface{}

	if instructions, ok := req["instructions"]; ok {
		if s, ok := instructions.(string); ok && s != "" {
			messages = append(messages, map[string]interface{}{
				"role":    "system",
				"content": s,
			})
		}
	}

	if input, ok := req["input"]; ok {
		switch v := input.(type) {
		case string:
			messages = append(messages, map[string]interface{}{
				"role":    "user",
				"content": v,
			})
		case []interface{}:
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
		}
	}

	out["messages"] = messages
	return out
}

func (c *responsesToOpenAIConverter) ConvertResponse(body []byte) ([]byte, error) {
	var resp map[string]interface{}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}

	out := copyMap(resp,
		"object",
		"output",
		"created_at",
	)

	out["object"] = "chat.completion"

	if created, ok := resp["created_at"]; ok {
		out["created"] = created
	}

	content := extractResponsesContent(resp)

	choice := map[string]interface{}{
		"index":         0,
		"finish_reason": "stop",
		"message": map[string]interface{}{
			"role":    "assistant",
			"content": content,
		},
	}
	out["choices"] = []map[string]interface{}{choice}

	if u, ok := resp["usage"].(map[string]interface{}); ok {
		out["usage"] = map[string]interface{}{
			"prompt_tokens":     toFloat64(u["input_tokens"]),
			"completion_tokens": toFloat64(u["output_tokens"]),
			"total_tokens":      toFloat64(u["input_tokens"]) + toFloat64(u["output_tokens"]),
		}
	}

	return json.Marshal(out)
}

func (c *responsesToOpenAIConverter) ConvertSSE(r io.ReadCloser) io.ReadCloser {
	pr, pw := io.Pipe()

	go func() {
		defer pw.Close()
		defer r.Close()

		events, errs := ParseSSE(r)

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
					resp, _ := data["response"].(map[string]interface{})
					chunk := map[string]interface{}{
						"id":     extractString(data, "response", "id"),
						"object": "chat.completion.chunk",
						"model":  extractString(data, "response", "model"),
						"choices": []map[string]interface{}{
							{
								"index": 0,
								"delta": map[string]interface{}{
									"role": "assistant",
								},
							},
						},
					}
					if resp != nil {
						if c, ok := resp["created_at"]; ok {
							chunk["created"] = c
						}
					}
					writeSSEJSON(pw, "", chunk)

				case "response.text.delta":
					var data map[string]interface{}
					if err := json.Unmarshal([]byte(event.Data), &data); err != nil {
						continue
					}
					chunk := map[string]interface{}{
						"object": "chat.completion.chunk",
						"choices": []map[string]interface{}{
							{
								"index": 0,
								"delta": map[string]interface{}{
									"content": data["delta"],
								},
							},
						},
					}
					writeSSEJSON(pw, "", chunk)

				case "response.done":
					var data map[string]interface{}
					if err := json.Unmarshal([]byte(event.Data), &data); err != nil {
						continue
					}
					resp, _ := data["response"].(map[string]interface{})
					if resp != nil {
						chunk := map[string]interface{}{
							"object": "chat.completion.chunk",
							"choices": []map[string]interface{}{
								{
									"index":         0,
									"delta":         map[string]interface{}{},
									"finish_reason": "stop",
								},
							},
						}
						if usage, ok := resp["usage"].(map[string]interface{}); ok {
							chunk["usage"] = map[string]interface{}{
								"prompt_tokens":     toFloat64(usage["input_tokens"]),
								"completion_tokens": toFloat64(usage["output_tokens"]),
								"total_tokens":      toFloat64(usage["input_tokens"]) + toFloat64(usage["output_tokens"]),
							}
						}
						writeSSEJSON(pw, "", chunk)
					}
					WriteSSE(pw, SSEEvent{Data: "[DONE]"})

				case "error":
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

// extractResponsesContent extracts the text from a Responses API output.
func extractResponsesContent(resp map[string]interface{}) string {
	output, _ := resp["output"].([]interface{})
	if len(output) == 0 {
		return ""
	}
	first, ok := output[0].(map[string]interface{})
	if !ok {
		return ""
	}
	content, _ := first["content"].([]interface{})
	if len(content) == 0 {
		return ""
	}
	firstContent, ok := content[0].(map[string]interface{})
	if !ok {
		return ""
	}
	text, _ := firstContent["text"].(string)
	return text
}

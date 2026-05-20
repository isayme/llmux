package convert

import (
	"encoding/json"
	"io"
	"time"
)

func convertAnthropicSSEToOpenAI(r io.Reader) io.ReadCloser {
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
						if id, ok := msg["id"].(string); ok {
							model = id // placeholder until we get model
						}
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
					finish := mapAnthropicStopReason(stopReason)

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

func convertOpenAISSEToAnthropic(r io.Reader) io.ReadCloser {
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

					// emit message_start
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

					// emit content_block_start
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
					// emit content_block_stop
					writeSSEJSON(pw, AnthropicSSEContentBlockStopEvent, map[string]interface{}{
						"type":  "content_block_stop",
						"index": 0,
					})

					// emit message_delta
					writeSSEJSON(pw, AnthropicSSEMessageDeltaEvent, map[string]interface{}{
						"type": "message_delta",
						"delta": map[string]interface{}{
							"stop_reason": mapOpenAIFinishReason(finishReason),
						},
						"usage": map[string]interface{}{
							"output_tokens": outputTokens,
						},
					})

					// emit message_stop
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

func extractString(m map[string]interface{}, keys ...string) string {
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

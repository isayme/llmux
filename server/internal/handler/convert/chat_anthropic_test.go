package convert

import (
	"testing"
	"encoding/json"
	"io"
	"strings"
)


// ============================================================
// openaiToAnthropicConverter — ConvertRequest
// ============================================================

func TestOpenAIToAnthropic_SystemMessages(t *testing.T) {
	c := &openaiToAnthropicConverter{}
	req := map[string]interface{}{
		"model": "gpt-4",
		"messages": []interface{}{
			map[string]interface{}{"role": "system", "content": "You are helpful."},
			map[string]interface{}{"role": "user", "content": "Hi"},
		},
	}
	out := c.ConvertRequest(req)

	system, _ := out["system"].(string)
	if system != "You are helpful." {
		t.Errorf("expected system, got %q", system)
	}

	msgs := out["messages"].([]interface{})
	if len(msgs) != 1 {
		t.Fatalf("expected 1 non-system message, got %d", len(msgs))
	}
	m := msgs[0].(map[string]interface{})
	if m["role"] != "user" {
		t.Error("expected user message remaining")
	}
}

func TestOpenAIToAnthropic_MultipleSystem(t *testing.T) {
	c := &openaiToAnthropicConverter{}
	req := map[string]interface{}{
		"messages": []interface{}{
			map[string]interface{}{"role": "system", "content": "First rule."},
			map[string]interface{}{"role": "system", "content": "Second rule."},
			map[string]interface{}{"role": "user", "content": "Hi"},
		},
	}
	out := c.ConvertRequest(req)
	system, _ := out["system"].(string)
	expected := "First rule.\n\nSecond rule."
	if system != expected {
		t.Errorf("expected %q, got %q", expected, system)
	}
}

func TestOpenAIToAnthropic_SystemAsContentBlocks(t *testing.T) {
	c := &openaiToAnthropicConverter{}
	req := map[string]interface{}{
		"messages": []interface{}{
			map[string]interface{}{
				"role": "system",
				"content": []interface{}{
					map[string]interface{}{"type": "text", "text": "Rule one."},
					map[string]interface{}{"type": "text", "text": "Rule two."},
				},
			},
		},
	}
	out := c.ConvertRequest(req)
	system, _ := out["system"].(string)
	expected := "Rule one.\n\nRule two."
	if system != expected {
		t.Errorf("expected %q, got %q", expected, system)
	}
}

func TestOpenAIToAnthropic_NoSystemMessage(t *testing.T) {
	c := &openaiToAnthropicConverter{}
	req := map[string]interface{}{
		"messages": []interface{}{
			map[string]interface{}{"role": "user", "content": "Hi"},
		},
	}
	out := c.ConvertRequest(req)
	if _, ok := out["system"]; ok {
		t.Error("expected no system field")
	}
}

func TestOpenAIToAnthropic_StopString(t *testing.T) {
	c := &openaiToAnthropicConverter{}
	req := map[string]interface{}{"stop": "END"}
	out := c.ConvertRequest(req)
	seqs, _ := out["stop_sequences"].([]string)
	if len(seqs) != 1 || seqs[0] != "END" {
		t.Errorf("expected [END], got %v", seqs)
	}
	if _, ok := out["stop"]; ok {
		t.Error("expected stop to be removed")
	}
}

func TestOpenAIToAnthropic_StopArray(t *testing.T) {
	c := &openaiToAnthropicConverter{}
	req := map[string]interface{}{"stop": []interface{}{"A", "B"}}
	out := c.ConvertRequest(req)
	seqs, _ := out["stop_sequences"].([]string)
	if len(seqs) != 2 || seqs[0] != "A" || seqs[1] != "B" {
		t.Errorf("expected [A B], got %v", seqs)
	}
}

func TestOpenAIToAnthropic_NoStop(t *testing.T) {
	c := &openaiToAnthropicConverter{}
	out := c.ConvertRequest(map[string]interface{}{})
	if _, ok := out["stop_sequences"]; ok {
		t.Error("expected no stop_sequences")
	}
}

func TestOpenAIToAnthropic_MaxTokensPresent(t *testing.T) {
	c := &openaiToAnthropicConverter{}
	req := map[string]interface{}{"max_tokens": 100}
	out := c.ConvertRequest(req)
	if out["max_tokens"] != 100 {
		t.Errorf("expected 100, got %v", out["max_tokens"])
	}
}

func TestOpenAIToAnthropic_MaxTokensMissing(t *testing.T) {
	c := &openaiToAnthropicConverter{}
	out := c.ConvertRequest(map[string]interface{}{})
	if out["max_tokens"] != 4096 {
		t.Errorf("expected 4096 default, got %v", out["max_tokens"])
	}
}

func TestOpenAIToAnthropic_PassthroughFields(t *testing.T) {
	c := &openaiToAnthropicConverter{}
	req := map[string]interface{}{
		"model":       "gpt-4",
		"temperature": 0.7,
		"top_p":       0.9,
		"stream":      true,
	}
	out := c.ConvertRequest(req)
	if out["model"] != "gpt-4" {
		t.Error("model not passed through")
	}
	if out["temperature"] != 0.7 {
		t.Error("temperature not passed through")
	}
	if out["stream"] != true {
		t.Error("stream not passed through")
	}
}

// ============================================================
// openaiToAnthropicConverter — ConvertResponse
// ============================================================

func TestOpenAIToAnthropic_ConvertResponse(t *testing.T) {
	c := &openaiToAnthropicConverter{}
	body := []byte(`{
		"id": "msg_01",
		"type": "message",
		"role": "assistant",
		"model": "claude-3",
		"content": [{"type": "text", "text": "Hello world"}],
		"stop_reason": "end_turn",
		"usage": {"input_tokens": 10, "output_tokens": 20}
	}`)
	out, err := c.ConvertResponse(body)
	if err != nil {
		t.Fatal(err)
	}
	var o map[string]interface{}
	json.Unmarshal(out, &o)

	if o["object"] != "chat.completion" {
		t.Error("expected object=chat.completion")
	}
	if _, ok := o["type"]; ok {
		t.Error("expected type to be removed")
	}
	choices := o["choices"].([]interface{})
	choice := choices[0].(map[string]interface{})
	msg := choice["message"].(map[string]interface{})
	if msg["content"] != "Hello world" {
		t.Errorf("expected Hello world, got %v", msg["content"])
	}
	if choice["finish_reason"] != "stop" {
		t.Errorf("expected finish_reason=stop, got %v", choice["finish_reason"])
	}
	usage := o["usage"].(map[string]interface{})
	if usage["prompt_tokens"].(float64) != 10 {
		t.Error("prompt_tokens mismatch")
	}
	if usage["completion_tokens"].(float64) != 20 {
		t.Error("completion_tokens mismatch")
	}
	if usage["total_tokens"].(float64) != 30 {
		t.Error("total_tokens mismatch")
	}
}

func TestOpenAIToAnthropic_ConvertResponse_StopReasonMaxTokens(t *testing.T) {
	c := &openaiToAnthropicConverter{}
	body := []byte(`{"content":[],"stop_reason":"max_tokens"}`)
	out, _ := c.ConvertResponse(body)
	var o map[string]interface{}
	json.Unmarshal(out, &o)
	choices := o["choices"].([]interface{})
	fr := choices[0].(map[string]interface{})["finish_reason"]
	if fr != "length" {
		t.Errorf("expected length, got %v", fr)
	}
}

func TestOpenAIToAnthropic_ConvertResponse_StopReasonStopSequence(t *testing.T) {
	c := &openaiToAnthropicConverter{}
	body := []byte(`{"content":[],"stop_reason":"stop_sequence"}`)
	out, _ := c.ConvertResponse(body)
	var o map[string]interface{}
	json.Unmarshal(out, &o)
	fr := o["choices"].([]interface{})[0].(map[string]interface{})["finish_reason"]
	if fr != "stop" {
		t.Errorf("expected stop, got %v", fr)
	}
}

func TestOpenAIToAnthropic_ConvertResponse_MissingUsage(t *testing.T) {
	c := &openaiToAnthropicConverter{}
	body := []byte(`{"content":[],"stop_reason":"end_turn"}`)
	out, _ := c.ConvertResponse(body)
	var o map[string]interface{}
	json.Unmarshal(out, &o)
	if o["usage"] != nil {
		t.Error("expected nil usage")
	}
}

func TestOpenAIToAnthropic_ConvertResponse_EmptyContent(t *testing.T) {
	c := &openaiToAnthropicConverter{}
	body := []byte(`{"content":[],"stop_reason":"end_turn"}`)
	out, _ := c.ConvertResponse(body)
	var o map[string]interface{}
	json.Unmarshal(out, &o)
	choices := o["choices"].([]interface{})
	msg := choices[0].(map[string]interface{})["message"].(map[string]interface{})
	if msg["content"] != "" {
		t.Error("expected empty content")
	}
}

// ============================================================
// anthropicToOpenAIConverter — ConvertRequest
// ============================================================

func TestAnthropicToOpenAI_SystemString(t *testing.T) {
	c := &anthropicToOpenAIConverter{}
	req := map[string]interface{}{
		"model":  "claude-3",
		"system": "You are helpful.",
		"messages": []interface{}{
			map[string]interface{}{"role": "user", "content": "Hi"},
		},
	}
	out := c.ConvertRequest(req)

	if _, ok := out["system"]; ok {
		t.Error("expected system to be removed")
	}

	msgs := out["messages"].([]interface{})
	if len(msgs) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(msgs))
	}
	sysMsg := msgs[0].(map[string]interface{})
	if sysMsg["role"] != "system" {
		t.Error("expected system role")
	}
	if sysMsg["content"] != "You are helpful." {
		t.Errorf("expected system content, got %v", sysMsg["content"])
	}
}

func TestAnthropicToOpenAI_SystemContentBlocks(t *testing.T) {
	c := &anthropicToOpenAIConverter{}
	req := map[string]interface{}{
		"system": []interface{}{
			map[string]interface{}{"type": "text", "text": "Rule one."},
			map[string]interface{}{"type": "text", "text": "Rule two."},
		},
	}
	out := c.ConvertRequest(req)
	msgs := out["messages"].([]interface{})
	sysMsg := msgs[0].(map[string]interface{})
	if sysMsg["content"] != "Rule one.\n\nRule two." {
		t.Errorf("expected joined content, got %q", sysMsg["content"])
	}
}

func TestAnthropicToOpenAI_NoSystem(t *testing.T) {
	c := &anthropicToOpenAIConverter{}
	req := map[string]interface{}{
		"messages": []interface{}{
			map[string]interface{}{"role": "user", "content": "Hi"},
		},
	}
	out := c.ConvertRequest(req)
	msgs := out["messages"].([]interface{})
	if len(msgs) != 1 {
		t.Errorf("expected 1 message, got %d", len(msgs))
	}
}

func TestAnthropicToOpenAI_StopSequences(t *testing.T) {
	c := &anthropicToOpenAIConverter{}
	req := map[string]interface{}{
		"stop_sequences": []interface{}{"A", "B"},
	}
	out := c.ConvertRequest(req)
	stop, _ := out["stop"].([]string)
	if len(stop) != 2 || stop[0] != "A" {
		t.Errorf("expected [A B], got %v", stop)
	}
	if _, ok := out["stop_sequences"]; ok {
		t.Error("expected stop_sequences to be removed")
	}
}

func TestAnthropicToOpenAI_NoStopSequences(t *testing.T) {
	c := &anthropicToOpenAIConverter{}
	out := c.ConvertRequest(map[string]interface{}{})
	if _, ok := out["stop"]; ok {
		t.Error("expected no stop")
	}
}

// ============================================================
// anthropicToOpenAIConverter — ConvertResponse
// ============================================================

func TestAnthropicToOpenAI_ConvertResponse(t *testing.T) {
	c := &anthropicToOpenAIConverter{}
	body := []byte(`{
		"id": "chatcmpl-123",
		"object": "chat.completion",
		"model": "gpt-4",
		"choices": [{
			"index": 0,
			"message": {"role": "assistant", "content": "Hello"},
			"finish_reason": "stop"
		}],
		"usage": {"prompt_tokens": 10, "completion_tokens": 20}
	}`)
	out, err := c.ConvertResponse(body)
	if err != nil {
		t.Fatal(err)
	}
	var a map[string]interface{}
	json.Unmarshal(out, &a)

	if a["type"] != "message" {
		t.Error("expected type=message")
	}
	if a["role"] != "assistant" {
		t.Error("expected role=assistant")
	}
	if _, ok := a["object"]; ok {
		t.Error("expected object to be removed")
	}
	content := a["content"].([]interface{})
	block := content[0].(map[string]interface{})
	if block["text"] != "Hello" || block["type"] != "text" {
		t.Errorf("expected text block, got %v", block)
	}
	if a["stop_reason"] != "end_turn" {
		t.Errorf("expected end_turn, got %v", a["stop_reason"])
	}
	usage := a["usage"].(map[string]interface{})
	if usage["input_tokens"].(float64) != 10 {
		t.Error("input_tokens mismatch")
	}
	if usage["output_tokens"].(float64) != 20 {
		t.Error("output_tokens mismatch")
	}
}

func TestAnthropicToOpenAI_ConvertResponse_LengthFinish(t *testing.T) {
	c := &anthropicToOpenAIConverter{}
	body := []byte(`{"choices":[{"finish_reason":"length"}]}`)
	out, _ := c.ConvertResponse(body)
	var a map[string]interface{}
	json.Unmarshal(out, &a)
	if a["stop_reason"] != "max_tokens" {
		t.Errorf("expected max_tokens, got %v", a["stop_reason"])
	}
}

func TestAnthropicToOpenAI_ConvertResponse_DefaultFinish(t *testing.T) {
	c := &anthropicToOpenAIConverter{}
	body := []byte(`{"choices":[{}]}`)
	out, _ := c.ConvertResponse(body)
	var a map[string]interface{}
	json.Unmarshal(out, &a)
	if a["stop_reason"] != "end_turn" {
		t.Errorf("expected default end_turn, got %v", a["stop_reason"])
	}
}

func TestAnthropicToOpenAI_ConvertResponse_MissingUsage(t *testing.T) {
	c := &anthropicToOpenAIConverter{}
	body := []byte(`{"choices":[]}`)
	out, _ := c.ConvertResponse(body)
	var a map[string]interface{}
	json.Unmarshal(out, &a)
	if a["usage"] != nil {
		t.Error("expected nil usage")
	}
}

// ============================================================
// SSE: Anthropic SSE → OpenAI SSE
// ============================================================

func TestConvertSSE_AnthropicToOpenAI(t *testing.T) {
	c := &openaiToAnthropicConverter{}
	input := "event: message_start\ndata: {\"type\":\"message_start\",\"message\":{\"id\":\"msg_1\",\"model\":\"claude-3\"}}\n\n" +
		"event: content_block_delta\ndata: {\"type\":\"content_block_delta\",\"delta\":{\"type\":\"text_delta\",\"text\":\"Hello\"}}\n\n" +
		"event: message_delta\ndata: {\"type\":\"message_delta\",\"delta\":{\"stop_reason\":\"end_turn\"}}\n\n" +
		"event: message_stop\ndata: {\"type\":\"message_stop\"}\n\n"

	reader := c.ConvertSSE(io.NopCloser(strings.NewReader(input)))
	result, _ := io.ReadAll(reader)

	resultStr := string(result)

	// Should contain an initial role chunk
	if !strings.Contains(resultStr, `"role":"assistant"`) {
		t.Error("expected role assistant in output")
	}
	// Should contain the content delta
	if !strings.Contains(resultStr, `"content":"Hello"`) {
		t.Error("expected content Hello in output")
	}
	// Should contain finish_reason
	if !strings.Contains(resultStr, `"finish_reason":"stop"`) {
		t.Error("expected finish_reason stop in output")
	}
	// Should end with [DONE]
	if !strings.Contains(resultStr, "[DONE]") {
		t.Error("expected [DONE] in output")
	}
}

func TestConvertSSE_AnthropicToOpenAI_PingIgnored(t *testing.T) {
	c := &openaiToAnthropicConverter{}
	input := "event: ping\ndata: {}\n\n"
	reader := c.ConvertSSE(io.NopCloser(strings.NewReader(input)))
	result, _ := io.ReadAll(reader)
	if len(result) > 0 {
		t.Errorf("expected empty, got %q", string(result))
	}
}

func TestConvertSSE_AnthropicToOpenAI_ContentBlockStartIgnored(t *testing.T) {
	c := &openaiToAnthropicConverter{}
	input := "event: message_start\ndata: {\"type\":\"message_start\",\"message\":{\"id\":\"m1\",\"model\":\"c\"}}\n\n" +
		"event: content_block_start\ndata: {\"type\":\"content_block_start\",\"index\":0,\"content_block\":{\"type\":\"text\",\"text\":\"\"}}\n\n" +
		"event: content_block_stop\ndata: {\"type\":\"content_block_stop\",\"index\":0}\n\n" +
		"event: message_stop\ndata: {\"type\":\"message_stop\"}\n\n"
	reader := c.ConvertSSE(io.NopCloser(strings.NewReader(input)))
	result, _ := io.ReadAll(reader)
	resultStr := string(result)
	if !strings.Contains(resultStr, "[DONE]") {
		t.Error("expected [DONE]")
	}
}

// ============================================================
// SSE: OpenAI SSE → Anthropic SSE
// ============================================================

func TestConvertSSE_OpenAIToAnthropic(t *testing.T) {
	c := &anthropicToOpenAIConverter{}
	input := `data: {"id":"chat-1","model":"gpt-4","choices":[{"delta":{"role":"assistant"}}]}` + "\n\n" +
		`data: {"choices":[{"delta":{"content":"Hi"}}]}` + "\n\n" +
		`data: {"choices":[{"delta":{},"finish_reason":"stop"}]}` + "\n\n" +
		`data: [DONE]` + "\n\n"

	reader := c.ConvertSSE(io.NopCloser(strings.NewReader(input)))
	result, _ := io.ReadAll(reader)
	resultStr := string(result)

	if !strings.Contains(resultStr, "message_start") {
		t.Error("expected message_start event")
	}
	if !strings.Contains(resultStr, "content_block_start") {
		t.Error("expected content_block_start event")
	}
	if !strings.Contains(resultStr, `"type":"text_delta"`) {
		t.Error("expected text_delta type")
	}
	if !strings.Contains(resultStr, `"text":"Hi"`) {
		t.Error("expected text Hi")
	}
	if !strings.Contains(resultStr, "content_block_stop") {
		t.Error("expected content_block_stop event")
	}
	if !strings.Contains(resultStr, "message_delta") {
		t.Error("expected message_delta event")
	}
	if !strings.Contains(resultStr, "message_stop") {
		t.Error("expected message_stop event")
	}
	if !strings.Contains(resultStr, `"stop_reason":"end_turn"`) {
		t.Error("expected end_turn stop_reason")
	}
}

func TestConvertSSE_OpenAIToAnthropic_DoneWithNoStart(t *testing.T) {
	c := &anthropicToOpenAIConverter{}
	input := "data: [DONE]\n\n"
	reader := c.ConvertSSE(io.NopCloser(strings.NewReader(input)))
	result, _ := io.ReadAll(reader)
	if len(result) > 0 {
		t.Errorf("expected empty, got %q", string(result))
	}
}

func TestConvertSSE_OpenAIToAnthropic_EmptyData(t *testing.T) {
	c := &anthropicToOpenAIConverter{}
	input := "data: \n\n" +
		`data: {"id":"chat-1","choices":[{"delta":{"content":"x"}}]}` + "\n\n" +
		"data: [DONE]\n\n"
	reader := c.ConvertSSE(io.NopCloser(strings.NewReader(input)))
	result, _ := io.ReadAll(reader)
	resultStr := string(result)
	if !strings.Contains(resultStr, `"text":"x"`) {
		t.Error("expected text x")
	}
}

func TestConvertSSE_OpenAIToAnthropic_NoChoices(t *testing.T) {
	c := &anthropicToOpenAIConverter{}
	input := `data: {"choices":[]}` + "\n\n"
	reader := c.ConvertSSE(io.NopCloser(strings.NewReader(input)))
	result, _ := io.ReadAll(reader)
	if len(result) > 0 {
		t.Errorf("expected empty, got %q", string(result))
	}
}

// ============================================================
// Edge: unknown fields passthrough in response
// ============================================================

func TestOpenAIToAnthropic_UnknownFieldPassthrough(t *testing.T) {
	c := &openaiToAnthropicConverter{}
	body := []byte(`{"id":"1","content":[],"stop_reason":"end_turn","custom_field":"custom_value"}`)
	out, _ := c.ConvertResponse(body)
	var o map[string]interface{}
	json.Unmarshal(out, &o)
	if o["custom_field"] != "custom_value" {
		t.Error("expected custom_field passthrough")
	}
}

func TestAnthropicToOpenAI_UnknownFieldPassthrough(t *testing.T) {
	c := &anthropicToOpenAIConverter{}
	body := []byte(`{"id":"1","object":"chat.completion","choices":[],"special":true}`)
	out, _ := c.ConvertResponse(body)
	var o map[string]interface{}
	json.Unmarshal(out, &o)
	if o["special"] != true {
		t.Error("expected special field passthrough")
	}
}

// ============================================================
// mapStopReason edge cases
// ============================================================

func TestMapStopReason_Unknown(t *testing.T) {
	c := &openaiToAnthropicConverter{}
	if r := c.mapStopReason("some_unknown_reason"); r != "stop" {
		t.Errorf("expected stop for unknown reason, got %q", r)
	}
}

// ============================================================
// extractTextFromContentBlocks edge cases
// ============================================================

func TestExtractTextFromContentBlocks_EmptyArray(t *testing.T) {
	c := &openaiToAnthropicConverter{}
	if s := c.extractTextFromContentBlocks(map[string]interface{}{"content": []interface{}{}}); s != "" {
		t.Errorf("expected empty, got %q", s)
	}
}

func TestExtractTextFromContentBlocks_NonMapItem(t *testing.T) {
	c := &openaiToAnthropicConverter{}
	if s := c.extractTextFromContentBlocks(map[string]interface{}{"content": []interface{}{"not-a-map"}}); s != "" {
		t.Errorf("expected empty, got %q", s)
	}
}

// ============================================================
// SSE: Anthropic → OpenAI edge cases
// ============================================================

func TestConvertSSE_AnthropicToOpenAI_ErrorIgnored(t *testing.T) {
	c := &openaiToAnthropicConverter{}
	// Unknown event types (including "error") are silently ignored
	input := "event: error\ndata: \"bad request\"\n\n"

	reader := c.ConvertSSE(io.NopCloser(strings.NewReader(input)))
	result, _ := io.ReadAll(reader)

	if len(result) > 0 {
		t.Errorf("expected empty for unhandled event type, got %q", string(result))
	}
}

func TestConvertSSE_OpenAIToAnthropic_InvalidJSON(t *testing.T) {
	c := &anthropicToOpenAIConverter{}
	input := "data: not-json\n\n"

	reader := c.ConvertSSE(io.NopCloser(strings.NewReader(input)))
	result, _ := io.ReadAll(reader)
	// Invalid JSON chunks should be silently skipped
	if len(result) > 0 {
		t.Errorf("expected empty for invalid JSON, got %q", string(result))
	}
}

func TestConvertSSE_OpenAIToAnthropic_NoDelta(t *testing.T) {
	c := &anthropicToOpenAIConverter{}
	input := `data: {"id":"chat-1","choices":[{"delta":{},"finish_reason":"stop"}]}` + "\n\n"

	reader := c.ConvertSSE(io.NopCloser(strings.NewReader(input)))
	result, _ := io.ReadAll(reader)
	resultStr := string(result)

	if !strings.Contains(resultStr, "message_start") {
		t.Error("expected message_start")
	}
	if !strings.Contains(resultStr, "message_stop") {
		t.Error("expected message_stop")
	}
}
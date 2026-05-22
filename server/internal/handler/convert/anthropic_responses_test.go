package convert

import (
	"testing"
	"encoding/json"
	"io"
	"strings"
)


// ============================================================
// responsesToAnthropicConverter — ConvertRequest
// ============================================================

func TestResponsesToAnthropic_ConvertRequest(t *testing.T) {
	c := &responsesToAnthropicConverter{}
	req := map[string]interface{}{
		"model":         "gpt-4o",
		"input":         "Hello!",
		"instructions":  "You are helpful.",
		"max_output_tokens": 100,
	}
	out := c.ConvertRequest(req)

	if out["system"] != "You are helpful." {
		t.Errorf("expected system='You are helpful.', got %v", out["system"])
	}
	messages, _ := out["messages"].([]interface{})
	if len(messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(messages))
	}
	msg, _ := messages[0].(map[string]interface{})
	if msg["role"] != "user" || msg["content"] != "Hello!" {
		t.Errorf("unexpected message: %v", msg)
	}
	if v, ok := out["max_tokens"]; !ok || toFloat64(v) != 100 {
		t.Errorf("expected max_tokens=100, got %v", out["max_tokens"])
	}
}

// ============================================================
// responsesToAnthropicConverter — ConvertResponse
// ============================================================

func TestResponsesToAnthropic_ConvertResponse(t *testing.T) {
	c := &responsesToAnthropicConverter{}
	body := []byte(`{
		"id": "resp_abc",
		"object": "response",
		"model": "gpt-4o",
		"output": [{
			"type": "message",
			"role": "assistant",
			"content": [{"type": "output_text", "text": "Hello!"}]
		}],
		"usage": {"input_tokens": 10, "output_tokens": 5}
	}`)
	out, err := c.ConvertResponse(body)
	if err != nil {
		t.Fatal(err)
	}
	var a map[string]interface{}
	json.Unmarshal(out, &a)

	if a["type"] != "message" {
		t.Errorf("expected type=message, got %v", a["type"])
	}
	if a["role"] != "assistant" {
		t.Error("expected role=assistant")
	}
	content, _ := a["content"].([]interface{})
	block, _ := content[0].(map[string]interface{})
	if block["text"] != "Hello!" {
		t.Errorf("expected text='Hello!', got %v", block["text"])
	}
	usage := a["usage"].(map[string]interface{})
	if usage["input_tokens"].(float64) != 10 {
		t.Error("input_tokens mismatch")
	}
	if usage["output_tokens"].(float64) != 5 {
		t.Error("output_tokens mismatch")
	}
}

// ============================================================
// anthropicToResponsesConverter — ConvertRequest
// ============================================================

func TestAnthropicToResponses_ConvertRequest(t *testing.T) {
	c := &anthropicToResponsesConverter{}
	req := map[string]interface{}{
		"model":  "claude-3",
		"messages": []interface{}{
			map[string]interface{}{"role": "user", "content": "Hi!"},
		},
		"system":     "Be helpful.",
		"max_tokens": 100,
	}
	out := c.ConvertRequest(req)

	if out["instructions"] != "Be helpful." {
		t.Errorf("expected instructions='Be helpful.', got %v", out["instructions"])
	}
	input, _ := out["input"].([]interface{})
	if len(input) != 1 {
		t.Fatalf("expected 1 input, got %d", len(input))
	}
	first, _ := input[0].(map[string]interface{})
	if first["role"] != "user" {
		t.Error("expected user role")
	}
	if v, ok := out["max_output_tokens"]; !ok || toFloat64(v) != 100 {
		t.Errorf("expected max_output_tokens=100, got %v", out["max_output_tokens"])
	}
	if _, ok := out["system"]; ok {
		t.Error("expected system removed")
	}
	if _, ok := out["messages"]; ok {
		t.Error("expected messages removed")
	}
}

func TestAnthropicToResponses_ConvertRequest_SystemContentBlocks(t *testing.T) {
	c := &anthropicToResponsesConverter{}
	req := map[string]interface{}{
		"system": []interface{}{
			map[string]interface{}{"type": "text", "text": "Rule one."},
			map[string]interface{}{"type": "text", "text": "Rule two."},
		},
	}
	out := c.ConvertRequest(req)
	if out["instructions"] != "Rule one.\n\nRule two." {
		t.Errorf("expected joined instructions, got %v", out["instructions"])
	}
}

func TestAnthropicToResponses_ConvertRequest_NoSystem(t *testing.T) {
	c := &anthropicToResponsesConverter{}
	req := map[string]interface{}{
		"messages": []interface{}{
			map[string]interface{}{"role": "user", "content": "Hi"},
		},
	}
	out := c.ConvertRequest(req)
	if _, ok := out["instructions"]; ok {
		t.Error("expected no instructions")
	}
}

// ============================================================
// anthropicToResponsesConverter — ConvertResponse
// ============================================================

func TestAnthropicToResponses_ConvertResponse(t *testing.T) {
	c := &anthropicToResponsesConverter{}
	body := []byte(`{
		"id": "msg_abc",
		"type": "message",
		"role": "assistant",
		"model": "claude-3",
		"content": [{"type": "text", "text": "Hello!"}],
		"usage": {"input_tokens": 10, "output_tokens": 5}
	}`)
	out, err := c.ConvertResponse(body)
	if err != nil {
		t.Fatal(err)
	}
	var o map[string]interface{}
	json.Unmarshal(out, &o)

	if o["object"] != "response" {
		t.Errorf("expected object=response, got %v", o["object"])
	}
	if _, ok := o["type"]; ok {
		t.Error("expected type removed")
	}
	output, _ := o["output"].([]interface{})
	if len(output) != 1 {
		t.Fatalf("expected 1 output, got %d", len(output))
	}
	first, _ := output[0].(map[string]interface{})
	content, _ := first["content"].([]interface{})
	firstContent, _ := content[0].(map[string]interface{})
	if firstContent["text"] != "Hello!" {
		t.Errorf("expected text='Hello!', got %v", firstContent["text"])
	}
}

// ============================================================
// SSE: Responses SSE → Anthropic SSE
// ============================================================

func TestConvertSSE_ResponsesToAnthropic(t *testing.T) {
	c := &responsesToAnthropicConverter{}
	input := "event: response.created\ndata: {\"type\":\"response.created\",\"response\":{\"id\":\"resp_1\"}}\n\n" +
		"event: response.text.delta\ndata: {\"type\":\"response.text.delta\",\"delta\":\"Hello\"}\n\n" +
		"event: response.done\ndata: {\"type\":\"response.done\"}\n\n"

	reader := c.ConvertSSE(io.NopCloser(strings.NewReader(input)))
	result, _ := io.ReadAll(reader)
	resultStr := string(result)

	if !strings.Contains(resultStr, AnthropicSSEMessageStartEvent) {
		t.Error("expected message_start event")
	}
	if !strings.Contains(resultStr, AnthropicSSEContentBlockStartEvent) {
		t.Error("expected content_block_start event")
	}
	if !strings.Contains(resultStr, `"type":"text_delta"`) {
		t.Error("expected text_delta type")
	}
	if !strings.Contains(resultStr, `"text":"Hello"`) {
		t.Error("expected text Hello")
	}
	if !strings.Contains(resultStr, AnthropicSSEContentBlockStopEvent) {
		t.Error("expected content_block_stop event")
	}
	if !strings.Contains(resultStr, AnthropicSSEMessageDeltaEvent) {
		t.Error("expected message_delta event")
	}
	if !strings.Contains(resultStr, AnthropicSSEMessageStopEvent) {
		t.Error("expected message_stop event")
	}
}

// ============================================================
// SSE: Anthropic SSE → Responses SSE
// ============================================================

func TestConvertSSE_AnthropicToResponses(t *testing.T) {
	c := &anthropicToResponsesConverter{}
	input := "event: message_start\ndata: {\"type\":\"message_start\",\"message\":{\"id\":\"msg_1\"}}\n\n" +
		"event: content_block_delta\ndata: {\"type\":\"content_block_delta\",\"index\":0,\"delta\":{\"type\":\"text_delta\",\"text\":\"Hi\"}}\n\n" +
		"event: message_stop\ndata: {\"type\":\"message_stop\"}\n\n"

	reader := c.ConvertSSE(io.NopCloser(strings.NewReader(input)))
	result, _ := io.ReadAll(reader)
	resultStr := string(result)

	if !strings.Contains(resultStr, "response.created") {
		t.Error("expected response.created event")
	}
	if !strings.Contains(resultStr, "response.output_item.added") {
		t.Error("expected response.output_item.added")
	}
	if !strings.Contains(resultStr, "response.content_part.added") {
		t.Error("expected response.content_part.added")
	}
	if !strings.Contains(resultStr, "response.text.delta") {
		t.Error("expected response.text.delta")
	}
	if !strings.Contains(resultStr, `"delta":"Hi"`) {
		t.Error("expected delta Hi")
	}
	if !strings.Contains(resultStr, "response.done") {
		t.Error("expected response.done")
	}
}

func TestResponsesToAnthropic_UnknownFieldPassthrough(t *testing.T) {
	c := &responsesToAnthropicConverter{}
	body := []byte(`{"id":"1","object":"response","output":[],"my_field":"x"}`)
	out, _ := c.ConvertResponse(body)
	var o map[string]interface{}
	json.Unmarshal(out, &o)
	if o["my_field"] != "x" {
		t.Error("expected my_field passthrough")
	}
}

func TestAnthropicToResponses_UnknownFieldPassthrough(t *testing.T) {
	c := &anthropicToResponsesConverter{}
	body := []byte(`{"id":"1","type":"message","content":[],"my_key":"my_val"}`)
	out, _ := c.ConvertResponse(body)
	var o map[string]interface{}
	json.Unmarshal(out, &o)
	if o["my_key"] != "my_val" {
		t.Error("expected my_key passthrough")
	}
}

// ============================================================
// responsesToAnthropicConverter — ConvertRequest edge cases
// ============================================================

func TestResponsesToAnthropic_InputArray(t *testing.T) {
	c := &responsesToAnthropicConverter{}
	req := map[string]interface{}{
		"input": []interface{}{
			map[string]interface{}{"type": "message", "role": "user", "content": "Hello"},
			map[string]interface{}{"type": "message", "role": "assistant", "content": "Hi"},
		},
		"instructions": "Be helpful.",
	}
	out := c.ConvertRequest(req)

	if out["system"] != "Be helpful." {
		t.Errorf("expected system='Be helpful.', got %v", out["system"])
	}
	messages, _ := out["messages"].([]interface{})
	if len(messages) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(messages))
	}
}

func TestResponsesToAnthropic_EmptyInstructions(t *testing.T) {
	c := &responsesToAnthropicConverter{}
	req := map[string]interface{}{
		"input":        "Hello",
		"instructions": "",
	}
	out := c.ConvertRequest(req)
	if _, ok := out["system"]; ok {
		t.Error("expected no system for empty instructions")
	}
}

// ============================================================
// SSE: Responses → Anthropic edge cases
// ============================================================

func TestConvertSSE_ResponsesToAnthropic_NoResponseCreated(t *testing.T) {
	c := &responsesToAnthropicConverter{}
	// text.delta without prior response.created should be ignored
	input := "event: response.text.delta\ndata: {\"type\":\"response.text.delta\",\"delta\":\"Hello\"}\n\n" +
		"event: response.done\ndata: {\"type\":\"response.done\"}\n\n"

	reader := c.ConvertSSE(io.NopCloser(strings.NewReader(input)))
	result, _ := io.ReadAll(reader)
	resultStr := string(result)

	// No events should be emitted without response.created
	if strings.Contains(resultStr, "content_block_delta") {
		t.Error("expected no content_block_delta without response.created")
	}
}
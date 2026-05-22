package convert

import (
	"testing"
	"encoding/json"
	"io"
	"strings"
)


// ============================================================
// responsesToOpenAIConverter — ConvertRequest
// ============================================================

func TestResponsesToOpenAI_StringInput(t *testing.T) {
	c := &responsesToOpenAIConverter{}
	req := map[string]interface{}{
		"model":         "gpt-4o",
		"input":         "Hello!",
		"instructions":  "You are helpful.",
		"max_output_tokens": 100,
	}
	out := c.ConvertRequest(req)

	messages, ok := out["messages"].([]interface{})
	if !ok {
		t.Fatal("expected messages array")
	}
	if len(messages) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(messages))
	}
	sysMsg, _ := messages[0].(map[string]interface{})
	if sysMsg["role"] != "system" || sysMsg["content"] != "You are helpful." {
		t.Errorf("unexpected system message: %v", sysMsg)
	}
	userMsg, _ := messages[1].(map[string]interface{})
	if userMsg["role"] != "user" || userMsg["content"] != "Hello!" {
		t.Errorf("unexpected user message: %v", userMsg)
	}
	if v, ok := out["max_tokens"]; !ok || toFloat64(v) != 100 {
		t.Errorf("expected max_tokens=100, got %v", out["max_tokens"])
	}
	if out["model"] != "gpt-4o" {
		t.Error("expected model passthrough")
	}
	if _, ok := out["max_output_tokens"]; ok {
		t.Error("expected max_output_tokens removed")
	}
	if _, ok := out["instructions"]; ok {
		t.Error("expected instructions removed")
	}
}

func TestResponsesToOpenAI_InputArray(t *testing.T) {
	c := &responsesToOpenAIConverter{}
	req := map[string]interface{}{
		"input": []interface{}{
			map[string]interface{}{"type": "message", "role": "user", "content": "Hi"},
			map[string]interface{}{"type": "message", "role": "assistant", "content": "Hello!"},
		},
	}
	out := c.ConvertRequest(req)
	messages, _ := out["messages"].([]interface{})
	if len(messages) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(messages))
	}
}

func TestResponsesToOpenAI_NoInstructions(t *testing.T) {
	c := &responsesToOpenAIConverter{}
	req := map[string]interface{}{
		"input": "Hi",
	}
	out := c.ConvertRequest(req)
	messages, _ := out["messages"].([]interface{})
	if len(messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(messages))
	}
	msg, _ := messages[0].(map[string]interface{})
	if msg["role"] != "user" {
		t.Error("expected user role")
	}
}

// ============================================================
// responsesToOpenAIConverter — ConvertResponse
// ============================================================

func TestResponsesToOpenAI_ConvertResponse(t *testing.T) {
	c := &responsesToOpenAIConverter{}
	body := []byte(`{
		"id": "resp_abc",
		"object": "response",
		"created_at": 1734567890,
		"model": "gpt-4o",
		"output": [{
			"type": "message",
			"id": "msg_1",
			"role": "assistant",
			"content": [{"type": "output_text", "text": "Hello world"}]
		}],
		"usage": {"input_tokens": 10, "output_tokens": 5, "total_tokens": 15}
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
	if _, ok := o["output"]; ok {
		t.Error("expected output to be removed")
	}
	if created, ok := o["created"]; !ok || created != float64(1734567890) {
		t.Errorf("expected created=1734567890, got %v", o["created"])
	}
	choices := o["choices"].([]interface{})
	choice := choices[0].(map[string]interface{})
	msg := choice["message"].(map[string]interface{})
	if msg["content"] != "Hello world" {
		t.Errorf("expected Hello world, got %v", msg["content"])
	}
	if msg["role"] != "assistant" {
		t.Errorf("expected assistant, got %v", msg["role"])
	}
	if choice["finish_reason"] != "stop" {
		t.Errorf("expected stop, got %v", choice["finish_reason"])
	}
	usage := o["usage"].(map[string]interface{})
	if usage["prompt_tokens"].(float64) != 10 {
		t.Error("prompt_tokens mismatch")
	}
	if usage["completion_tokens"].(float64) != 5 {
		t.Error("completion_tokens mismatch")
	}
	if usage["total_tokens"].(float64) != 15 {
		t.Error("total_tokens mismatch")
	}
}

func TestResponsesToOpenAI_ConvertResponse_EmptyOutput(t *testing.T) {
	c := &responsesToOpenAIConverter{}
	body := []byte(`{"id":"r1","object":"response","output":[]}`)
	out, err := c.ConvertResponse(body)
	if err != nil {
		t.Fatal(err)
	}
	var o map[string]interface{}
	json.Unmarshal(out, &o)
	choices := o["choices"].([]interface{})
	msg := choices[0].(map[string]interface{})["message"].(map[string]interface{})
	if msg["content"] != "" {
		t.Error("expected empty content")
	}
}

// ============================================================
// openaiToResponsesConverter — ConvertRequest
// ============================================================

func TestOpenAIToResponses_ConvertRequest(t *testing.T) {
	c := &openaiToResponsesConverter{}
	req := map[string]interface{}{
		"model": "gpt-4o",
		"messages": []interface{}{
			map[string]interface{}{"role": "system", "content": "Be helpful."},
			map[string]interface{}{"role": "user", "content": "Hi!"},
		},
		"max_tokens": 100,
	}
	out := c.ConvertRequest(req)

	if out["instructions"] != "Be helpful." {
		t.Errorf("expected instructions='Be helpful.', got %v", out["instructions"])
	}
	input, _ := out["input"].([]interface{})
	if len(input) != 1 {
		t.Fatalf("expected 1 input item, got %d", len(input))
	}
	first, _ := input[0].(map[string]interface{})
	if first["role"] != "user" {
		t.Error("expected user role")
	}
	if v, ok := out["max_output_tokens"]; !ok || toFloat64(v) != 100 {
		t.Errorf("expected max_output_tokens=100, got %v", out["max_output_tokens"])
	}
	if _, ok := out["max_tokens"]; ok {
		t.Error("expected max_tokens removed")
	}
}

func TestOpenAIToResponses_ConvertRequest_NoSystem(t *testing.T) {
	c := &openaiToResponsesConverter{}
	req := map[string]interface{}{
		"messages": []interface{}{
			map[string]interface{}{"role": "user", "content": "Hi"},
		},
	}
	out := c.ConvertRequest(req)
	if _, ok := out["instructions"]; ok {
		t.Error("expected no instructions")
	}
	input, _ := out["input"].([]interface{})
	if len(input) != 1 {
		t.Fatalf("expected 1 input, got %d", len(input))
	}
}

// ============================================================
// openaiToResponsesConverter — ConvertResponse
// ============================================================

func TestOpenAIToResponses_ConvertResponse(t *testing.T) {
	c := &openaiToResponsesConverter{}
	body := []byte(`{
		"id": "chatcmpl-123",
		"object": "chat.completion",
		"created": 1734567890,
		"model": "gpt-4o",
		"choices": [{
			"index": 0,
			"message": {"role": "assistant", "content": "Hello!"},
			"finish_reason": "stop"
		}],
		"usage": {"prompt_tokens": 10, "completion_tokens": 5, "total_tokens": 15}
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
	if created, ok := o["created_at"]; !ok || created != float64(1734567890) {
		t.Errorf("expected created_at=1734567890, got %v", o["created_at"])
	}
	output, _ := o["output"].([]interface{})
	if len(output) != 1 {
		t.Fatalf("expected 1 output item, got %d", len(output))
	}
	first, _ := output[0].(map[string]interface{})
	content, _ := first["content"].([]interface{})
	firstContent, _ := content[0].(map[string]interface{})
	if firstContent["text"] != "Hello!" {
		t.Errorf("expected text='Hello!', got %v", firstContent["text"])
	}
	usage := o["usage"].(map[string]interface{})
	if usage["input_tokens"].(float64) != 10 {
		t.Error("input_tokens mismatch")
	}
	if usage["output_tokens"].(float64) != 5 {
		t.Error("output_tokens mismatch")
	}
}

// ============================================================
// SSE: Responses SSE → OpenAI SSE
// ============================================================

func TestConvertSSE_ResponsesToOpenAI(t *testing.T) {
	c := &responsesToOpenAIConverter{}
	input := "event: response.created\ndata: {\"type\":\"response.created\",\"response\":{\"id\":\"resp_1\",\"model\":\"gpt-4o\"}}\n\n" +
		"event: response.text.delta\ndata: {\"type\":\"response.text.delta\",\"delta\":\"Hello\"}\n\n" +
		"event: response.done\ndata: {\"type\":\"response.done\",\"response\":{\"id\":\"resp_1\",\"model\":\"gpt-4o\",\"usage\":{\"input_tokens\":10,\"output_tokens\":5}}}\n\n"

	reader := c.ConvertSSE(io.NopCloser(strings.NewReader(input)))
	result, _ := io.ReadAll(reader)
	resultStr := string(result)

	if !strings.Contains(resultStr, `"role":"assistant"`) {
		t.Error("expected role assistant in output")
	}
	if !strings.Contains(resultStr, `"content":"Hello"`) {
		t.Error("expected content Hello in output")
	}
	if !strings.Contains(resultStr, `"finish_reason":"stop"`) {
		t.Error("expected finish_reason stop in output")
	}
	if !strings.Contains(resultStr, "[DONE]") {
		t.Error("expected [DONE] in output")
	}
	if !strings.Contains(resultStr, `"prompt_tokens":10`) {
		t.Error("expected prompt_tokens=10 in output")
	}
}

// ============================================================
// SSE: OpenAI SSE → Responses SSE
// ============================================================

func TestConvertSSE_OpenAIToResponses(t *testing.T) {
	c := &openaiToResponsesConverter{}
	input := `data: {"id":"chat-1","model":"gpt-4o","choices":[{"delta":{"role":"assistant"}}]}` + "\n\n" +
		`data: {"choices":[{"delta":{"content":"Hi"}}]}` + "\n\n" +
		`data: {"choices":[{"delta":{},"finish_reason":"stop"}]}` + "\n\n" +
		`data: [DONE]` + "\n\n"

	reader := c.ConvertSSE(io.NopCloser(strings.NewReader(input)))
	result, _ := io.ReadAll(reader)
	resultStr := string(result)

	if !strings.Contains(resultStr, "response.created") {
		t.Error("expected response.created event")
	}
	if !strings.Contains(resultStr, "response.output_item.added") {
		t.Error("expected response.output_item.added event")
	}
	if !strings.Contains(resultStr, "response.content_part.added") {
		t.Error("expected response.content_part.added event")
	}
	if !strings.Contains(resultStr, "response.text.delta") {
		t.Error("expected response.text.delta event")
	}
	if !strings.Contains(resultStr, `"delta":"Hi"`) {
		t.Error("expected delta Hi")
	}
	if !strings.Contains(resultStr, "response.done") {
		t.Error("expected response.done event")
	}
}

// ============================================================
// Edge: unknown field passthrough in response — new converters
// ============================================================

func TestResponsesToOpenAI_UnknownFieldPassthrough(t *testing.T) {
	c := &responsesToOpenAIConverter{}
	body := []byte(`{"id":"1","object":"response","output":[],"custom_field":"custom_value"}`)
	out, _ := c.ConvertResponse(body)
	var o map[string]interface{}
	json.Unmarshal(out, &o)
	if o["custom_field"] != "custom_value" {
		t.Error("expected custom_field passthrough")
	}
}

func TestOpenAIToResponses_UnknownFieldPassthrough(t *testing.T) {
	c := &openaiToResponsesConverter{}
	body := []byte(`{"id":"1","object":"chat.completion","choices":[],"special":true}`)
	out, _ := c.ConvertResponse(body)
	var o map[string]interface{}
	json.Unmarshal(out, &o)
	if o["special"] != true {
		t.Error("expected special field passthrough")
	}
}

// ============================================================
// extractResponsesContent edge cases
// ============================================================

func TestExtractResponsesContent_OutputNotMap(t *testing.T) {
	resp := map[string]interface{}{
		"output": []interface{}{"not-a-map"},
	}
	if s := extractResponsesContent(resp); s != "" {
		t.Errorf("expected empty, got %q", s)
	}
}

func TestExtractResponsesContent_EmptyContentArray(t *testing.T) {
	resp := map[string]interface{}{
		"output": []interface{}{
			map[string]interface{}{"type": "message", "content": []interface{}{}},
		},
	}
	if s := extractResponsesContent(resp); s != "" {
		t.Errorf("expected empty, got %q", s)
	}
}

func TestExtractResponsesContent_ContentNotMap(t *testing.T) {
	resp := map[string]interface{}{
		"output": []interface{}{
			map[string]interface{}{"type": "message", "content": []interface{}{"not-a-map"}},
		},
	}
	if s := extractResponsesContent(resp); s != "" {
		t.Errorf("expected empty, got %q", s)
	}
}

// ============================================================
// SSE: Responses → OpenAI edge cases
// ============================================================

func TestConvertSSE_ResponsesToOpenAI_ErrorEvent(t *testing.T) {
	c := &responsesToOpenAIConverter{}
	input := "event: error\ndata: \"something went wrong\"\n\n"

	reader := c.ConvertSSE(io.NopCloser(strings.NewReader(input)))
	result, _ := io.ReadAll(reader)
	resultStr := string(result)

	if !strings.Contains(resultStr, `"error"`) {
		t.Error("expected error in output")
	}
}
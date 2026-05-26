package convert

import (
	"encoding/json"
	"io"
	"strings"
	"testing"
)

// ============================================================
// responsesToAnthropicConverter — ConvertRequest
// ============================================================

func TestResponsesToAnthropic_ConvertRequest(t *testing.T) {
	c := &responsesToAnthropicConverter{}
	mt := 100
	req := &OpenAIResponsesRequest{
		Model:           "gpt-4o",
		Input:           "Hello!",
		Instructions:    "You are helpful.",
		MaxOutputTokens: &mt,
	}
	result, err := c.ConvertRequest(req)
	if err != nil {
		t.Fatal(err)
	}
	out := result.(*AnthropicRequest)

	if out.System != "You are helpful." {
		t.Errorf("expected system='You are helpful.', got %v", out.System)
	}
	if len(out.Messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(out.Messages))
	}
	if out.Messages[0].Role != AnthropicRoleUser || out.Messages[0].Content != "Hello!" {
		t.Errorf("unexpected message: %v", out.Messages[0])
	}
	if out.MaxTokens != 100 {
		t.Errorf("expected max_tokens=100, got %d", out.MaxTokens)
	}
}

func TestResponsesToAnthropic_InputArray(t *testing.T) {
	c := &responsesToAnthropicConverter{}
	req := &OpenAIResponsesRequest{
		Input: []interface{}{
			map[string]interface{}{"type": "message", "role": "user", "content": "Hello"},
			map[string]interface{}{"type": "message", "role": "assistant", "content": "Hi"},
		},
		Instructions: "Be helpful.",
	}
	result, err := c.ConvertRequest(req)
	if err != nil {
		t.Fatal(err)
	}
	out := result.(*AnthropicRequest)

	if out.System != "Be helpful." {
		t.Errorf("expected system='Be helpful.', got %v", out.System)
	}
	if len(out.Messages) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(out.Messages))
	}
}

func TestResponsesToAnthropic_EmptyInstructions(t *testing.T) {
	c := &responsesToAnthropicConverter{}
	req := &OpenAIResponsesRequest{
		Input:        "Hello",
		Instructions: "",
	}
	result, err := c.ConvertRequest(req)
	if err != nil {
		t.Fatal(err)
	}
	out := result.(*AnthropicRequest)

	if out.System != "" {
		t.Error("expected no system for empty instructions")
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
	var a AnthropicResponse
	json.Unmarshal(out, &a)

	if a.Type != AnthropicObjectMessage {
		t.Errorf("expected type=message, got %v", a.Type)
	}
	if a.Role != AnthropicRoleAssistant {
		t.Error("expected role=assistant")
	}
	if len(a.Content) != 1 {
		t.Fatal("expected 1 content block")
	}
	if a.Content[0].Text != "Hello!" {
		t.Errorf("expected text='Hello!', got %v", a.Content[0].Text)
	}
	if a.Usage == nil {
		t.Fatal("expected usage")
	}
	if a.Usage.InputTokens != 10 {
		t.Error("input_tokens mismatch")
	}
	if a.Usage.OutputTokens != 5 {
		t.Error("output_tokens mismatch")
	}
}

// ============================================================
// anthropicToResponsesConverter — ConvertRequest
// ============================================================

func TestAnthropicToResponses_ConvertRequest(t *testing.T) {
	c := &anthropicToResponsesConverter{}
	req := &AnthropicRequest{
		Model: "claude-3",
		Messages: []AnthropicMessage{
			{Role: AnthropicRoleUser, Content: "Hi!"},
		},
		System:    "Be helpful.",
		MaxTokens: 100,
	}
	result, err := c.ConvertRequest(req)
	if err != nil {
		t.Fatal(err)
	}
	out := result.(*OpenAIResponsesRequest)

	if out.Instructions != "Be helpful." {
		t.Errorf("expected instructions='Be helpful.', got %v", out.Instructions)
	}
	data, err := json.Marshal(out)
	if err != nil {
		t.Fatal(err)
	}
	var raw map[string]interface{}
	json.Unmarshal(data, &raw)
	input, ok := raw["input"].([]interface{})
	if !ok {
		t.Fatal("expected input array")
	}
	if len(input) != 1 {
		t.Fatalf("expected 1 input, got %d", len(input))
	}
	first, ok := input[0].(map[string]interface{})
	if !ok {
		t.Fatal("expected map in input")
	}
	if first["role"] != AnthropicRoleUser {
		t.Error("expected user role")
	}
	if out.MaxOutputTokens == nil || *out.MaxOutputTokens != 100 {
		t.Errorf("expected max_output_tokens=100, got %v", out.MaxOutputTokens)
	}
}

func TestAnthropicToResponses_ConvertRequest_NoSystem(t *testing.T) {
	c := &anthropicToResponsesConverter{}
	req := &AnthropicRequest{
		Messages: []AnthropicMessage{
			{Role: AnthropicRoleUser, Content: "Hi"},
		},
	}
	result, err := c.ConvertRequest(req)
	if err != nil {
		t.Fatal(err)
	}
	out := result.(*OpenAIResponsesRequest)

	if out.Instructions != "" {
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
	var o OpenAIResponsesResponse
	json.Unmarshal(out, &o)

	if o.Object != ResponsesObject {
		t.Errorf("expected object=response, got %v", o.Object)
	}
	if len(o.Output) != 1 {
		t.Fatalf("expected 1 output, got %d", len(o.Output))
	}
	if len(o.Output[0].Content) != 1 {
		t.Fatalf("expected 1 content block, got %d", len(o.Output[0].Content))
	}
	if o.Output[0].Content[0].Text != "Hello!" {
		t.Errorf("expected text='Hello!', got %v", o.Output[0].Content[0].Text)
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

	if !strings.Contains(resultStr, ResponsesSSECreated) {
		t.Error("expected response.created event")
	}
	if !strings.Contains(resultStr, ResponsesSSEOutputItemAdded) {
		t.Error("expected response.output_item.added")
	}
	if !strings.Contains(resultStr, ResponsesSSEContentPartAdded) {
		t.Error("expected response.content_part.added")
	}
	if !strings.Contains(resultStr, ResponsesSSETextDelta) {
		t.Error("expected response.text.delta")
	}
	if !strings.Contains(resultStr, `"delta":"Hi"`) {
		t.Error("expected delta Hi")
	}
	if !strings.Contains(resultStr, ResponsesSSEDone) {
		t.Error("expected response.done")
	}
}

// ============================================================
// SSE: Responses → Anthropic edge cases
// ============================================================

func TestConvertSSE_ResponsesToAnthropic_NoResponseCreated(t *testing.T) {
	c := &responsesToAnthropicConverter{}
	input := "event: response.text.delta\ndata: {\"type\":\"response.text.delta\",\"delta\":\"Hello\"}\n\n" +
		"event: response.done\ndata: {\"type\":\"response.done\"}\n\n"

	reader := c.ConvertSSE(io.NopCloser(strings.NewReader(input)))
	result, _ := io.ReadAll(reader)
	resultStr := string(result)

	if strings.Contains(resultStr, AnthropicSSEContentBlockDeltaEvent) {
		t.Error("expected no content_block_delta without response.created")
	}
}

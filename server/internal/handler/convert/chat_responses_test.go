package convert

import (
	"encoding/json"
	"io"
	"strings"
	"testing"
)

// ============================================================
// responsesToOpenAIConverter — ConvertRequest
// ============================================================

func TestResponsesToOpenAI_StringInput(t *testing.T) {
	c := &responsesToOpenAIConverter{}
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
	out := result.(*OpenAIChatRequest)

	if len(out.Messages) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(out.Messages))
	}
	if out.Messages[0].Role != OpenAIRoleSystem || out.Messages[0].Content != "You are helpful." {
		t.Errorf("unexpected system message: %v", out.Messages[0])
	}
	if out.Messages[1].Role != OpenAIRoleUser || out.Messages[1].Content != "Hello!" {
		t.Errorf("unexpected user message: %v", out.Messages[1])
	}
	if out.MaxTokens == nil || *out.MaxTokens != 100 {
		t.Errorf("expected max_tokens=100, got %v", out.MaxTokens)
	}
	if out.Model != "gpt-4o" {
		t.Error("expected model passthrough")
	}
}

func TestResponsesToOpenAI_InputArray(t *testing.T) {
	c := &responsesToOpenAIConverter{}
	req := &OpenAIResponsesRequest{
		Input: []interface{}{
			map[string]interface{}{"type": "message", "role": "user", "content": "Hi"},
			map[string]interface{}{"type": "message", "role": "assistant", "content": "Hello!"},
		},
	}
	result, err := c.ConvertRequest(req)
	if err != nil {
		t.Fatal(err)
	}
	out := result.(*OpenAIChatRequest)

	if len(out.Messages) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(out.Messages))
	}
}

func TestResponsesToOpenAI_ReasoningEffort(t *testing.T) {
	c := &responsesToOpenAIConverter{}
	req := &OpenAIResponsesRequest{
		Input: "Hello",
		Reasoning: &OpenAIReasoning{
			Effort: OpenAIReasoningEffortHigh,
		},
	}
	result, err := c.ConvertRequest(req)
	if err != nil {
		t.Fatal(err)
	}
	out := result.(*OpenAIChatRequest)

	if out.ReasoningEffort == nil {
		t.Fatal("expected ReasoningEffort to be set")
	}
	if *out.ReasoningEffort != OpenAIReasoningEffortHigh {
		t.Errorf("expected reasoning_effort=high, got %q", *out.ReasoningEffort)
	}
}

func TestResponsesToOpenAI_ReasoningEffortXHigh(t *testing.T) {
	c := &responsesToOpenAIConverter{}
	req := &OpenAIResponsesRequest{
		Input: "Hello",
		Reasoning: &OpenAIReasoning{
			Effort: "xhigh",
		},
	}
	result, err := c.ConvertRequest(req)
	if err != nil {
		t.Fatal(err)
	}
	out := result.(*OpenAIChatRequest)

	if out.ReasoningEffort == nil {
		t.Fatal("expected ReasoningEffort to be set")
	}
	if *out.ReasoningEffort != "xhigh" {
		t.Errorf("expected reasoning_effort=xhigh, got %q", *out.ReasoningEffort)
	}
}

func TestResponsesToOpenAI_NoReasoning(t *testing.T) {
	c := &responsesToOpenAIConverter{}
	result, err := c.ConvertRequest(&OpenAIResponsesRequest{Input: "Hello"})
	if err != nil {
		t.Fatal(err)
	}
	out := result.(*OpenAIChatRequest)

	if out.ReasoningEffort != nil {
		t.Error("expected no ReasoningEffort when Reasoning not set")
	}
}

func TestResponsesToOpenAI_NoInstructions(t *testing.T) {
	c := &responsesToOpenAIConverter{}
	req := &OpenAIResponsesRequest{
		Input: "Hi",
	}
	result, err := c.ConvertRequest(req)
	if err != nil {
		t.Fatal(err)
	}
	out := result.(*OpenAIChatRequest)

	if len(out.Messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(out.Messages))
	}
	if out.Messages[0].Role != OpenAIRoleUser {
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
	var o OpenAIChatResponse
	json.Unmarshal(out, &o)

	if o.Object != OpenAIChatObject {
		t.Error("expected object=chat.completion")
	}
	if o.Created != 1734567890 {
		t.Errorf("expected created=1734567890, got %d", o.Created)
	}
	if len(o.Choices) != 1 {
		t.Fatal("expected 1 choice")
	}
	if defaultString(o.Choices[0].Message.Content, "") != "Hello world" {
		t.Errorf("expected Hello world, got %v", o.Choices[0].Message.Content)
	}
	if o.Choices[0].Message.Role != OpenAIRoleAssistant {
		t.Errorf("expected assistant, got %v", o.Choices[0].Message.Role)
	}
	if o.Choices[0].FinishReason != OpenAIFinishReasonStop {
		t.Errorf("expected stop, got %v", o.Choices[0].FinishReason)
	}
	if o.Usage == nil {
		t.Fatal("expected usage")
	}
	if o.Usage.PromptTokens != 10 {
		t.Error("prompt_tokens mismatch")
	}
	if o.Usage.CompletionTokens != 5 {
		t.Error("completion_tokens mismatch")
	}
	if o.Usage.TotalTokens != 15 {
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
	var o OpenAIChatResponse
	json.Unmarshal(out, &o)

	if len(o.Choices) != 1 {
		t.Fatal("expected 1 choice")
	}
	if defaultString(o.Choices[0].Message.Content, "") != "" {
		t.Error("expected empty content")
	}
}

// ============================================================
// openaiToResponsesConverter — ConvertRequest
// ============================================================

func TestOpenAIToResponses_ConvertRequest(t *testing.T) {
	c := &openaiToResponsesConverter{}
	mt := 100
	req := &OpenAIChatRequest{
		Model: "gpt-4o",
		Messages: []OpenAIChatMessage{
			{Role: OpenAIRoleSystem, Content: "Be helpful."},
			{Role: OpenAIRoleUser, Content: "Hi!"},
		},
		MaxTokens: &mt,
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
		t.Fatalf("expected 1 input item, got %d", len(input))
	}
	first, ok := input[0].(map[string]interface{})
	if !ok {
		t.Fatal("expected map in input")
	}
	if first["role"] != OpenAIRoleUser {
		t.Error("expected user role")
	}
	if out.MaxOutputTokens == nil || *out.MaxOutputTokens != 100 {
		t.Errorf("expected max_output_tokens=100, got %v", out.MaxOutputTokens)
	}
}

func TestOpenAIToResponses_ReasoningEffort(t *testing.T) {
	c := &openaiToResponsesConverter{}
	effort := OpenAIReasoningEffortHigh
	req := &OpenAIChatRequest{
		ReasoningEffort: &effort,
	}
	result, err := c.ConvertRequest(req)
	if err != nil {
		t.Fatal(err)
	}
	out := result.(*OpenAIResponsesRequest)

	if out.Reasoning == nil {
		t.Fatal("expected Reasoning to be set")
	}
	if out.Reasoning.Effort != OpenAIReasoningEffortHigh {
		t.Errorf("expected effort=high, got %q", out.Reasoning.Effort)
	}
}

func TestOpenAIToResponses_ReasoningEffortNone(t *testing.T) {
	c := &openaiToResponsesConverter{}
	effort := OpenAIReasoningEffortNone
	req := &OpenAIChatRequest{
		ReasoningEffort: &effort,
	}
	result, err := c.ConvertRequest(req)
	if err != nil {
		t.Fatal(err)
	}
	out := result.(*OpenAIResponsesRequest)

	if out.Reasoning == nil {
		t.Fatal("expected Reasoning to be set")
	}
	if out.Reasoning.Effort != OpenAIReasoningEffortNone {
		t.Errorf("expected effort=none, got %q", out.Reasoning.Effort)
	}
}

func TestOpenAIToResponses_NoReasoningEffort(t *testing.T) {
	c := &openaiToResponsesConverter{}
	result, err := c.ConvertRequest(&OpenAIChatRequest{})
	if err != nil {
		t.Fatal(err)
	}
	out := result.(*OpenAIResponsesRequest)

	if out.Reasoning != nil {
		t.Error("expected no Reasoning when reasoning_effort not set")
	}
}

func TestOpenAIToResponses_ConvertRequest_NoSystem(t *testing.T) {
	c := &openaiToResponsesConverter{}
	req := &OpenAIChatRequest{
		Messages: []OpenAIChatMessage{
			{Role: OpenAIRoleUser, Content: "Hi"},
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
	var o OpenAIResponsesResponse
	json.Unmarshal(out, &o)

	if o.Object != ResponsesObject {
		t.Errorf("expected object=response, got %v", o.Object)
	}
	if o.CreatedAt != 1734567890 {
		t.Errorf("expected created_at=1734567890, got %d", o.CreatedAt)
	}
	if len(o.Output) != 1 {
		t.Fatalf("expected 1 output item, got %d", len(o.Output))
	}
	if len(o.Output[0].Content) != 1 {
		t.Fatalf("expected 1 content block, got %d", len(o.Output[0].Content))
	}
	if o.Output[0].Content[0].Text != "Hello!" {
		t.Errorf("expected text='Hello!', got %v", o.Output[0].Content[0].Text)
	}
	if o.Usage == nil {
		t.Fatal("expected usage")
	}
	if o.Usage.InputTokens != 10 {
		t.Error("input_tokens mismatch")
	}
	if o.Usage.OutputTokens != 5 {
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
	if !strings.Contains(resultStr, SSEDoneMarker) {
		t.Error("expected [DONE] in output")
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

	if !strings.Contains(resultStr, ResponsesSSECreated) {
		t.Error("expected response.created event")
	}
	if !strings.Contains(resultStr, ResponsesSSEOutputItemAdded) {
		t.Error("expected response.output_item.added event")
	}
	if !strings.Contains(resultStr, ResponsesSSEContentPartAdded) {
		t.Error("expected response.content_part.added event")
	}
	if !strings.Contains(resultStr, ResponsesSSETextDelta) {
		t.Error("expected response.text.delta event")
	}
	if !strings.Contains(resultStr, `"delta":"Hi"`) {
		t.Error("expected delta Hi")
	}
	if !strings.Contains(resultStr, ResponsesSSEDone) {
		t.Error("expected response.done event")
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

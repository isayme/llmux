package convert

import (
	"encoding/json"
	"io"
	"strings"
	"testing"
)

// ============================================================
// openaiToAnthropicConverter — ConvertRequest
// ============================================================

func TestOpenAIToAnthropic_SystemMessages(t *testing.T) {
	c := &openaiToAnthropicConverter{}
	req := &OpenAIChatRequest{
		Model: "gpt-4",
		Messages: []OpenAIChatMessage{
			{Role: OpenAIRoleSystem, Content: "You are helpful."},
			{Role: OpenAIRoleUser, Content: "Hi"},
		},
	}
	result, err := c.ConvertRequest(req)
	if err != nil {
		t.Fatal(err)
	}
	out := result.(*AnthropicRequest)

	if out.System != "You are helpful." {
		t.Errorf("expected system, got %q", out.System)
	}
	if len(out.Messages) != 1 {
		t.Fatalf("expected 1 non-system message, got %d", len(out.Messages))
	}
	if out.Messages[0].Role != AnthropicRoleUser {
		t.Error("expected user message remaining")
	}
}

func TestOpenAIToAnthropic_MultipleSystem(t *testing.T) {
	c := &openaiToAnthropicConverter{}
	req := &OpenAIChatRequest{
		Messages: []OpenAIChatMessage{
			{Role: OpenAIRoleSystem, Content: "First rule."},
			{Role: OpenAIRoleSystem, Content: "Second rule."},
			{Role: OpenAIRoleUser, Content: "Hi"},
		},
	}
	result, err := c.ConvertRequest(req)
	if err != nil {
		t.Fatal(err)
	}
	out := result.(*AnthropicRequest)

	expected := "First rule.\n\nSecond rule."
	if out.System != expected {
		t.Errorf("expected %q, got %q", expected, out.System)
	}
}

func TestOpenAIToAnthropic_NoSystemMessage(t *testing.T) {
	c := &openaiToAnthropicConverter{}
	req := &OpenAIChatRequest{
		Messages: []OpenAIChatMessage{
			{Role: OpenAIRoleUser, Content: "Hi"},
		},
	}
	result, err := c.ConvertRequest(req)
	if err != nil {
		t.Fatal(err)
	}
	out := result.(*AnthropicRequest)

	if out.System != "" {
		t.Error("expected no system field")
	}
}

func TestOpenAIToAnthropic_StopSequences(t *testing.T) {
	c := &openaiToAnthropicConverter{}
	req := &OpenAIChatRequest{
		Stop: &OpenAIChatCompletionStop{Values: []string{"END"}},
	}
	result, err := c.ConvertRequest(req)
	if err != nil {
		t.Fatal(err)
	}
	out := result.(*AnthropicRequest)

	if len(out.StopSequences) != 1 || out.StopSequences[0] != "END" {
		t.Errorf("expected [END], got %v", out.StopSequences)
	}
}

func TestOpenAIToAnthropic_NoStop(t *testing.T) {
	c := &openaiToAnthropicConverter{}
	result, err := c.ConvertRequest(&OpenAIChatRequest{})
	if err != nil {
		t.Fatal(err)
	}
	out := result.(*AnthropicRequest)

	if len(out.StopSequences) != 0 {
		t.Error("expected no stop_sequences")
	}
}

func TestOpenAIToAnthropic_MaxTokensPresent(t *testing.T) {
	c := &openaiToAnthropicConverter{}
	req := &OpenAIChatRequest{MaxTokens: intPtr(100)}
	result, err := c.ConvertRequest(req)
	if err != nil {
		t.Fatal(err)
	}
	out := result.(*AnthropicRequest)

	if out.MaxTokens != 100 {
		t.Errorf("expected 100, got %d", out.MaxTokens)
	}
}

func TestOpenAIToAnthropic_MaxTokensMissing(t *testing.T) {
	c := &openaiToAnthropicConverter{}
	result, err := c.ConvertRequest(&OpenAIChatRequest{})
	if err != nil {
		t.Fatal(err)
	}
	out := result.(*AnthropicRequest)

	if out.MaxTokens != 4096 {
		t.Errorf("expected 4096 default, got %d", out.MaxTokens)
	}
}

func TestOpenAIToAnthropic_PassthroughFields(t *testing.T) {
	temp := 0.7
	topP := 0.9
	stream := true
	c := &openaiToAnthropicConverter{}
	req := &OpenAIChatRequest{
		Model:       "gpt-4",
		Temperature: &temp,
		TopP:        &topP,
		Stream:      &stream,
	}
	result, err := c.ConvertRequest(req)
	if err != nil {
		t.Fatal(err)
	}
	out := result.(*AnthropicRequest)

	if out.Model != "gpt-4" {
		t.Error("model not passed through")
	}
	if out.Temperature == nil || *out.Temperature != 0.7 {
		t.Error("temperature not passed through")
	}
	if out.Stream == nil || *out.Stream != true {
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
	var o OpenAIChatResponse
	json.Unmarshal(out, &o)

	if o.Object != OpenAIChatObject {
		t.Error("expected object=chat.completion")
	}
	if len(o.Choices) != 1 {
		t.Fatal("expected 1 choice")
	}
	if defaultString(o.Choices[0].Message.Content, "") != "Hello world" {
		t.Errorf("expected Hello world, got %v", o.Choices[0].Message.Content)
	}
	if o.Choices[0].FinishReason != OpenAIFinishReasonStop {
		t.Errorf("expected finish_reason=stop, got %v", o.Choices[0].FinishReason)
	}
	if o.Usage == nil {
		t.Fatal("expected usage")
	}
	if o.Usage.PromptTokens != 10 {
		t.Error("prompt_tokens mismatch")
	}
	if o.Usage.CompletionTokens != 20 {
		t.Error("completion_tokens mismatch")
	}
	if o.Usage.TotalTokens != 30 {
		t.Error("total_tokens mismatch")
	}
}

func TestOpenAIToAnthropic_ConvertResponse_StopReasonMaxTokens(t *testing.T) {
	c := &openaiToAnthropicConverter{}
	body := []byte(`{"content":[{"type":"text","text":""}],"stop_reason":"max_tokens"}`)
	out, _ := c.ConvertResponse(body)
	var o OpenAIChatResponse
	json.Unmarshal(out, &o)
	if o.Choices[0].FinishReason != OpenAIFinishReasonLength {
		t.Errorf("expected length, got %v", o.Choices[0].FinishReason)
	}
}

func TestOpenAIToAnthropic_ConvertResponse_MissingUsage(t *testing.T) {
	c := &openaiToAnthropicConverter{}
	body := []byte(`{"content":[{"type":"text","text":""}],"stop_reason":"end_turn"}`)
	out, _ := c.ConvertResponse(body)
	var o OpenAIChatResponse
	json.Unmarshal(out, &o)
	if o.Usage != nil {
		t.Error("expected nil usage")
	}
}

func TestOpenAIToAnthropic_ConvertResponse_EmptyContent(t *testing.T) {
	c := &openaiToAnthropicConverter{}
	body := []byte(`{"content":[],"stop_reason":"end_turn"}`)
	out, _ := c.ConvertResponse(body)
	var o OpenAIChatResponse
	json.Unmarshal(out, &o)
	if defaultString(o.Choices[0].Message.Content, "") != "" {
		t.Error("expected empty content")
	}
}

// ============================================================
// anthropicToOpenAIConverter — ConvertRequest
// ============================================================

func TestAnthropicToOpenAI_SystemString(t *testing.T) {
	c := &anthropicToOpenAIConverter{}
	req := &AnthropicRequest{
		Model:  "claude-3",
		System: "You are helpful.",
		Messages: []AnthropicMessageParam{
			{Role: OpenAIRoleUser, Content: "Hi"},
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
	if out.Messages[0].Role != OpenAIRoleSystem {
		t.Error("expected system role")
	}
	if out.Messages[0].Content != "You are helpful." {
		t.Errorf("expected system content, got %v", out.Messages[0].Content)
	}
}

func TestAnthropicToOpenAI_NoSystem(t *testing.T) {
	c := &anthropicToOpenAIConverter{}
	req := &AnthropicRequest{
		Messages: []AnthropicMessageParam{
			{Role: AnthropicRoleUser, Content: "Hi"},
		},
	}
	result, err := c.ConvertRequest(req)
	if err != nil {
		t.Fatal(err)
	}
	out := result.(*OpenAIChatRequest)

	if len(out.Messages) != 1 {
		t.Errorf("expected 1 message, got %d", len(out.Messages))
	}
}

func TestAnthropicToOpenAI_StopSequences(t *testing.T) {
	c := &anthropicToOpenAIConverter{}
	req := &AnthropicRequest{
		StopSequences: []string{"A", "B"},
	}
	result, err := c.ConvertRequest(req)
	if err != nil {
		t.Fatal(err)
	}
	out := result.(*OpenAIChatRequest)

	if out.Stop == nil || len(out.Stop.Values) != 2 || out.Stop.Values[0] != "A" {
		t.Errorf("expected [A B], got %v", out.Stop)
	}
}

func TestAnthropicToOpenAI_NoStopSequences(t *testing.T) {
	c := &anthropicToOpenAIConverter{}
	result, err := c.ConvertRequest(&AnthropicRequest{})
	if err != nil {
		t.Fatal(err)
	}
	out := result.(*OpenAIChatRequest)

	if out.Stop != nil {
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
	var a AnthropicResponse
	json.Unmarshal(out, &a)

	if a.Type != AnthropicObjectMessage {
		t.Error("expected type=message")
	}
	if a.Role != AnthropicRoleAssistant {
		t.Error("expected role=assistant")
	}
	if len(a.Content) != 1 {
		t.Fatal("expected 1 content block")
	}
	if a.Content[0].Text != "Hello" || a.Content[0].Type != AnthropicContentTypeText {
		t.Errorf("expected text block, got %+v", a.Content[0])
	}
	if defaultString(a.StopReason, "") != AnthropicStopReasonEndTurn {
		t.Errorf("expected end_turn, got %v", a.StopReason)
	}
	if a.Usage == nil {
		t.Fatal("expected usage")
	}
	if a.Usage.InputTokens != 10 {
		t.Error("input_tokens mismatch")
	}
	if a.Usage.OutputTokens != 20 {
		t.Error("output_tokens mismatch")
	}
}

func TestAnthropicToOpenAI_ConvertResponse_LengthFinish(t *testing.T) {
	c := &anthropicToOpenAIConverter{}
	body := []byte(`{"choices":[{"finish_reason":"length"}]}`)
	out, _ := c.ConvertResponse(body)
	var a AnthropicResponse
	json.Unmarshal(out, &a)
	if defaultString(a.StopReason, "") != AnthropicStopReasonMaxTokens {
		t.Errorf("expected max_tokens, got %v", a.StopReason)
	}
}

func TestAnthropicToOpenAI_ConvertResponse_MissingUsage(t *testing.T) {
	c := &anthropicToOpenAIConverter{}
	body := []byte(`{"choices":[]}`)
	out, _ := c.ConvertResponse(body)
	var a AnthropicResponse
	json.Unmarshal(out, &a)
	if a.Usage != nil {
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
	if !strings.Contains(resultStr, SSEDoneMarker) {
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

	if !strings.Contains(resultStr, AnthropicSSEMessageStartEvent) {
		t.Error("expected message_start event")
	}
	if !strings.Contains(resultStr, AnthropicSSEContentBlockStartEvent) {
		t.Error("expected content_block_start event")
	}
	if !strings.Contains(resultStr, `"type":"text_delta"`) {
		t.Error("expected text_delta type")
	}
	if !strings.Contains(resultStr, `"text":"Hi"`) {
		t.Error("expected text Hi")
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
// Edge: invalid SSE JSON
// ============================================================

func TestConvertSSE_OpenAIToAnthropic_InvalidJSON(t *testing.T) {
	c := &anthropicToOpenAIConverter{}
	input := "data: not-json\n\n"

	reader := c.ConvertSSE(io.NopCloser(strings.NewReader(input)))
	result, _ := io.ReadAll(reader)
	if len(result) > 0 {
		t.Errorf("expected empty for invalid JSON, got %q", string(result))
	}
}

func TestConvertSSE_AnthropicToOpenAI_ErrorIgnored(t *testing.T) {
	c := &openaiToAnthropicConverter{}
	input := "event: error\ndata: \"bad request\"\n\n"

	reader := c.ConvertSSE(io.NopCloser(strings.NewReader(input)))
	result, _ := io.ReadAll(reader)

	if len(result) > 0 {
		t.Errorf("expected empty for unhandled event type, got %q", string(result))
	}
}

func TestConvertSSE_OpenAIToAnthropic_NoDelta(t *testing.T) {
	c := &anthropicToOpenAIConverter{}
	input := `data: {"id":"chat-1","choices":[{"delta":{},"finish_reason":"stop"}]}` + "\n\n"

	reader := c.ConvertSSE(io.NopCloser(strings.NewReader(input)))
	result, _ := io.ReadAll(reader)
	resultStr := string(result)

	if !strings.Contains(resultStr, AnthropicSSEMessageStartEvent) {
		t.Error("expected message_start")
	}
	if !strings.Contains(resultStr, AnthropicSSEMessageStopEvent) {
		t.Error("expected message_stop")
	}
}

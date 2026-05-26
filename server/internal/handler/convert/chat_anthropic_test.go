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

func TestOpenAIToAnthropic_ReasoningEffortHigh(t *testing.T) {
	c := &openaiToAnthropicConverter{}
	effort := OpenAIReasoningEffortHigh
	req := &OpenAIChatRequest{
		ReasoningEffort: &effort,
		MaxTokens:       intPtr(8192),
	}
	result, err := c.ConvertRequest(req)
	if err != nil {
		t.Fatal(err)
	}
	out := result.(*AnthropicRequest)

	if out.Thinking == nil {
		t.Fatal("expected Thinking to be set")
	}
	if out.Thinking.Type != AnthropicThinkingEnabled {
		t.Errorf("expected thinking type=enabled, got %q", out.Thinking.Type)
	}
	if out.Thinking.BudgetTokens < 1024 {
		t.Errorf("expected budget_tokens >= 1024, got %d", out.Thinking.BudgetTokens)
	}
	if out.OutputConfig == nil {
		t.Fatal("expected OutputConfig to be set")
	}
	if out.OutputConfig.Effort != "high" {
		t.Errorf("expected effort=high, got %q", out.OutputConfig.Effort)
	}
}

func TestOpenAIToAnthropic_ReasoningEffortNone(t *testing.T) {
	c := &openaiToAnthropicConverter{}
	effort := OpenAIReasoningEffortNone
	req := &OpenAIChatRequest{
		ReasoningEffort: &effort,
	}
	result, err := c.ConvertRequest(req)
	if err != nil {
		t.Fatal(err)
	}
	out := result.(*AnthropicRequest)

	if out.Thinking == nil {
		t.Fatal("expected Thinking to be set")
	}
	if out.Thinking.Type != AnthropicThinkingDisabled {
		t.Errorf("expected thinking type=disabled, got %q", out.Thinking.Type)
	}
	if out.OutputConfig != nil {
		t.Error("expected no OutputConfig for reasoning_effort=none")
	}
}

func TestOpenAIToAnthropic_ReasoningEffortXHigh(t *testing.T) {
	c := &openaiToAnthropicConverter{}
	effort := OpenAIReasoningEffortXHigh
	req := &OpenAIChatRequest{
		ReasoningEffort: &effort,
	}
	result, err := c.ConvertRequest(req)
	if err != nil {
		t.Fatal(err)
	}
	out := result.(*AnthropicRequest)

	if out.OutputConfig == nil {
		t.Fatal("expected OutputConfig to be set")
	}
	if out.OutputConfig.Effort != "max" {
		t.Errorf("expected effort=max (xhigh→max), got %q", out.OutputConfig.Effort)
	}
}

func TestOpenAIToAnthropic_ReasoningEffortNotSet(t *testing.T) {
	c := &openaiToAnthropicConverter{}
	req := &OpenAIChatRequest{}
	result, err := c.ConvertRequest(req)
	if err != nil {
		t.Fatal(err)
	}
	out := result.(*AnthropicRequest)

	if out.Thinking != nil {
		t.Error("expected no Thinking when reasoning_effort is not set")
	}
	if out.OutputConfig != nil {
		t.Error("expected no OutputConfig when reasoning_effort is not set")
	}
}

func TestOpenAIToAnthropic_ReasoningEffortBudgetFromMaxTokens(t *testing.T) {
	c := &openaiToAnthropicConverter{}
	effort := OpenAIReasoningEffortHigh
	req := &OpenAIChatRequest{
		ReasoningEffort: &effort,
		MaxTokens:       intPtr(2000),
	}
	result, err := c.ConvertRequest(req)
	if err != nil {
		t.Fatal(err)
	}
	out := result.(*AnthropicRequest)

	if out.Thinking.BudgetTokens < 1000 || out.Thinking.BudgetTokens > 1100 {
		t.Errorf("expected budget_tokens ~1000 (50%% of 2000), got %d", out.Thinking.BudgetTokens)
	}
}

func TestOpenAIToAnthropic_ReasoningEffortBudgetFromMaxCompletionTokens(t *testing.T) {
	c := &openaiToAnthropicConverter{}
	effort := OpenAIReasoningEffortHigh
	req := &OpenAIChatRequest{
		ReasoningEffort:     &effort,
		MaxCompletionTokens: intPtr(2000),
		MaxTokens:           intPtr(4000),
	}
	result, err := c.ConvertRequest(req)
	if err != nil {
		t.Fatal(err)
	}
	out := result.(*AnthropicRequest)

	if out.Thinking.BudgetTokens < 1550 || out.Thinking.BudgetTokens > 1650 {
		t.Errorf("expected budget_tokens ~1600 (80%% of 2000 from MaxCompletionTokens), got %d", out.Thinking.BudgetTokens)
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

func TestOpenAIToAnthropic_ConvertResponse_ThinkingContent(t *testing.T) {
	c := &openaiToAnthropicConverter{}
	body := []byte(`{
		"id": "msg_01",
		"type": "message",
		"role": "assistant",
		"model": "claude-3",
		"content": [
			{"type": "thinking", "thinking": "I need to reason step by step..."},
			{"type": "text", "text": "Here is the answer."}
		],
		"stop_reason": "end_turn"
	}`)
	out, err := c.ConvertResponse(body)
	if err != nil {
		t.Fatal(err)
	}
	var o OpenAIChatResponse
	json.Unmarshal(out, &o)

	if defaultString(o.Choices[0].Message.ReasoningContent, "") != "I need to reason step by step..." {
		t.Errorf("expected reasoning_content, got %v", o.Choices[0].Message.ReasoningContent)
	}
	if defaultString(o.Choices[0].Message.Content, "") != "Here is the answer." {
		t.Errorf("expected content='Here is the answer.', got %v", o.Choices[0].Message.Content)
	}
}

func TestOpenAIToAnthropic_ConvertResponse_ThinkingOnly(t *testing.T) {
	c := &openaiToAnthropicConverter{}
	body := []byte(`{
		"id": "msg_01",
		"type": "message",
		"role": "assistant",
		"model": "claude-3",
		"content": [
			{"type": "thinking", "thinking": "Just thinking"}
		],
		"stop_reason": "end_turn"
	}`)
	out, err := c.ConvertResponse(body)
	if err != nil {
		t.Fatal(err)
	}
	var o OpenAIChatResponse
	json.Unmarshal(out, &o)

	if defaultString(o.Choices[0].Message.ReasoningContent, "") != "Just thinking" {
		t.Errorf("expected reasoning_content='Just thinking', got %v", o.Choices[0].Message.ReasoningContent)
	}
	if defaultString(o.Choices[0].Message.Content, "") != "" {
		t.Error("expected empty content when only thinking block present")
	}
}

func TestOpenAIToAnthropic_ConvertResponse_RedactedThinking(t *testing.T) {
	c := &openaiToAnthropicConverter{}
	body := []byte(`{
		"id": "msg_01",
		"type": "message",
		"role": "assistant",
		"model": "claude-3",
		"content": [
			{"type": "redacted_thinking", "data": "encrypted_data_here"},
			{"type": "text", "text": "Final answer"}
		],
		"stop_reason": "end_turn"
	}`)
	out, err := c.ConvertResponse(body)
	if err != nil {
		t.Fatal(err)
	}
	var o OpenAIChatResponse
	json.Unmarshal(out, &o)

	if o.Choices[0].Message.ReasoningContent != nil {
		t.Error("expected no reasoning_content for redacted_thinking")
	}
	if defaultString(o.Choices[0].Message.Content, "") != "Final answer" {
		t.Errorf("expected content='Final answer', got %v", o.Choices[0].Message.Content)
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

func TestAnthropicToOpenAI_ThinkingEnabled(t *testing.T) {
	c := &anthropicToOpenAIConverter{}
	req := &AnthropicRequest{
		Thinking: &AnthropicThinkingConfigParam{
			Type:         AnthropicThinkingEnabled,
			BudgetTokens: 4096,
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

func TestAnthropicToOpenAI_ThinkingDisabled(t *testing.T) {
	c := &anthropicToOpenAIConverter{}
	req := &AnthropicRequest{
		Thinking: &AnthropicThinkingConfigParam{
			Type: AnthropicThinkingDisabled,
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
	if *out.ReasoningEffort != OpenAIReasoningEffortNone {
		t.Errorf("expected reasoning_effort=none, got %q", *out.ReasoningEffort)
	}
}

func TestAnthropicToOpenAI_OutputConfigEffort(t *testing.T) {
	c := &anthropicToOpenAIConverter{}
	req := &AnthropicRequest{
		OutputConfig: &AnthropicOutputConfig{
			Effort: "max",
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
	if *out.ReasoningEffort != OpenAIReasoningEffortXHigh {
		t.Errorf("expected reasoning_effort=xhigh (max→xhigh), got %q", *out.ReasoningEffort)
	}
}

func TestAnthropicToOpenAI_OutputConfigOverridesThinking(t *testing.T) {
	c := &anthropicToOpenAIConverter{}
	req := &AnthropicRequest{
		Thinking: &AnthropicThinkingConfigParam{
			Type: AnthropicThinkingDisabled,
		},
		OutputConfig: &AnthropicOutputConfig{
			Effort: "high",
		},
	}
	result, err := c.ConvertRequest(req)
	if err != nil {
		t.Fatal(err)
	}
	out := result.(*OpenAIChatRequest)

	if out.ReasoningEffort == nil {
		t.Fatal("expected ReasoningEffort to be set (OutputConfig overrides Thinking)")
	}
	if *out.ReasoningEffort != OpenAIReasoningEffortHigh {
		t.Errorf("expected reasoning_effort=high from OutputConfig, got %q", *out.ReasoningEffort)
	}
}

func TestAnthropicToOpenAI_NoThinking(t *testing.T) {
	c := &anthropicToOpenAIConverter{}
	result, err := c.ConvertRequest(&AnthropicRequest{})
	if err != nil {
		t.Fatal(err)
	}
	out := result.(*OpenAIChatRequest)

	if out.ReasoningEffort != nil {
		t.Error("expected no ReasoningEffort when Thinking/OutputConfig not set")
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

func TestAnthropicToOpenAI_ConvertResponse_ReasoningContent(t *testing.T) {
	c := &anthropicToOpenAIConverter{}
	body := []byte(`{
		"choices": [{
			"index": 0,
			"message": {
				"role": "assistant",
				"content": "Final answer",
				"reasoning_content": "Step by step thinking..."
			},
			"finish_reason": "stop"
		}]
	}`)
	out, err := c.ConvertResponse(body)
	if err != nil {
		t.Fatal(err)
	}
	var a AnthropicResponse
	json.Unmarshal(out, &a)

	if len(a.Content) != 2 {
		t.Fatalf("expected 2 content blocks (thinking + text), got %d", len(a.Content))
	}
	if a.Content[0].Type != AnthropicContentTypeThinking {
		t.Errorf("expected first block type=thinking, got %q", a.Content[0].Type)
	}
	if a.Content[0].Thinking != "Step by step thinking..." {
		t.Errorf("expected thinking='Step by step thinking...', got %q", a.Content[0].Thinking)
	}
	if a.Content[1].Type != AnthropicContentTypeText {
		t.Errorf("expected second block type=text, got %q", a.Content[1].Type)
	}
	if a.Content[1].Text != "Final answer" {
		t.Errorf("expected text='Final answer', got %q", a.Content[1].Text)
	}
}

func TestAnthropicToOpenAI_ConvertResponse_ReasoningOnly(t *testing.T) {
	c := &anthropicToOpenAIConverter{}
	body := []byte(`{
		"choices": [{
			"index": 0,
			"message": {
				"role": "assistant",
				"reasoning_content": "Just thinking"
			},
			"finish_reason": "stop"
		}]
	}`)
	out, err := c.ConvertResponse(body)
	if err != nil {
		t.Fatal(err)
	}
	var a AnthropicResponse
	json.Unmarshal(out, &a)

	if len(a.Content) != 1 {
		t.Fatalf("expected 1 content block (thinking only), got %d", len(a.Content))
	}
	if a.Content[0].Type != AnthropicContentTypeThinking {
		t.Errorf("expected block type=thinking, got %q", a.Content[0].Type)
	}
	if a.Content[0].Text != "" {
		t.Error("expected no text in thinking-only response")
	}
}

func TestAnthropicToOpenAI_ConvertResponse_NoReasoning(t *testing.T) {
	c := &anthropicToOpenAIConverter{}
	body := []byte(`{
		"choices": [{
			"index": 0,
			"message": {"role": "assistant", "content": "Just text"},
			"finish_reason": "stop"
		}]
	}`)
	out, err := c.ConvertResponse(body)
	if err != nil {
		t.Fatal(err)
	}
	var a AnthropicResponse
	json.Unmarshal(out, &a)

	if len(a.Content) != 1 {
		t.Fatalf("expected 1 content block (text only), got %d", len(a.Content))
	}
	if a.Content[0].Type != AnthropicContentTypeText {
		t.Errorf("expected block type=text, got %q", a.Content[0].Type)
	}
	if a.Content[0].Text != "Just text" {
		t.Errorf("expected text='Just text', got %q", a.Content[0].Text)
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

func TestConvertSSE_AnthropicToOpenAI_ThinkingDelta(t *testing.T) {
	c := &openaiToAnthropicConverter{}
	input := "event: message_start\ndata: {\"type\":\"message_start\",\"message\":{\"id\":\"msg_1\",\"model\":\"claude-3\"}}\n\n" +
		"event: content_block_delta\ndata: {\"type\":\"content_block_delta\",\"index\":0,\"delta\":{\"type\":\"thinking_delta\",\"thinking\":\"I'm thinking step by\"}}\n\n" +
		"event: content_block_delta\ndata: {\"type\":\"content_block_delta\",\"index\":0,\"delta\":{\"type\":\"thinking_delta\",\"thinking\":\" step...\"}}\n\n" +
		"event: content_block_delta\ndata: {\"type\":\"content_block_delta\",\"index\":0,\"delta\":{\"type\":\"text_delta\",\"text\":\"Here is the answer.\"}}\n\n" +
		"event: message_delta\ndata: {\"type\":\"message_delta\",\"delta\":{\"stop_reason\":\"end_turn\"}}\n\n" +
		"event: message_stop\ndata: {\"type\":\"message_stop\"}\n\n"

	reader := c.ConvertSSE(io.NopCloser(strings.NewReader(input)))
	result, _ := io.ReadAll(reader)
	resultStr := string(result)

	// Each thinking delta is a separate SSE chunk, so we check for individual pieces
	if !strings.Contains(resultStr, `"reasoning_content":"I'm thinking step by"`) {
		t.Errorf("expected first reasoning_content delta, got %s", resultStr)
	}
	if !strings.Contains(resultStr, `"reasoning_content":" step..."`) {
		t.Errorf("expected second reasoning_content delta, got %s", resultStr)
	}
	if !strings.Contains(resultStr, `"content":"Here is the answer."`) {
		t.Error("expected content text delta")
	}
}

func TestConvertSSE_AnthropicToOpenAI_ThinkingDeltaOnly(t *testing.T) {
	c := &openaiToAnthropicConverter{}
	input := "event: message_start\ndata: {\"type\":\"message_start\",\"message\":{\"id\":\"m1\",\"model\":\"c3\"}}\n\n" +
		"event: content_block_delta\ndata: {\"type\":\"content_block_delta\",\"index\":0,\"delta\":{\"type\":\"thinking_delta\",\"thinking\":\"Just thinking\"}}\n\n" +
		"event: message_delta\ndata: {\"type\":\"message_delta\",\"delta\":{\"stop_reason\":\"end_turn\"}}\n\n" +
		"event: message_stop\ndata: {\"type\":\"message_stop\"}\n\n"

	reader := c.ConvertSSE(io.NopCloser(strings.NewReader(input)))
	result, _ := io.ReadAll(reader)
	resultStr := string(result)

	if !strings.Contains(resultStr, `"reasoning_content":"Just thinking"`) {
		t.Errorf("expected reasoning_content, got %s", resultStr)
	}
}

func TestConvertSSE_AnthropicToOpenAI_EmptyThinkingDeltaSkipped(t *testing.T) {
	c := &openaiToAnthropicConverter{}
	input := "event: message_start\ndata: {\"type\":\"message_start\",\"message\":{\"id\":\"m1\",\"model\":\"c3\"}}\n\n" +
		"event: content_block_delta\ndata: {\"type\":\"content_block_delta\",\"index\":0,\"delta\":{\"type\":\"thinking_delta\",\"thinking\":\"\"}}\n\n" +
		"event: message_stop\ndata: {\"type\":\"message_stop\"}\n\n"

	reader := c.ConvertSSE(io.NopCloser(strings.NewReader(input)))
	result, _ := io.ReadAll(reader)
	resultStr := string(result)

	if strings.Contains(resultStr, `"reasoning_content"`) {
		t.Error("expected no reasoning_content for empty thinking delta")
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

func TestConvertSSE_OpenAIToAnthropic_ReasoningContent(t *testing.T) {
	c := &anthropicToOpenAIConverter{}
	input := `data: {"id":"chat-1","model":"gpt-4","choices":[{"delta":{"role":"assistant"}}]}` + "\n\n" +
		`data: {"choices":[{"delta":{"reasoning_content":"Thinking step by"}}]}` + "\n\n" +
		`data: {"choices":[{"delta":{"reasoning_content":" step..."}}]}` + "\n\n" +
		`data: {"choices":[{"delta":{"content":"Here is the answer."}}]}` + "\n\n" +
		`data: {"choices":[{"delta":{},"finish_reason":"stop"}]}` + "\n\n" +
		`data: [DONE]` + "\n\n"

	reader := c.ConvertSSE(io.NopCloser(strings.NewReader(input)))
	result, _ := io.ReadAll(reader)
	resultStr := string(result)

	if !strings.Contains(resultStr, `"type":"thinking_delta"`) {
		t.Errorf("expected thinking_delta type, got %s", resultStr)
	}
	if !strings.Contains(resultStr, `"thinking":"Thinking step by"`) {
		t.Errorf("expected first thinking delta, got %s", resultStr)
	}
	if !strings.Contains(resultStr, `"thinking":" step..."`) {
		t.Errorf("expected second thinking delta, got %s", resultStr)
	}
	if !strings.Contains(resultStr, `"type":"text_delta"`) {
		t.Errorf("expected text_delta type, got %s", resultStr)
	}
	if !strings.Contains(resultStr, `"text":"Here is the answer."`) {
		t.Errorf("expected text delta, got %s", resultStr)
	}
}

func TestConvertSSE_OpenAIToAnthropic_ReasoningWithoutContent(t *testing.T) {
	c := &anthropicToOpenAIConverter{}
	input := `data: {"id":"chat-1","choices":[{"delta":{"role":"assistant"}}]}` + "\n\n" +
		`data: {"choices":[{"delta":{"reasoning_content":"Only thinking"}}]}` + "\n\n" +
		`data: {"choices":[{"delta":{},"finish_reason":"stop"}]}` + "\n\n" +
		`data: [DONE]` + "\n\n"

	reader := c.ConvertSSE(io.NopCloser(strings.NewReader(input)))
	result, _ := io.ReadAll(reader)
	resultStr := string(result)

	if !strings.Contains(resultStr, `"type":"thinking_delta"`) {
		t.Errorf("expected thinking_delta, got %s", resultStr)
	}
	if strings.Contains(resultStr, `"type":"text_delta"`) {
		t.Error("expected no text_delta when only reasoning_content is present")
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

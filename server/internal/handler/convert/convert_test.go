package convert

import (
	"bytes"
	"encoding/json"
	"io"
	"strings"
	"testing"
)

// ============================================================
// copyMap
// ============================================================

func TestCopyMap(t *testing.T) {
	src := map[string]interface{}{
		"a": 1,
		"b": "two",
		"c": true,
	}
	out := copyMap(src, "b")
	if out["a"] != 1 {
		t.Error("expected a=1")
	}
	if out["b"] != nil {
		t.Error("expected b to be omitted")
	}
	if out["c"] != true {
		t.Error("expected c=true")
	}
}

func TestCopyMapEmptyOmit(t *testing.T) {
	src := map[string]interface{}{"a": 1}
	out := copyMap(src)
	if len(out) != 1 || out["a"] != 1 {
		t.Error("expected passthrough")
	}
}

func TestCopyMapEmptySrc(t *testing.T) {
	out := copyMap(map[string]interface{}{}, "a")
	if len(out) != 0 {
		t.Error("expected empty")
	}
}

// ============================================================
// extractString
// ============================================================

func TestExtractString(t *testing.T) {
	m := map[string]interface{}{
		"msg": map[string]interface{}{
			"id": "abc123",
		},
	}
	if s := extractString(m, "msg", "id"); s != "abc123" {
		t.Errorf("expected abc123, got %q", s)
	}
}

func TestExtractStringMissingKey(t *testing.T) {
	m := map[string]interface{}{"a": "1"}
	if s := extractString(m, "a", "b"); s != "" {
		t.Errorf("expected empty, got %q", s)
	}
}

func TestExtractStringNotAMap(t *testing.T) {
	m := map[string]interface{}{"a": "string"}
	if s := extractString(m, "a", "b"); s != "" {
		t.Errorf("expected empty, got %q", s)
	}
}

// ============================================================
// GetConverter
// ============================================================

func TestGetConverterOpenAIToAnthropic(t *testing.T) {
	c := GetConverter(ProtocolOpenAI, "anthropic")
	if _, ok := c.(*openaiToAnthropicConverter); !ok {
		t.Errorf("expected openaiToAnthropicConverter, got %T", c)
	}
}

func TestGetConverterAnthropicToOpenAI(t *testing.T) {
	c := GetConverter(ProtocolAnthropic, "openai")
	if _, ok := c.(*anthropicToOpenAIConverter); !ok {
		t.Errorf("expected anthropicToOpenAIConverter, got %T", c)
	}
}

func TestGetConverterSameProtocol(t *testing.T) {
	c := GetConverter(ProtocolOpenAI, "openai")
	if _, ok := c.(*noopConverter); !ok {
		t.Errorf("expected noopConverter, got %T", c)
	}
	c2 := GetConverter(ProtocolAnthropic, "anthropic")
	if _, ok := c2.(*noopConverter); !ok {
		t.Errorf("expected noopConverter, got %T", c2)
	}
}

// ============================================================
// noopConverter
// ============================================================

func TestNoopConverter(t *testing.T) {
	c := &noopConverter{}
	req := map[string]interface{}{"model": "gpt-4"}
	out := c.ConvertRequest(req)
	if out["model"] != "gpt-4" {
		t.Error("expected passthrough request")
	}

	body := []byte(`{"ok":true}`)
	outBody, err := c.ConvertResponse(body)
	if err != nil || !bytes.Equal(outBody, body) {
		t.Error("expected passthrough response")
	}

	r := io.NopCloser(strings.NewReader("test"))
	outReader := c.ConvertSSE(r)
	if outReader != r {
		t.Error("expected passthrough SSE")
	}
}

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
// SSE: WriteSSE
// ============================================================

func TestWriteSSE_WithEvent(t *testing.T) {
	var buf bytes.Buffer
	err := WriteSSE(&buf, SSEEvent{Event: "message_start", Data: `{"type":"message_start"}`})
	if err != nil {
		t.Fatal(err)
	}
	expected := "event: message_start\ndata: {\"type\":\"message_start\"}\n\n"
	if buf.String() != expected {
		t.Errorf("expected %q, got %q", expected, buf.String())
	}
}

func TestWriteSSE_NoEvent(t *testing.T) {
	var buf bytes.Buffer
	err := WriteSSE(&buf, SSEEvent{Data: `{"content":"hi"}`})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "data: ") {
		t.Error("expected data: prefix")
	}
	if strings.Contains(buf.String(), "event: ") {
		t.Error("expected no event: prefix")
	}
}

func TestWriteSSE_Done(t *testing.T) {
	var buf bytes.Buffer
	err := WriteSSE(&buf, SSEEvent{Data: "[DONE]"})
	if err != nil {
		t.Fatal(err)
	}
	if buf.String() != "data: [DONE]\n\n" {
		t.Errorf("expected data: [DONE], got %q", buf.String())
	}
}

func TestWriteSSE_EmptyEvent(t *testing.T) {
	var buf bytes.Buffer
	err := WriteSSE(&buf, SSEEvent{Event: "", Data: "{}"})
	if err != nil {
		t.Fatal(err)
	}
	if buf.String() != "data: {}\n\n" {
		t.Errorf("expected simple data, got %q", buf.String())
	}
}

// ============================================================
// SSE: ParseSSE
// ============================================================

func TestParseSSE_BasicEvent(t *testing.T) {
	input := "event: message_start\ndata: {\"type\":\"msg\"}\n\n"
	r := strings.NewReader(input)
	events, errs := ParseSSE(r)

	var evt SSEEvent
	select {
	case evt = <-events:
	case <-errs:
		t.Fatal("unexpected error")
	}

	if evt.Event != "message_start" {
		t.Errorf("expected message_start, got %q", evt.Event)
	}
	if evt.Data != `{"type":"msg"}` {
		t.Errorf("got data %q", evt.Data)
	}
}

func TestParseSSE_MultipleEvents(t *testing.T) {
	input := "data: first\n\ndata: second\n\n"
	r := strings.NewReader(input)
	events, _ := ParseSSE(r)

	e1 := <-events
	if e1.Data != "first" {
		t.Errorf("expected first, got %q", e1.Data)
	}
	e2 := <-events
	if e2.Data != "second" {
		t.Errorf("expected second, got %q", e2.Data)
	}
}

func TestParseSSE_MultiLineData(t *testing.T) {
	input := "data: line1\ndata: line2\n\n"
	r := strings.NewReader(input)
	events, _ := ParseSSE(r)

	e := <-events
	if e.Data != "line1\nline2" {
		t.Errorf("expected joined lines, got %q", e.Data)
	}
}

func TestParseSSE_CommentIgnored(t *testing.T) {
	input := ": comment line\ndata: real\n\n"
	r := strings.NewReader(input)
	events, _ := ParseSSE(r)

	e := <-events
	if e.Data != "real" {
		t.Errorf("expected real, got %q", e.Data)
	}
}

func TestParseSSE_EmptyInput(t *testing.T) {
	events, errs := ParseSSE(strings.NewReader(""))
	_, ok := <-events
	if ok {
		t.Error("expected closed events channel")
	}
	_, ok = <-errs
	if ok {
		t.Error("expected closed errs channel")
	}
}

func TestParseSSE_DoneEvent(t *testing.T) {
	input := "data: [DONE]\n\n"
	r := strings.NewReader(input)
	events, _ := ParseSSE(r)

	e := <-events
	if e.Data != "[DONE]" {
		t.Errorf("expected [DONE], got %q", e.Data)
	}
}

func TestParseSSE_OpenAIFormat(t *testing.T) {
	input := `data: {"id":"1","choices":[{"delta":{"content":"hi"}}]}` + "\n\n"
	r := strings.NewReader(input)
	events, _ := ParseSSE(r)

	e := <-events
	if e.Event != "" {
		t.Error("expected empty event for OpenAI format")
	}
	if e.Data != `{"id":"1","choices":[{"delta":{"content":"hi"}}]}` {
		t.Errorf("got data %q", e.Data)
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
// Helper: writeSSEJSON
// ============================================================

func TestWriteSSEJSON(t *testing.T) {
	var buf bytes.Buffer
	err := writeSSEJSON(&buf, "my_event", map[string]string{"key": "val"})
	if err != nil {
		t.Fatal(err)
	}
	result := buf.String()
	if !strings.Contains(result, "event: my_event") {
		t.Error("expected event: my_event")
	}
	if !strings.Contains(result, `"key":"val"`) {
		t.Error("expected JSON payload")
	}
}

// ============================================================
// Edge: ConvertResponse with invalid JSON
// ============================================================

func TestConvertResponse_InvalidJSON(t *testing.T) {
	c := &openaiToAnthropicConverter{}
	_, err := c.ConvertResponse([]byte("not-json"))
	if err == nil {
		t.Error("expected error for invalid JSON")
	}

	c2 := &anthropicToOpenAIConverter{}
	_, err = c2.ConvertResponse([]byte("not-json"))
	if err == nil {
		t.Error("expected error for invalid JSON")
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
// Round-trip: request stays valid JSON after conversion
// ============================================================

func TestRoundTrip_RequestJSON(t *testing.T) {
	oaiReq := map[string]interface{}{
		"model": "gpt-4",
		"messages": []interface{}{
			map[string]interface{}{"role": "system", "content": "You are helpful."},
			map[string]interface{}{"role": "user", "content": "Hello"},
		},
		"stop": "END",
	}

	c := &openaiToAnthropicConverter{}
	converted := c.ConvertRequest(oaiReq)

	// Verify result is marshalable as JSON
	data, err := json.Marshal(converted)
	if err != nil {
		t.Fatalf("converted request is not valid JSON: %v", err)
	}

	// Unmarshal back and check fields
	var back map[string]interface{}
	json.Unmarshal(data, &back)

	if _, ok := back["system"]; !ok {
		t.Error("expected system field in converted request")
	}
	if _, ok := back["max_tokens"]; !ok {
		t.Error("expected max_tokens default")
	}
	if _, ok := back["stop_sequences"]; !ok {
		t.Error("expected stop_sequences")
	}
	if _, ok := back["stop"]; ok {
		t.Error("expected stop to be removed")
	}
}

func TestRoundTrip_ResponseJSON(t *testing.T) {
	anthResp := []byte(`{"id":"1","type":"message","role":"assistant","model":"c3","content":[{"type":"text","text":"Hi"}],"stop_reason":"end_turn","usage":{"input_tokens":5,"output_tokens":10}}`)

	c := &openaiToAnthropicConverter{}
	out, err := c.ConvertResponse(anthResp)
	if err != nil {
		t.Fatal(err)
	}

	var oai map[string]interface{}
	if err := json.Unmarshal(out, &oai); err != nil {
		t.Fatalf("converted response is not valid JSON: %v", err)
	}
	if oai["object"] != "chat.completion" {
		t.Error("expected object")
	}
	choices := oai["choices"].([]interface{})
	choice := choices[0].(map[string]interface{})
	msg := choice["message"].(map[string]interface{})
	if msg["role"] != "assistant" {
		t.Error("expected assistant role")
	}
	if msg["content"] != "Hi" {
		t.Error("expected Hi content")
	}
}

// ============================================================
// GetConverter — new protocol combinations
// ============================================================

func TestGetConverterResponsesToOpenAI(t *testing.T) {
	c := GetConverter(ProtocolOpenAIResponses, "openai")
	if _, ok := c.(*responsesToOpenAIConverter); !ok {
		t.Errorf("expected responsesToOpenAIConverter, got %T", c)
	}
}

func TestGetConverterOpenAIToResponses(t *testing.T) {
	c := GetConverter(ProtocolOpenAI, "openai_responses")
	if _, ok := c.(*openaiToResponsesConverter); !ok {
		t.Errorf("expected openaiToResponsesConverter, got %T", c)
	}
}

func TestGetConverterResponsesToAnthropic(t *testing.T) {
	c := GetConverter(ProtocolOpenAIResponses, "anthropic")
	if _, ok := c.(*responsesToAnthropicConverter); !ok {
		t.Errorf("expected responsesToAnthropicConverter, got %T", c)
	}
}

func TestGetConverterAnthropicToResponses(t *testing.T) {
	c := GetConverter(ProtocolAnthropic, "openai_responses")
	if _, ok := c.(*anthropicToResponsesConverter); !ok {
		t.Errorf("expected anthropicToResponsesConverter, got %T", c)
	}
}

func TestGetConverterResponsesSameProtocol(t *testing.T) {
	c := GetConverter(ProtocolOpenAIResponses, "openai_responses")
	if _, ok := c.(*noopConverter); !ok {
		t.Errorf("expected noopConverter, got %T", c)
	}
}

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

// ============================================================
// Edge: ConvertResponse invalid JSON for new converters
// ============================================================

func TestConvertResponse_InvalidJSON_NewConverters(t *testing.T) {
	c1 := &responsesToOpenAIConverter{}
	_, err := c1.ConvertResponse([]byte("not-json"))
	if err == nil {
		t.Error("expected error for invalid JSON")
	}

	c2 := &openaiToResponsesConverter{}
	_, err = c2.ConvertResponse([]byte("not-json"))
	if err == nil {
		t.Error("expected error for invalid JSON")
	}

	c3 := &responsesToAnthropicConverter{}
	_, err = c3.ConvertResponse([]byte("not-json"))
	if err == nil {
		t.Error("expected error for invalid JSON")
	}

	c4 := &anthropicToResponsesConverter{}
	_, err = c4.ConvertResponse([]byte("not-json"))
	if err == nil {
		t.Error("expected error for invalid JSON")
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

package convert

import (
	"testing"
	"bytes"
	"encoding/json"
	"io"
	"strings"
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
// toFloat64
// ============================================================

func TestToFloat64_Int(t *testing.T) {
	if v := toFloat64(int(42)); v != 42.0 {
		t.Errorf("expected 42.0, got %v", v)
	}
}

func TestToFloat64_Int64(t *testing.T) {
	if v := toFloat64(int64(99)); v != 99.0 {
		t.Errorf("expected 99.0, got %v", v)
	}
}

func TestToFloat64_Default(t *testing.T) {
	if v := toFloat64("not-a-number"); v != 0 {
		t.Errorf("expected 0 for non-number, got %v", v)
	}
}

// ============================================================
// extractString edge cases
// ============================================================

func TestExtractString_NonStringValue(t *testing.T) {
	m := map[string]interface{}{
		"a": map[string]interface{}{"b": 42},
	}
	if s := extractString(m, "a", "b"); s != "" {
		t.Errorf("expected empty for non-string, got %q", s)
	}
}

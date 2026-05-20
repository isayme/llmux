package convert

import (
	"encoding/json"
	"io"

	"llmux/internal/constant"
)

type UsedAIProtocol int

const (
	ProtocolOpenAI UsedAIProtocol = iota
	ProtocolAnthropic
)

func ConvertRequestBody(usedProtocol UsedAIProtocol, providerType string, req map[string]interface{}) map[string]interface{} {
	if usedProtocol == ProtocolOpenAI && providerType == constant.ProviderTypeAnthropic {
		return convertOpenAIToAnthropic(req)
	}
	if usedProtocol == ProtocolAnthropic && providerType == constant.ProviderTypeOpenAI {
		return convertAnthropicToOpenAI(req)
	}
	return req
}

func ConvertResponseBody(usedProtocol UsedAIProtocol, providerType string, body []byte) ([]byte, error) {
	if usedProtocol == ProtocolOpenAI && providerType == constant.ProviderTypeAnthropic {
		return convertAnthropicResponseToOpenAI(body)
	}
	if usedProtocol == ProtocolAnthropic && providerType == constant.ProviderTypeOpenAI {
		return convertOpenAIResponseToAnthropic(body)
	}
	return body, nil
}

func ConvertResponseStream(direction UsedAIProtocol, providerType string, body io.ReadCloser) io.ReadCloser {
	if direction == ProtocolOpenAI && providerType == constant.ProviderTypeAnthropic {
		return convertAnthropicSSEToOpenAI(body)
	}
	if direction == ProtocolAnthropic && providerType == constant.ProviderTypeOpenAI {
		return convertOpenAISSEToAnthropic(body)
	}
	return body
}

// --- OpenAI request → Anthropic request ---

func convertOpenAIToAnthropic(req map[string]interface{}) map[string]interface{} {
	out := copyMap(req,
		"messages",   // extracted, system role removed
		"stop",       // renamed to stop_sequences
		"max_tokens", // required by Anthropic, defaults to 4096
	)

	out = extractSystemFromMessages(out, req)
	out = convertStopToStopSequences(out, req)
	out = ensureMaxTokens(out, req)
	return out
}

// extractSystemFromMessages removes system-role messages from the messages array
// and places their content into the Anthropic top-level "system" field.
// OpenAI represents system prompts as messages with role="system"; Anthropic uses
// a dedicated "system" field (string or content block array).
func extractSystemFromMessages(out, req map[string]interface{}) map[string]interface{} {
	messages, ok := req["messages"].([]interface{})
	if !ok {
		return out
	}

	var nonSystem []interface{}
	var systemContents []string

	for _, m := range messages {
		msg, ok := m.(map[string]interface{})
		if !ok {
			nonSystem = append(nonSystem, m)
			continue
		}
		if role, _ := msg["role"].(string); role != "system" {
			nonSystem = append(nonSystem, m)
			continue
		}
		// Collect system message text from both string and content-block formats.
		if content, _ := msg["content"].(string); content != "" {
			systemContents = append(systemContents, content)
		}
		if content, ok := msg["content"].([]interface{}); ok {
			for _, c := range content {
				if cm, ok := c.(map[string]interface{}); ok {
					if text, _ := cm["text"].(string); text != "" {
						systemContents = append(systemContents, text)
					}
				}
			}
		}
	}

	out["messages"] = nonSystem

	if len(systemContents) > 0 {
		systemStr := systemContents[0]
		for i := 1; i < len(systemContents); i++ {
			systemStr += "\n\n" + systemContents[i]
		}
		out["system"] = systemStr
	}
	return out
}

// convertStopToStopSequences renames OpenAI's "stop" field to Anthropic's
// "stop_sequences". OpenAI accepts both a single string or an array;
// Anthropic requires an array.
func convertStopToStopSequences(out, req map[string]interface{}) map[string]interface{} {
	stopVal, ok := req["stop"]
	if !ok {
		return out
	}
	switch v := stopVal.(type) {
	case string:
		out["stop_sequences"] = []string{v}
	case []interface{}:
		seqs := make([]string, 0, len(v))
		for _, s := range v {
			if str, ok := s.(string); ok {
				seqs = append(seqs, str)
			}
		}
		out["stop_sequences"] = seqs
	}
	return out
}

// ensureMaxTokens guarantees "max_tokens" is present.
// Anthropic requires this field; OpenAI does not. Default to 4096 when missing.
func ensureMaxTokens(out, req map[string]interface{}) map[string]interface{} {
	if req["max_tokens"] != nil {
		out["max_tokens"] = req["max_tokens"]
	} else {
		out["max_tokens"] = 4096
	}
	return out
}

// --- Anthropic request → OpenAI request ---

func convertAnthropicToOpenAI(req map[string]interface{}) map[string]interface{} {
	out := copyMap(req,
		"system",         // moved into messages array
		"stop_sequences", // renamed to stop
	)

	out = injectSystemIntoMessages(out, req)
	out = convertStopSequencesToStop(out, req)
	return out
}

// injectSystemIntoMessages moves Anthropic's top-level "system" field into the
// OpenAI messages array as a message with role="system".
// Anthropic uses a dedicated "system" field; OpenAI expects it inline.
func injectSystemIntoMessages(out, req map[string]interface{}) map[string]interface{} {
	system, ok := req["system"]
	if !ok {
		return out
	}

	var systemMsg map[string]interface{}

	switch s := system.(type) {
	case string:
		if s != "" {
			systemMsg = map[string]interface{}{"role": "system", "content": s}
		}
	case []interface{}:
		// Anthropic system can be an array of text content blocks.
		var texts []string
		for _, block := range s {
			if bm, ok := block.(map[string]interface{}); ok {
				if text, _ := bm["text"].(string); text != "" {
					texts = append(texts, text)
				}
			}
		}
		if len(texts) > 0 {
			content := texts[0]
			for i := 1; i < len(texts); i++ {
				content += "\n\n" + texts[i]
			}
			systemMsg = map[string]interface{}{"role": "system", "content": content}
		}
	}

	if systemMsg != nil {
		messages, _ := req["messages"].([]interface{})
		out["messages"] = append([]interface{}{systemMsg}, messages...)
	}
	return out
}

// convertStopSequencesToStop renames Anthropic's "stop_sequences" ([]string)
// to OpenAI's "stop" field.
func convertStopSequencesToStop(out, req map[string]interface{}) map[string]interface{} {
	seqs, ok := req["stop_sequences"].([]interface{})
	if !ok {
		return out
	}
	strs := make([]string, 0, len(seqs))
	for _, s := range seqs {
		if str, ok := s.(string); ok {
			strs = append(strs, str)
		}
	}
	out["stop"] = strs
	return out
}

// --- Anthropic response → OpenAI response ---

func convertAnthropicResponseToOpenAI(body []byte) ([]byte, error) {
	var a map[string]interface{}
	if err := json.Unmarshal(body, &a); err != nil {
		return nil, err
	}

	out := copyMap(a,
		"type",        // replaced with object: "chat.completion"
		"content",     // restructured into choices[0].message
		"stop_reason", // renamed to choices[0].finish_reason
		"usage",       // field names differ
		"role",        // added explicitly below
	)

	out["object"] = "chat.completion"
	content := extractTextFromContentBlocks(a)
	sr, _ := a["stop_reason"].(string)
	finishReason := mapAnthropicStopReason(sr)
	usage := remapUsageToOpenAI(a)

	out["choices"] = []map[string]interface{}{
		{
			"index": 0,
			"message": map[string]interface{}{
				"role":    "assistant",
				"content": content,
			},
			"finish_reason": finishReason,
		},
	}
	out["usage"] = usage

	return json.Marshal(out)
}

// extractTextFromContentBlocks extracts the text from Anthropic's content[0] block.
// Anthropic returns content as [{type: "text", text: "..."}];
// OpenAI uses choices[0].message.content as a plain string.
func extractTextFromContentBlocks(a map[string]interface{}) string {
	contentArr, _ := a["content"].([]interface{})
	if len(contentArr) == 0 {
		return ""
	}
	block, ok := contentArr[0].(map[string]interface{})
	if !ok {
		return ""
	}
	text, _ := block["text"].(string)
	return text
}

// mapAnthropicStopReason converts Anthropic's stop_reason to OpenAI's finish_reason.
// end_turn → stop, max_tokens → length, stop_sequence → stop.
func mapAnthropicStopReason(stopReason string) string {
	switch stopReason {
	case "end_turn":
		return "stop"
	case "max_tokens":
		return "length"
	case "stop_sequence":
		return "stop"
	default:
		return "stop"
	}
}

// remapUsageToOpenAI converts Anthropic usage (input_tokens, output_tokens) to
// OpenAI usage (prompt_tokens, completion_tokens, total_tokens).
func remapUsageToOpenAI(a map[string]interface{}) map[string]interface{} {
	u, ok := a["usage"].(map[string]interface{})
	if !ok {
		return nil
	}
	input, _ := u["input_tokens"].(float64)
	output, _ := u["output_tokens"].(float64)
	return map[string]interface{}{
		"prompt_tokens":     int(input),
		"completion_tokens": int(output),
		"total_tokens":      int(input) + int(output),
	}
}

// --- OpenAI response → Anthropic response ---

func convertOpenAIResponseToAnthropic(body []byte) ([]byte, error) {
	var o map[string]interface{}
	if err := json.Unmarshal(body, &o); err != nil {
		return nil, err
	}

	out := copyMap(o,
		"object",  // replaced with type: "message"
		"choices", // restructured into content[]
		"usage",   // field names differ
	)

	out["type"] = "message"
	out["role"] = "assistant"
	content, finishReason := extractContentAndFinishReason(o)
	stopReason := mapOpenAIFinishReason(finishReason)
	usage := remapUsageToAnthropic(o)

	out["content"] = []map[string]interface{}{
		{"type": "text", "text": content},
	}
	out["stop_reason"] = stopReason
	out["usage"] = usage

	return json.Marshal(out)
}

// extractContentAndFinishReason pulls the assistant message content and
// finish_reason from OpenAI's choices[0].
// OpenAI nests these in choices[0].message and choices[0].finish_reason.
func extractContentAndFinishReason(o map[string]interface{}) (content string, finishReason string) {
	choices, _ := o["choices"].([]interface{})
	if len(choices) == 0 {
		return "", ""
	}
	choice, ok := choices[0].(map[string]interface{})
	if !ok {
		return "", ""
	}
	if msg, ok := choice["message"].(map[string]interface{}); ok {
		content, _ = msg["content"].(string)
	}
	finishReason, _ = choice["finish_reason"].(string)
	return
}

// mapOpenAIFinishReason converts OpenAI's finish_reason to Anthropic's stop_reason.
// stop → end_turn, length → max_tokens.
func mapOpenAIFinishReason(finishReason string) string {
	switch finishReason {
	case "stop":
		return "end_turn"
	case "length":
		return "max_tokens"
	default:
		return "end_turn"
	}
}

// remapUsageToAnthropic converts OpenAI usage (prompt_tokens, completion_tokens)
// to Anthropic usage (input_tokens, output_tokens).
func remapUsageToAnthropic(o map[string]interface{}) map[string]interface{} {
	u, ok := o["usage"].(map[string]interface{})
	if !ok {
		return nil
	}
	prompt, _ := u["prompt_tokens"].(float64)
	completion, _ := u["completion_tokens"].(float64)
	return map[string]interface{}{
		"input_tokens":  int(prompt),
		"output_tokens": int(completion),
	}
}

// copyMap creates a shallow copy of src, omitting the given keys.
func copyMap(src map[string]interface{}, omit ...string) map[string]interface{} {
	out := make(map[string]interface{})
	skip := make(map[string]bool, len(omit))
	for _, k := range omit {
		skip[k] = true
	}
	for k, v := range src {
		if skip[k] {
			continue
		}
		out[k] = v
	}
	return out
}

package convert

const (
	AnthropicSSEMessageStartEvent      = "message_start"
	AnthropicSSEContentBlockStartEvent = "content_block_start"
	AnthropicSSEContentBlockDeltaEvent = "content_block_delta"
	AnthropicSSEContentBlockStopEvent  = "content_block_stop"
	AnthropicSSEMessageDeltaEvent      = "message_delta"
	AnthropicSSEMessageStopEvent       = "message_stop"
	AnthropicSSEPingEvent              = "ping"
)

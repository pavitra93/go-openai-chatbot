package send_receive

import (
	client_mcp "github.com/pavitra93/11-openai-chats/external/clients/mcp"
	client_openai "github.com/pavitra93/11-openai-chats/external/clients/openai"
)

func NewSenderRecieverStrategy(kind string, openaiConfig *client_openai.OpenAIConfig, mcpManager *client_mcp.Manager) SendAndRecieveOpenAIStrategy {
	switch kind {
	case "once":
		return NewOnceStrategy(openaiConfig, mcpManager)
	default:
		return NewStreamStrategy(openaiConfig, mcpManager)
	}
}

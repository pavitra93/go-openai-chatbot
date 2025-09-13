package openaiapi

import (
	singletons "github.com/pavitra93/11-openai-chats/external/clients"
	"github.com/pavitra93/11-openai-chats/internal/strategy"
)

func NewSenderRecieverStrategy(kind string, config *singletons.OpenAIConfig) strategy.SendAndRecieveOpenAIStrategy {
	switch kind {
	case "once":
		return NewStrategyOnce(config)
	default:
		return NewStreamStrategy(config)
	}
}

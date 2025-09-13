package clients

import "github.com/openai/openai-go/v2"

type OpenAIConfig struct {
	OpenAPIClient *openai.Client
	MaxTokens     int64
	Temperature   float64
	SystemMessage string
	History       *openai.ChatCompletionNewParams
	AllowHistory  bool
	HistorySize   int
}

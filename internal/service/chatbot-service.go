package service

import "github.com/openai/openai-go/v2"

type ChatbotService struct {
	OpenAPIClient *openai.Client
	MaxTokens     int64
	Temperature   float64
	SystemMessage string
	AllowHistory  bool
	History       *openai.ChatCompletionNewParams
	HistorySize   int
}

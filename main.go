package main

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"

	"github.com/joho/godotenv"
	"github.com/openai/openai-go/v2"
	singletons "github.com/pavitra93/11-openai-chats/external/clients"
	"github.com/pavitra93/11-openai-chats/internal/openaiapi"
	"github.com/pavitra93/11-openai-chats/internal/service/chatbot"
	"github.com/pavitra93/11-openai-chats/pkg/logger"
)

func main() {
	// Load environment variables
	_ = godotenv.Load()

	// Initialize slog
	logger.SetupLogger()

	// Get environment variables
	openapiKey := os.Getenv("OPENAI_API_KEY")
	maxTokens, _ := strconv.ParseInt(os.Getenv("MAX_TOKENS"), 10, 64)
	temperature, _ := strconv.ParseFloat(os.Getenv("TEMPERATURE"), 64)
	systemMessage := os.Getenv("SYSTEM_MESSAGE")
	if openapiKey == "" || maxTokens == 0 || temperature == 0 || systemMessage == "" {
		slog.Error("Error loading one of environment variables.",
			slog.Group("error",
				slog.String("message", "Error loading environment variables."),
			))
		os.Exit(1)
	}

	// Initialize OpenAI client & set Config
	openAIServiceClient := singletons.GetOpenAIClientInstance(openapiKey)

	// Initialize OpenAI Config
	OpenaiCfg := &singletons.OpenAIConfig{
		OpenAPIClient: openAIServiceClient.OpenAIClient,
		MaxTokens:     maxTokens,
		Temperature:   temperature,
		SystemMessage: systemMessage,
		History: &openai.ChatCompletionNewParams{
			Messages: []openai.ChatCompletionMessageParamUnion{
				openai.SystemMessage(systemMessage),
			},
		},
		AllowHistory: true,
		HistorySize:  5,
	}

	// Initialize Sender Strategy as Stream or Once
	SenderStrategy := openaiapi.NewSenderRecieverStrategy("stream", OpenaiCfg)

	// Initialize Chatbot Service wi
	//fmt.Println("========Chatbot with No Memory=========")
	//NoMemoryChatbotService := &chatbot.NoMemoryChatbotService{SenderStrategy: SenderStrategy}
	//NoMemoryChatbotService.RunNoMemoryChatbot()

	fmt.Println("========Chatbot with Memory=========")
	MemoryChatbotService := &chatbot.MemoryChatbotService{SenderStrategy: SenderStrategy}
	MemoryChatbotService.RunMemoryChatbot()

}

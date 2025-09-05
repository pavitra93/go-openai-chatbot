package main

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"

	"github.com/joho/godotenv"
	singletons "github.com/pavitra93/11-openai-chats/external/clients"
	"github.com/pavitra93/11-openai-chats/internal/service"
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

	// Initialize openai client
	openAIServiceClient := singletons.GetOpenAIClientInstance(openapiKey)

	// Initialize chatbot service
	ChatbotService := &service.ChatbotService{
		OpenAPIClient: openAIServiceClient.OpenAIClient,
		MaxTokens:     maxTokens,
		Temperature:   temperature,
		SystemMessage: systemMessage,
	}

	// Initialize worker service
	WorkerService := &service.WorkerService{ChatbotService}

	//fmt.Println("========Chatbot with No Memory=========")

	//NoMemoryChatbotService := &service.NoMemoryChatbotService{ChatbotService}
	//NoMemoryChatbotService.RunNoMemoryChatbot(WorkerService)

	//fmt.Println("========Chatbot with Memory=========")
	//MemoryChatbotService := &service.MemoryChatbotService{ChatbotService}
	//MemoryChatbotService.RunMemoryChatbot(WorkerService)

	fmt.Println("========Chatbot with Streaming Memory=========")
	StreamingChatBotService := &service.StreamingMemoryChatbotService{ChatbotService}
	StreamingChatBotService.RunStreamingMemoryChatbot(WorkerService)
}

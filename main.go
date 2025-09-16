package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
	"github.com/openai/openai-go/v2"
	mcp_client "github.com/pavitra93/11-openai-chats/external/clients/mcp"
	openai_client "github.com/pavitra93/11-openai-chats/external/clients/openai"
	send_receive "github.com/pavitra93/11-openai-chats/internal/send-receive"
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
	openAIServiceClient := openai_client.GetOpenAIClientInstance(openapiKey)
	OpenaiCfg := &openai_client.OpenAIConfig{
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

	// Initialize MCP Clients & set Config
	mcpManager := mcp_client.GetManager()
	slog.Info("MCP Manager initialized")

	// Build server list and register concurrently with retries
	servers := []mcp_client.MCPServerConfig{
		{
			Name:     os.Getenv("ACCUWEATHER_MCP_NAME"),
			Endpoint: os.Getenv("ACCUWEATHER_MCP_SERVER_URL"),
		},
		{
			Name:     os.Getenv("NOTION_MCP_NAME"),
			Endpoint: os.Getenv("NOTION_MCP_SERVER_URL"),
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := mcpManager.RegisterServers(ctx, servers); err != nil {
		slog.Error("failed to register some MCP servers", "error", err)
	}

	slog.Info("MCP servers registered", "order", mcpManager.ListServersInOrder())

	// Initialize Sender Strategy as Stream or Once
	SenderStrategy := send_receive.NewSenderRecieverStrategy("once", OpenaiCfg, mcpManager)

	fmt.Println("========Chatbot with Memory=========")
	MemoryChatbotService := &chatbot.MemoryChatbotService{SenderStrategy: SenderStrategy}
	MemoryChatbotService.RunMemoryChatbot()

}

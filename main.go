package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"
	"strconv"

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
	log.Printf("MCP Manager initialized")

	// Register MCP Server for day to day common tools
	WeatherMCPServerConfig := &mcp_client.MCPServerConfig{
		Name:     os.Getenv("ACCUWEATHER_MCP_NAME"),
		Endpoint: os.Getenv("ACCUWEATHER_MCP_SERVER_URL"),
	}
	err := mcpManager.RegisterServer(context.Background(), WeatherMCPServerConfig)
	if err != nil {
		log.Printf("MCP Manager failed to register server: %v", err)
	}

	// Register MCP Server for day to day common tools
	//NotionMCPServerConfig := &mcp_client.MCPServerConfig{
	//	Name:      "notion",
	//	Transport: "stdio",
	//	Command:   "notion-mcp-server",
	//	Args:      []string{"--port", "0"},
	//}
	//err = mcpManager.RegisterServer(context.Background(), NotionMCPServerConfig)
	//if err != nil {
	//	log.Printf("MCP Manager failed to register server: %v", err)
	//}
	//log.Printf("MCP Manager registered servers: %v", mcpManager)

	// Initialize Sender Strategy as Stream or Once
	SenderStrategy := send_receive.NewSenderRecieverStrategy("once", OpenaiCfg, mcpManager)

	fmt.Println("========Chatbot with Memory=========")
	MemoryChatbotService := &chatbot.MemoryChatbotService{SenderStrategy: SenderStrategy}
	MemoryChatbotService.RunMemoryChatbot()

}

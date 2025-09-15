package send_receive

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"sync"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/openai/openai-go/v2"
	client_mcp "github.com/pavitra93/11-openai-chats/external/clients/mcp"
	client_openai "github.com/pavitra93/11-openai-chats/external/clients/openai"
)

type StrategyOnce struct {
	OpenAIConfig *client_openai.OpenAIConfig
	MCPManager   *client_mcp.Manager
}

func NewOnceStrategy(config *client_openai.OpenAIConfig, mcpManager *client_mcp.Manager) *StrategyOnce {
	return &StrategyOnce{
		OpenAIConfig: config,
		MCPManager:   mcpManager,
	}
}

func (w *StrategyOnce) SendtoOpenAI(ctx context.Context, messages <-chan string, reciever chan<- string, wg *sync.WaitGroup) {
	defer wg.Done()
	for {
		select {
		case <-ctx.Done():
			return
		case message, ok := <-messages:
			if !ok {
				return
			}

			// Construct the common params
			param := &openai.ChatCompletionNewParams{
				Model:       openai.ChatModelGPT4_1,
				Seed:        openai.Int(0),
				MaxTokens:   openai.Int(w.OpenAIConfig.MaxTokens),
				Temperature: openai.Float(w.OpenAIConfig.Temperature),
			}

			var toolCollection = make([]openai.ChatCompletionToolUnionParam, 0)
			for _, tool := range w.MCPManager.GetAllSchemas() {
				toolCollection = append(toolCollection, tool...)
			}
			param.Tools = toolCollection

			// append user message
			w.OpenAIConfig.History.Messages = append(w.OpenAIConfig.History.Messages, openai.UserMessage(message))

		iterate:

			param.Messages = w.OpenAIConfig.History.Messages

			// Send the request (use ctx)
			resp, err := w.OpenAIConfig.OpenAPIClient.Chat.Completions.New(ctx, *param)
			if err != nil {
				reciever <- "Error: " + err.Error()
				return
			}

			// safety: ensure we have at least one choice
			if len(resp.Choices) == 0 {
				reciever <- "Error: " + err.Error()
				return
			}

			choice := resp.Choices[0]
			toolCalls := choice.Message.ToolCalls
			log.Printf("toolCalls: %v", toolCalls)

			// If there are no tools calls, it's a regular assistant response.
			if len(toolCalls) == 0 {
				if len(choice.Message.Content) > 0 && w.OpenAIConfig.AllowHistory {
					// append assistant message to history
					w.OpenAIConfig.History.Messages = append(w.OpenAIConfig.History.Messages, choice.Message.ToParam())
				}

				// send messages back to channel
				reciever <- choice.Message.Content
				slog.Info("Message sent to reciever channel")
			} else {
				// **Important**: append the assistant message that *requested* the tools call
				if w.OpenAIConfig.AllowHistory {
					w.OpenAIConfig.History.Messages = append(w.OpenAIConfig.History.Messages, choice.Message.ToParam())
				}

				for _, toolCall := range toolCalls {

					// sometimes toolCall.Type may be "function" or you can check toolCall.Function != nil
					if toolCall.Type != "function" {
						continue
					}

					// 1) Parse the JSON-encoded arguments string into a map
					var args map[string]any
					err := json.Unmarshal([]byte(toolCall.Function.Arguments), &args)
					if err != nil {
						log.Printf("failed to parse tool args for %s: %v", toolCall.Function.Name, err)
						// append an error tool message back to history so API sees we responded
						if w.OpenAIConfig.AllowHistory {
							errMsg := fmt.Sprintf("error_parsing_args: %v", err)
							w.OpenAIConfig.History.Messages = append(
								w.OpenAIConfig.History.Messages,
								openai.ToolMessage(errMsg, toolCall.ID),
							)
						}
						continue
					}

					// 2) Build CallToolParams (use field names - adjust if your SDK has different fields)
					params := &mcp.CallToolParams{
						Name:      toolCall.Function.Name,
						Arguments: args,
					}
					log.Printf("Calling MCP tool %s with params: %+v", params.Name, params.Arguments)

					// 3) Call the MCP tool and check error
					respStr, err := w.MCPManager.CallTool(toolCall.ID, toolCall.Function.Name, args)
					if err != nil {
						log.Printf("CallTool error for %s: %v", params.Name, err)
						if w.OpenAIConfig.AllowHistory {
							w.OpenAIConfig.History.Messages = append(
								w.OpenAIConfig.History.Messages,
								openai.ToolMessage(fmt.Sprintf("tool_error: %v", err), toolCall.ID),
							)
						}
						continue
					}

					log.Printf("Tool Call Response for %s: %s", params.Name, respStr)

					// 5) Append tool response to conversation history (must follow the assistant message)
					if w.OpenAIConfig.AllowHistory {
						w.OpenAIConfig.History.Messages = append(
							w.OpenAIConfig.History.Messages,
							openai.ToolMessage(respStr, toolCall.ID),
						)
					}

				}

				goto iterate
			}
		}
	}
}

func (w *StrategyOnce) RecieveFromOpenAI(ctx context.Context, reciever <-chan string, done chan<- bool, wg *sync.WaitGroup) {
	defer wg.Done()
	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-reciever:
			if !ok {
				return
			}
			slog.Info("Message recieved from reciever channel", "Message", msg)
			fmt.Printf("🤖 Chatbot: %s\n", msg)
			done <- true
		}
	}
}

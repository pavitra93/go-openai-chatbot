package send_receive

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"runtime/debug"
	"strings"
	"sync"
	"time"

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

	// per-run correlation id and panic stack trace capture
	reqID := fmt.Sprintf("once-%d", time.Now().UnixNano())
	defer func() {
		if r := recover(); r != nil {
			slog.Error("SendtoOpenAI panic",
				"req", reqID,
				"recover", r,
				"stack", string(debug.Stack()),
			)
			reciever <- "Error: internal panic in SendtoOpenAI"
		}
	}()

	step := 0
	next := func() int { step++; return step }

	for {
		select {
		case <-ctx.Done():
			return
		case message, ok := <-messages:
			if !ok {
				return
			}

			slog.Info("received user message", "req", reqID, "step", next(), "message", message)

			// Construct the common params
			param := &openai.ChatCompletionNewParams{
				Model:       openai.ChatModelGPT4_1,
				Seed:        openai.Int(0),
				Temperature: openai.Float(w.OpenAIConfig.Temperature),
			}

			var toolCollection = make([]openai.ChatCompletionToolUnionParam, 0)
			for _, tool := range w.MCPManager.GetAllSchemas() {
				toolCollection = append(toolCollection, tool...)
			}
			param.Tools = toolCollection
			slog.Info("tools assembled", "req", reqID, "step", next(), "tools_count", len(toolCollection))

			// append user message
			w.OpenAIConfig.History.Messages = append(w.OpenAIConfig.History.Messages, openai.UserMessage(message))
			slog.Info("history appended user", "req", reqID, "step", next(), "history_len", len(w.OpenAIConfig.History.Messages))

		iterate:

			param.Messages = w.OpenAIConfig.History.Messages

			// Send the request (use ctx)
			slog.Info("sending completion request", "req", reqID, "step", next())
			resp, err := w.OpenAIConfig.OpenAPIClient.Chat.Completions.New(ctx, *param)
			if err != nil {
				slog.Error("completion request failed", "req", reqID, "step", step, "error", err)
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
			slog.Info("received tool calls", "req", reqID, "step", next(), "count", len(toolCalls))

			// If there are no tools calls, it's a regular assistant response.
			if len(toolCalls) == 0 {
				if len(choice.Message.Content) > 0 && w.OpenAIConfig.AllowHistory {
					// append assistant message to history
					w.OpenAIConfig.History.Messages = append(w.OpenAIConfig.History.Messages, choice.Message.ToParam())
				}

				// send messages back to channel
				reciever <- choice.Message.Content
				slog.Info("assistant message delivered", "req", reqID, "step", next())
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

					// Debug: Log the raw arguments string
					slog.Info("tool args raw", "req", reqID, "step", next(), "tool", toolCall.Function.Name, "len", len(toolCall.Function.Arguments))

					// 1) Parse the JSON-encoded arguments string into a map
					var args map[string]any
					err := json.Unmarshal([]byte(toolCall.Function.Arguments), &args)
					if err != nil {
						slog.Error("failed to parse tool args", "req", reqID, "step", step, "tool", toolCall.Function.Name, "error", err)
						slog.Info("tool args raw copy", "req", reqID, "step", step, "raw", toolCall.Function.Arguments)

						// Try to fix common JSON truncation issues
						argsStr := toolCall.Function.Arguments
						if !strings.HasSuffix(argsStr, "}") && !strings.HasSuffix(argsStr, "]") {
							slog.Warn("args appear truncated; attempting fix", "req", reqID, "step", next())

							// For Notion API calls, try to create a simpler structure
							if strings.Contains(toolCall.Function.Name, "notion") && strings.Contains(argsStr, "\"content\":\"") {
								// Extract the title and create a simple page structure
								titleStart := strings.Index(argsStr, "\"title\":[{\"text\":{\"content\":\"")
								if titleStart > 0 {
									titleStart += len("\"title\":[{\"text\":{\"content\":\"")
									titleEnd := strings.Index(argsStr[titleStart:], "\"")
									if titleEnd > 0 {
										title := argsStr[titleStart : titleStart+titleEnd]

										// Create a simplified page structure
										argsStr = fmt.Sprintf(`{"parent":{"page_id":"ca42c764-61c4-45f6-9aaf-22910ec57800"},"properties":{"title":[{"text":{"content":"%s"}}]}}`, title)
										slog.Info("created simplified args", "req", reqID, "step", next())
										err = json.Unmarshal([]byte(argsStr), &args)
										if err != nil {
											slog.Error("simplified args parse failed", "req", reqID, "step", step, "error", err)
										}
									}
								}
							} else {
								// Generic fix attempt
								lastQuote := strings.LastIndex(argsStr, "\"")
								if lastQuote > 0 {
									argsStr = argsStr[:lastQuote+1] + "]}}}"
									slog.Info("attempting generic fix parse", "req", reqID, "step", next())
									err = json.Unmarshal([]byte(argsStr), &args)
									if err != nil {
										slog.Error("generic fix parse failed", "req", reqID, "step", step, "error", err)
									}
								}
							}
						}

						if err != nil {
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
					}

					// 2) Build CallToolParams (use field names - adjust if your SDK has different fields)
					params := &mcp.CallToolParams{
						Name:      toolCall.Function.Name,
						Arguments: args,
					}
					slog.Info("calling MCP tool", "req", reqID, "step", next(), "tool", params.Name)

					// Debug: Log the parsed arguments structure
					argsBytes, _ := json.Marshal(args)
					slog.Debug("parsed args json", "req", reqID, "step", step, "json", string(argsBytes))

					// 3) Call the MCP tool and check error
					respStr, err := w.MCPManager.CallTool(toolCall.ID, toolCall.Function.Name, args)
					if err != nil {
						slog.Error("CallTool error", "req", reqID, "step", step, "tool", params.Name, "error", err)
						if w.OpenAIConfig.AllowHistory {
							w.OpenAIConfig.History.Messages = append(
								w.OpenAIConfig.History.Messages,
								openai.ToolMessage(fmt.Sprintf("tool_error: %v", err), toolCall.ID),
							)
						}
						continue
					}

					slog.Info("tool call response", "req", reqID, "step", next(), "tool", params.Name)

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

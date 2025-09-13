package openaiapi

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/openai/openai-go/v2"
	singletons "github.com/pavitra93/11-openai-chats/external/clients"
)

type StrategyOnce struct {
	OpenAIConfig *singletons.OpenAIConfig
}

func NewStrategyOnce(config *singletons.OpenAIConfig) *StrategyOnce {
	return &StrategyOnce{
		OpenAIConfig: config,
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

			// append user message
			w.OpenAIConfig.History.Messages = append(w.OpenAIConfig.History.Messages, openai.UserMessage(message))

			// send messages to OpenAI
			param := openai.ChatCompletionNewParams{
				Messages:    w.OpenAIConfig.History.Messages,
				Model:       openai.ChatModelGPT4_1,
				MaxTokens:   openai.Int(w.OpenAIConfig.MaxTokens),
				Temperature: openai.Float(w.OpenAIConfig.Temperature),
			}

			// Send the request
			resp, err := w.OpenAIConfig.OpenAPIClient.Chat.Completions.New(context.TODO(), param)
			if err != nil {
				reciever <- "Error: " + err.Error()
				return
			}

			slog.Info("Response from OpenAI", "Content", resp.Choices[0].Message.Content, "finish reason", resp.Choices[0].FinishReason)

			// Safely print the first text part if the SDK returns structured content
			if len(resp.Choices) > 0 && len(resp.Choices[0].Message.Content) > 0 && w.OpenAIConfig.AllowHistory {
				w.OpenAIConfig.History.Messages = append(w.OpenAIConfig.History.Messages, resp.Choices[0].Message.ToParam())
			}

			// send messages back to channel
			reciever <- resp.Choices[0].Message.Content
			slog.Info("Message sent to reciever channel")
		}

	}

}

func (w *StrategyOnce) RecieveFromOpenAI(ctx context.Context, messages <-chan string, done chan<- bool, wg *sync.WaitGroup) {
	defer wg.Done()
	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-messages:
			if !ok {
				return
			}
			slog.Info("Message recieved from reciever channel", "Message", msg)
			fmt.Printf("🤖 Chatbot: %s\n", msg)
			done <- true
		}
	}
}

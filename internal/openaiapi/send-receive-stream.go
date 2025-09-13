package openaiapi

import (
	"bufio"
	"context"
	"log/slog"
	"os"
	"strings"
	"sync"

	"github.com/openai/openai-go/v2"
	singletons "github.com/pavitra93/11-openai-chats/external/clients"
	"github.com/pavitra93/11-openai-chats/pkg/utils"
)

type StreamStrategy struct {
	OpenAIConfig *singletons.OpenAIConfig
}

func NewStreamStrategy(config *singletons.OpenAIConfig) *StreamStrategy {
	return &StreamStrategy{
		OpenAIConfig: config,
	}
}

func (w *StreamStrategy) SendtoOpenAI(ctx context.Context, messages <-chan string, reciever chan<- string, wg *sync.WaitGroup) {
	defer wg.Done()
	for {
		select {
		case <-ctx.Done():
			slog.Info("StreamToOpenAI: context cancelled", "err", ctx.Err())
			return
		case message, ok := <-messages:
			if !ok {
				slog.Info("StreamToOpenAI: messages channel closed")
				return
			}

			// make history window and append user message
			w.OpenAIConfig.History.Messages = utils.MakeHistoryWindow(w.OpenAIConfig.History.Messages, message, w.OpenAIConfig.HistorySize)
			w.OpenAIConfig.History.Messages = append(w.OpenAIConfig.History.Messages, openai.UserMessage(message))
			slog.Info("History window created", "History", w.OpenAIConfig.History.Messages)

			// send messages to OpenAI
			param := openai.ChatCompletionNewParams{
				Messages:    w.OpenAIConfig.History.Messages,
				Model:       openai.ChatModelGPT4_1,
				MaxTokens:   openai.Int(w.OpenAIConfig.MaxTokens),
				Temperature: openai.Float(w.OpenAIConfig.Temperature),
			}

			acc := openai.ChatCompletionAccumulator{}

			// Send the request
			stream := w.OpenAIConfig.OpenAPIClient.Chat.Completions.NewStreaming(context.TODO(), param)
			for stream.Next() {
				chunk := stream.Current()

				acc.AddChunk(chunk)

				// When this fires, the current chunk value will not contain content data
				if justCompleted, ok := acc.JustFinishedContent(); ok {
					slog.Info("Streaming Just Completed", "Message", justCompleted)
					reciever <- "stream:completed"
				}

				// It's best to use chunks after handling JustFinished events.
				// Here we print the delta of the content, if it exists.
				if len(chunk.Choices) > 0 && chunk.Choices[0].Delta.Content != "" {
					// send messages back to channel
					reciever <- chunk.Choices[0].Delta.Content
				}
			}

			if err := stream.Err(); err != nil {
				slog.Error("Error streaming response from OpenAI.",
					slog.Group("error",
						slog.String("message", err.Error()),
					))
			}

			if acc.Usage.TotalTokens > 0 {
				slog.Info("Streaming finished with usage", "Token Usage", acc.Usage.TotalTokens)
			}

			slog.Info("Response from OpenAI", "Content", acc.Choices[0].Message.Content, "finish reason", acc.Choices[0].FinishReason)

			// Safely print the first text part if the SDK returns structured content
			if len(acc.Choices[0].Message.Content) > 0 && len(acc.Choices[0].Message.Content) > 0 && w.OpenAIConfig.AllowHistory {
				w.OpenAIConfig.History.Messages = append(w.OpenAIConfig.History.Messages, acc.Choices[0].Message.ToParam())
			}

		}

	}

}

func (w *StreamStrategy) RecieveFromOpenAI(ctx context.Context, messages <-chan string, done chan<- bool, wg *sync.WaitGroup) {
	defer wg.Done()

	writer := bufio.NewWriter(os.Stdout)
	defer writer.Flush()

	const prefix = "🤖 Chatbot: "
	const startSentinel = "stream:start"
	const endSentinel = "stream:completed"

	var assembled strings.Builder
	inStream := false // are we currently streaming a chatbot reply?

	for {
		select {
		case <-ctx.Done():
			slog.Info("StreamFromOpenAI: context cancelled", "err", ctx.Err())
			// best-effort notify
			select {
			case done <- false:
			default:
			}
			return

		case msg, ok := <-messages:
			if !ok {
				// upstream closed; if we were mid-stream, finish it
				if inStream {
					// print newline to finish line
					writer.WriteString("\n")
					writer.Flush()
					// best-effort done signal
					select {
					case done <- true:
					default:
					}
				}
				slog.Info("StreamFromOpenAI: messages channel closed")
				return
			}

			// start of a new chatbot reply
			if msg == startSentinel {
				inStream = true
				assembled.Reset()
				// print prefix once for this reply
				if _, err := writer.WriteString(prefix); err != nil {
					slog.Error("StreamFromOpenAI: write error", "err", err)
				}
				_ = writer.Flush()
				continue
			}

			// end of current chatbot reply
			if msg == endSentinel {
				if inStream {
					// finish the line with newline
					if _, err := writer.WriteString("\n"); err != nil {
						slog.Error("StreamFromOpenAI: write error", "err", err)
					}
					_ = writer.Flush()

					// notify completion (non-blocking)
					select {
					case done <- true:
					default:
					}

					inStream = false
				}
				continue
			}

			// regular token: ensure we are in a stream (if not, treat as implicit start)
			if !inStream {
				// defensive: if producer didn't send start sentinel, treat this token as first token
				inStream = true
				assembled.Reset()
				if _, err := writer.WriteString(prefix); err != nil {
					slog.Error("StreamFromOpenAI: write error", "err", err)
				}
				_ = writer.Flush()
			}

			// write token and flush immediately
			if _, err := writer.WriteString(msg); err != nil {
				slog.Error("StreamFromOpenAI: write error", "err", err)
			}
			_ = writer.Flush()

			// append to assembled (no prefix duplication)
			assembled.WriteString(msg)
		}
	}
}

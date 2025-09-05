package service

import (
	"bufio"
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"sync"

	"github.com/openai/openai-go/v2"
)

type WorkerService struct {
	ChatbotService *ChatbotService
}

func (w *WorkerService) SendMessagestoOpenAI(ctx context.Context, messages <-chan []openai.ChatCompletionMessageParamUnion, reciever chan<- string, wg *sync.WaitGroup, history bool) {
	defer wg.Done()
	for {
		select {
		case <-ctx.Done():
			return
		case message, ok := <-messages:
			if !ok {
				return
			}

			// send messages to OpenAI
			param := openai.ChatCompletionNewParams{
				Messages:    message,
				Model:       openai.ChatModelGPT4_1,
				MaxTokens:   openai.Int(w.ChatbotService.MaxTokens),
				Temperature: openai.Float(w.ChatbotService.Temperature),
			}

			// Send the request
			resp, err := w.ChatbotService.OpenAPIClient.Chat.Completions.New(context.TODO(), param)
			if err != nil {
				reciever <- "Error: " + err.Error()
				return
			}

			slog.Info("Response from OpenAI", "Content", resp.Choices[0].Message.Content, "finish reason", resp.Choices[0].FinishReason)

			// Safely print the first text part if the SDK returns structured content
			if len(resp.Choices) > 0 && len(resp.Choices[0].Message.Content) > 0 && history {
				w.ChatbotService.History.Messages = append(w.ChatbotService.History.Messages, resp.Choices[0].Message.ToParam())
			}

			// send messages back to channel
			reciever <- resp.Choices[0].Message.Content
			slog.Info("Message sent to reciever channel")
		}

	}

}

func (w *WorkerService) RecieveMessagesfromOpenAI(ctx context.Context, messages <-chan string, done chan<- bool, wg *sync.WaitGroup) {
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

func (w *WorkerService) StreamToOpenAI(ctx context.Context, messages <-chan []openai.ChatCompletionMessageParamUnion, reciever chan<- string, wg *sync.WaitGroup) {
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

			// send messages to OpenAI
			param := openai.ChatCompletionNewParams{
				Messages:    message,
				Model:       openai.ChatModelGPT4_1,
				MaxTokens:   openai.Int(w.ChatbotService.MaxTokens),
				Temperature: openai.Float(w.ChatbotService.Temperature),
			}

			acc := openai.ChatCompletionAccumulator{}

			// Send the request
			stream := w.ChatbotService.OpenAPIClient.Chat.Completions.NewStreaming(context.TODO(), param)
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
			if len(acc.Choices[0].Message.Content) > 0 && len(acc.Choices[0].Message.Content) > 0 {
				w.ChatbotService.History.Messages = append(w.ChatbotService.History.Messages, acc.Choices[0].Message.ToParam())
			}

		}

	}

}

func (w *WorkerService) StreamFromOpenAI(ctx context.Context, messages <-chan string, done chan<- bool, wg *sync.WaitGroup) {
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

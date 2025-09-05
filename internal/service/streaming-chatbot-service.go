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
	"github.com/pavitra93/11-openai-chats/internal/worker"
	"github.com/pavitra93/11-openai-chats/pkg/utils"
)

type StreamingMemoryChatbotService struct {
	ChatbotService *ChatbotService
}

func (s *StreamingMemoryChatbotService) RunStreamingMemoryChatbot(worker worker.Worker) {

	// start chatbot
	fmt.Println("Hello with Streaming Memory Chatbot")

	// send and receive messages channel
	JobMessages := make(chan []openai.ChatCompletionMessageParamUnion)
	ReceiveMessages := make(chan string)

	// create a done channel
	doneChan := make(chan bool)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// create a wait group
	wg := &sync.WaitGroup{}

	wg.Add(2)

	// allow history
	s.ChatbotService.AllowHistory = true

	// set history size
	s.ChatbotService.HistorySize = 5

	// initialize history
	s.ChatbotService.History = &openai.ChatCompletionNewParams{
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.SystemMessage(s.ChatbotService.SystemMessage),
		},
	}

	// start a goroutine to send and receive messages from OpenAI
	go worker.StreamToOpenAI(ctx, JobMessages, ReceiveMessages, wg)
	go worker.StreamFromOpenAI(ctx, ReceiveMessages, doneChan, wg)

	// initialize reader
	reader := bufio.NewReader(os.Stdin)

	// start chat loop
	for {
		dispatched := false
		fmt.Print("🧔🏻‍♂️ You: ")
		userMessage, _ := reader.ReadString('\n')
		userMessage = strings.TrimSpace(userMessage)
		slog.Info(userMessage)

		// handle exit, quit and bye
		switch userMessage {
		case "", " ":
			fmt.Println("Please type your message")
			continue
		case "exit", "quit", "bye":
			fmt.Println("Bye. Thanks for chatting with me.")
			// cancel context
			cancel()

			// stop goroutines
			close(JobMessages)

			// wait for goroutines to finish
			wg.Wait()

			// close channels
			close(ReceiveMessages)
			slog.Info("Chat explicitly stopped by user")
			return
		default:
			// make a history window and append a user message
			s.ChatbotService.History.Messages = utils.MakeHistoryWindow(s.ChatbotService.History.Messages, userMessage, s.ChatbotService.HistorySize)
			s.ChatbotService.History.Messages = append(s.ChatbotService.History.Messages, openai.UserMessage(userMessage))
			JobMessages <- s.ChatbotService.History.Messages
			slog.Info("Message sent to sender channel")
			dispatched = true
			fmt.Println("Bot is thinking...💭")

		}

		if dispatched {
			select {
			case <-doneChan:
				continue
			case <-ctx.Done():
				return
			}
		}

	}

}

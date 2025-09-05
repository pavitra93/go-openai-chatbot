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
)

type NoMemoryChatbotService struct {
	ChatbotService *ChatbotService
}

func (n *NoMemoryChatbotService) RunNoMemoryChatbot(worker worker.Worker) {

	// start chatbot
	fmt.Println("Hello with Memory Chatbot")

	// send and recieve messages channel
	JobMessages := make(chan []openai.ChatCompletionMessageParamUnion)
	ReceiveMessages := make(chan string)

	// create done channel
	doneChan := make(chan bool)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// create wait group
	wg := &sync.WaitGroup{}

	wg.Add(2)

	// start goroutine to send & recieve messages from OpenAI
	go worker.SendMessagestoOpenAI(ctx, JobMessages, ReceiveMessages, wg, n.ChatbotService.AllowHistory)
	go worker.RecieveMessagesfromOpenAI(ctx, ReceiveMessages, doneChan, wg)

	// initialize reader
	reader := bufio.NewReader(os.Stdin)

	// start chat loop
	for {
		// initialize openai chat
		n.ChatbotService.History = &openai.ChatCompletionNewParams{
			Messages: []openai.ChatCompletionMessageParamUnion{
				openai.SystemMessage(n.ChatbotService.SystemMessage),
			},
		}

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
			// append user message
			n.ChatbotService.History.Messages = append(n.ChatbotService.History.Messages, openai.UserMessage(userMessage))
			JobMessages <- n.ChatbotService.History.Messages
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

package chatbot

import (
	"bufio"
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"sync"

	"github.com/pavitra93/11-openai-chats/internal/send-receive"
)

type MemoryChatbotService struct {
	SenderStrategy send_receive.SendAndRecieveOpenAIStrategy
}

func (m *MemoryChatbotService) RunMemoryChatbot() {

	// start chatbot
	fmt.Println("Hello with Memory Chatbot")

	// send and recieve messages channel
	JobMessages := make(chan string)
	ReceiveMessages := make(chan string)

	// create done channel
	doneChan := make(chan bool)

	// create wait group
	wg := &sync.WaitGroup{}

	wg.Add(2)

	// create context and cancel function with chatbot service
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// start goroutine to send & recieve messages from OpenAI
	go m.SenderStrategy.SendtoOpenAI(ctx, JobMessages, ReceiveMessages, wg)
	go m.SenderStrategy.RecieveFromOpenAI(ctx, ReceiveMessages, doneChan, wg)

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
			JobMessages <- userMessage
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

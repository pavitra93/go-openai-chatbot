package worker

import (
	"context"
	"sync"

	"github.com/openai/openai-go/v2"
)

type Worker interface {
	SendMessagestoOpenAI(ctx context.Context, messages <-chan []openai.ChatCompletionMessageParamUnion, receiver chan<- string, wg *sync.WaitGroup, history bool)
	RecieveMessagesfromOpenAI(ctx context.Context, messages <-chan string, done chan<- bool, wg *sync.WaitGroup)
	StreamToOpenAI(ctx context.Context, messages <-chan []openai.ChatCompletionMessageParamUnion, receiver chan<- string, wg *sync.WaitGroup)
	StreamFromOpenAI(ctx context.Context, messages <-chan string, done chan<- bool, wg *sync.WaitGroup)
}

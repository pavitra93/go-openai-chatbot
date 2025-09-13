package strategy

import (
	"context"
	"sync"
)

type SendAndRecieveOpenAIStrategy interface {
	SendtoOpenAI(ctx context.Context, messages <-chan string, receiver chan<- string, wg *sync.WaitGroup)
	RecieveFromOpenAI(ctx context.Context, messages <-chan string, done chan<- bool, wg *sync.WaitGroup)
}

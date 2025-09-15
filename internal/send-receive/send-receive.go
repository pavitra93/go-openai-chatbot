package send_receive

import (
	"context"
	"sync"
)

type SendAndRecieveOpenAIStrategy interface {
	SendtoOpenAI(ctx context.Context, messages <-chan string, reciever chan<- string, wg *sync.WaitGroup)
	RecieveFromOpenAI(ctx context.Context, reciever <-chan string, done chan<- bool, wg *sync.WaitGroup)
}

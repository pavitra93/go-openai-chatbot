package clients

import (
	"sync"

	"github.com/openai/openai-go/v2"
	"github.com/openai/openai-go/v2/option"
)

type openAIServiceClient struct {
	OpenAIClient *openai.Client
}

var openAIInstance *openAIServiceClient
var once sync.Once

func GetOpenAIClientInstance(openapiKey string) *openAIServiceClient {
	once.Do(func() {
		client := openai.NewClient(
			option.WithAPIKey(openapiKey), // defaults to os.LookupEnv("OPENAI_API_KEY")
		)
		openAIInstance = &openAIServiceClient{
			OpenAIClient: &client,
		}
	})

	return openAIInstance

}

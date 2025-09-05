# GPT-4o Chatbot Services (Go) — No Memory, Memory, and Streaming
Built Chatbot services in Go that showcase three patterns:
- No-memory chatbot: stateless request/response.
- Memory chatbot: conversational context across turns.
- Memory and streaming chatbot: token-by-token streaming to the terminal.

Designed with clean interfaces, safe concurrency (goroutines + channels), structured logging (slog), and an efficient singleton OpenAI client.
## Highlights
- Three service flavors: stateless, stateful, and streaming
- Singleton OpenAI client for efficiency and connection reuse
- Interface-first design with struct embedding to share core behavior
- Streaming flush mechanism for responsive token output
- Context-aware timeouts and cancellation
- Channel-based pipelines: producer → service → consumer
- Structured logging with slog (request IDs, stages, errors)
- Clean, testable architecture with clear seams

## Architecture at a Glance
``` text
+-------------------+       +-----------------------------+       +-----------------------+
|   User/CLI Input  |  -->  |   Chat Service Interface    |  -->  |  Terminal Consumer    |
| (stdin or caller) |       | (NoMem | Memory | Streaming)|       | (prints/streams)      |
+-------------------+       +-----------------------------+       +-----------------------+
                                   |
                                   v
                           +---------------+
                           | OpenAI Client |
                           |  (singleton)  |
                           +---------------+

Concurrency (streaming):
[user messages] -> (receiver goroutine) -> [service] -> (tokens chan) -> (consumer goroutine)
```
Core principles:
- Single external client instance (singleton) to avoid redundant connections.
- Interfaces + struct embedding for reusable behaviors.
- Channels decouple production and consumption, enabling responsive streaming.
- Flush mechanism ensures timely updates during streaming (partial buffers).

## Requirements
- Go 1.25+
- OpenAI API key with access to GPT-4o
- Network access to OpenAI API

## Installation
``` bash
git clone <your-repo-url>
cd <your-repo>
go mod download
```
## Logging and Observability
- Uses slog for structured, leveled logging.
- Adds request IDs and key lifecycle events (start, token flush, end, errors).
- Ready for integration with OpenTelemetry (traces/spans) if needed.

Debug example:
``` bash
LOG_LEVEL=debug go run ./cmd/stream --prompt "Stream a short poem."
```
## Error Handling and Resilience
- Context-aware requests: deadlines and cancellation are respected.
- Goroutine lifecycle management to prevent leaks.

## Security
- Never log secrets; redact OPENAI_API_KEY in logs.
- Consider rate limiting if exposing as a server.
- Validate inputs before sending upstream.

## Roadmap
- Integration with Tool calling OpenAI API 
- Pluggable vector-store memory (semantic recall)
- Persistent conversation sessions


## Acknowledgements
- OpenAI API
- Go community

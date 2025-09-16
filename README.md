# OpenAI ChatBot with Multi-MCP Integration

A sophisticated Go-based chatbot application that integrates OpenAI's GPT-4 with multiple Model Context Protocol (MCP) servers to provide intelligent conversations with access to various external services and APIs.

## 🌟 Features

- **OpenAI GPT-4 Integration**: Powered by OpenAI's latest GPT-4 model for intelligent conversations
- **Memory-enabled Chat**: Maintains conversation history for context-aware responses
- **Multi-MCP Integration**: Supports multiple MCP servers simultaneously for diverse functionality
- **Weather Forecasting**: Real-time weather data via AccuWeather API through MCP weather server
- **Notion Integration**: Create and manage Notion pages, databases, and content
- **Redis Integration**: Use Redis MCP for key/value storage (session, user state, chat index)
- **Structured Architecture**: Clean separation of concerns with modular design
- **Concurrent Processing**: Uses goroutines and channels for efficient message handling
- **Comprehensive Logging**: JSON-structured logging with file output
- **Environment Configuration**: Flexible configuration through environment variables
- **Transport Flexibility**: Supports both HTTP/SSE and stdio transport modes

## 🏗️ Architecture

### Project Structure

```
├── external/
│   └── clients/
│       ├── mcp/                 # MCP client manager
│       └── openai/              # OpenAI client wrapper
├── internal/
│   ├── send-receive/            # Message handling strategies
│   └── service/
│       └── chatbot/             # Chatbot service implementations
├── pkg/
│   ├── logger/                  # Logging utilities
│   └── utils/                   # Helper utilities
├── logs/                        # Application logs
├── main.go                      # Application entry point
└── go.mod                       # Go module dependencies
```

### Key Components

1. **MCP Manager**: Manages connections to multiple MCP servers and tool schemas
2. **OpenAI Client**: Singleton wrapper for OpenAI API interactions
3. **Chatbot Service**: Handles conversation flow and user interactions
4. **Send/Receive Strategies**: Implements different message handling patterns
5. **Transport Factory**: Supports multiple transport modes (HTTP/SSE, stdio)
6. **Logger**: Structured JSON logging with file output

## 🚀 Quick Start

### Prerequisites

- Go 1.25.0 or later
- OpenAI API key
- Node.js (for MCP servers)
- API keys for desired integrations:
  - AccuWeather API key (for weather)
  - Notion API token (for Notion integration)
  - Redis URL (for Redis MCP, optional)

### 1. Clone the Repository

```bash
git clone https://github.com/pavitra93/11-openai-chats.git
cd 11-openai-chats
```

### 2. Install Dependencies

```bash
go mod tidy
```

### 3. Environment Setup

Create a `.env` file in the project root:

```bash
# OpenAI Configuration
OPENAI_API_KEY=your_openai_api_key_here
MAX_TOKENS=1000
TEMPERATURE=0.7

# System prompt (use file for sensitive/long prompts)
# If SYSTEM_MESSAGE_FILE is set, it takes precedence over SYSTEM_MESSAGE
SYSTEM_MESSAGE_FILE=prompts/system_message.txt
# SYSTEM_MESSAGE="You are a helpful AI assistant..."

# MCP Server Configuration
# Weather Server
ACCUWEATHER_MCP_NAME=weather
ACCUWEATHER_MCP_SERVER_URL=http://127.0.0.1:4004/mcp
ACCUWEATHER_API_KEY=your_accuweather_api_key

# Notion Server
NOTION_MCP_NAME=notion
NOTION_MCP_SERVER_URL=http://127.0.0.1:4005/mcp
NOTION_MCP_API_KEY=your_notion_api_token

# Redis Server (optional)
# Option A: Hosted Smithery endpoint (requires a publicly reachable Redis)
REDIS_MCP_NAME=redis
REDIS_MCP_SERVER_URL=https://server.smithery.ai/@redis/mcp-redis/mcp
# Provide your cloud Redis connection string to the hosted server (cannot reach local 127.0.0.1)
# Example cloud URL: redis://:password@host:port/0
# Option B: Local supergateway (see below) -> set REDIS_MCP_SERVER_URL=http://127.0.0.1:4010/mcp
```

Notes:
- Place your system prompt in `prompts/system_message.txt` (gitignored) and point `SYSTEM_MESSAGE_FILE` to it. The app will load and trim the file contents at startup.
- If `SYSTEM_MESSAGE_FILE` is not set, the app will use `SYSTEM_MESSAGE`.

### 4. Start MCP Servers

Before running the chatbot, start the required MCP servers:

#### 4.1. Weather Server

```bash
$env:ACCUWEATHER_API_KEY = "<ACCUWEATHER_API_KEY>"

npx -y supergateway \
  --stdio "npx -y @timlukahorstmann/mcp-weather" \
  --port 4004 \
  --baseUrl http://127.0.0.1 \
  --outputTransport streamableHttp \
  --env ACCUWEATHER_API_KEY="$env:ACCUWEATHER_API_KEY"
```

#### 4.2. Notion Server

```bash 
$env:NOTION_TOKEN = "<NOTION_TOKEN>"

npx -y supergateway \
  --stdio "npx -y @notionhq/notion-mcp-server" \
  --port 4005 \
  --baseUrl http://127.0.0.1 \
  --outputTransport streamableHttp \
  --env NOTION_TOKEN="$env:NOTION_TOKEN"
```

#### 4.3. Redis Server (choose one)

- Hosted (Smithery endpoint): set `REDIS_MCP_SERVER_URL=https://server.smithery.ai/@redis/mcp-redis/mcp` and configure your cloud Redis URL for that hosted instance per provider docs.
- Local (with your local Redis):

```powershell
# Start Redis locally (Docker example)
docker run -d --name redis -p 6379:6379 redis:7

# Point MCP Redis server to local Redis
$env:REDIS_URL = "redis://127.0.0.1:6379/0"

# Expose Redis MCP over HTTP via supergateway
npx -y supergateway \
  --stdio "npx -y @redis/mcp-redis" \
  --port 4010 \
  --baseUrl http://127.0.0.1 \
  --outputTransport streamableHttp \
  --env REDIS_URL="$env:REDIS_URL"
```

Then set:
```bash
REDIS_MCP_NAME=redis
REDIS_MCP_SERVER_URL=https://server.smithery.ai/@redis/mcp-redis/mcp
```

### 5. Run the Application

```bash
go run main.go
```

## ⚡ Concurrent Multi-Server Registration

The `MCP Manager` can register multiple MCP servers concurrently with retries and track the registration order.

Example in `main.go`:

```go
servers := []mcp_client.MCPServerConfig{
    { Name: os.Getenv("ACCUWEATHER_MCP_NAME"), Endpoint: os.Getenv("ACCUWEATHER_MCP_SERVER_URL") },
    { Name: os.Getenv("NOTION_MCP_NAME"),     Endpoint: os.Getenv("NOTION_MCP_SERVER_URL") },
    { Name: os.Getenv("REDIS_MCP_NAME"),      Endpoint: os.Getenv("REDIS_MCP_SERVER_URL") },
}
ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
defer cancel()
if err := mcpManager.RegisterServers(ctx, servers); err != nil {
    slog.Error("failed to register some MCP servers", "error", err)
}
slog.Info("MCP servers registered", "order", mcpManager.ListServersInOrder())
```

## 🎯 Usage Notes for Redis MCP

- Tool names will be prefixed by the server name, e.g., `redis__get`, `redis__set`.
- Hosted Smithery servers cannot access your local 127.0.0.1; provide a publicly reachable Redis URL.
- For local development, prefer the local supergateway setup shown above.

## 🔧 Configuration

### Environment Variables

| Variable | Description | Required | Default |
|----------|-------------|----------|---------|
| `OPENAI_API_KEY` | Your OpenAI API key | Yes | - |
| `MAX_TOKENS` | Maximum tokens per response | Yes | - |
| `TEMPERATURE` | OpenAI temperature setting (0-1) | Yes | - |
| `SYSTEM_MESSAGE_FILE` | Path to file containing system prompt | Recommended | - |
| `SYSTEM_MESSAGE` | System prompt string (fallback) | Optional | - |
| `ACCUWEATHER_MCP_NAME` | Name for the MCP weather server | Yes | - |
| `ACCUWEATHER_MCP_SERVER_URL` | MCP weather server endpoint | Yes | - |
| `ACCUWEATHER_API_KEY` | AccuWeather API key | Yes | - |
| `NOTION_MCP_NAME` | Name for the MCP Notion server | Yes | - |
| `NOTION_MCP_SERVER_URL` | MCP Notion server endpoint | Yes | - |
| `NOTION_MCP_API_KEY` | Notion API token | Yes | - |
| `REDIS_MCP_NAME` | Name for the Redis MCP server | Optional | - |
| `REDIS_MCP_SERVER_URL` | Redis MCP server endpoint | Optional | - |
| `REDIS_URL` | Redis connection string for the Redis MCP server | Optional | - |

## 📊 Logging

The application uses structured JSON logging (see `pkg/logger`). Stack traces are captured for panics, and step-by-step request tracing is included in once-mode.

- **Location**: `logs/app.log`
- **Format**: JSON with timestamp, level, and message
- **Level**: Info and above

## 🔌 MCP Integration

- Weather (AccuWeather) via `@timlukahorstmann/mcp-weather`
- Notion via `@notionhq/notion-mcp-server`
- Redis via `@redis/mcp-redis` (local) or hosted Smithery endpoint

## 🚨 Troubleshooting

- Ensure MCP servers are running and accessible at configured endpoints
- Hosted MCPs cannot reach your local services; provide public URLs for dependencies (e.g., Redis)
- For prompt secrecy, prefer `SYSTEM_MESSAGE_FILE` over inlining strings

## 📝 License

This project is licensed under the MIT License - see the LICENSE file for details.
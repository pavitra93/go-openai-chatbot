# OpenAI ChatBot with Multi-MCP Integration

A sophisticated Go-based chatbot application that integrates OpenAI's GPT-4 with multiple Model Context Protocol (MCP) servers to provide intelligent conversations with access to various external services and APIs.

## 🌟 Features

- **OpenAI GPT-4 Integration**: Powered by OpenAI's latest GPT-4 model for intelligent conversations
- **Memory-enabled Chat**: Maintains conversation history for context-aware responses
- **Multi-MCP Integration**: Supports multiple MCP servers simultaneously for diverse functionality
- **Weather Forecasting**: Real-time weather data via AccuWeather API through MCP weather server
- **Notion Integration**: Create and manage Notion pages, databases, and content
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
SYSTEM_MESSAGE="You are a helpful AI assistant with access to various tools and services. You can help with weather information, Notion pages, and more."

# MCP Server Configuration
# Weather Server
ACCUWEATHER_MCP_NAME=weather
ACCUWEATHER_MCP_SERVER_URL=http://127.0.0.1:4004/mcp
ACCUWEATHER_API_KEY=your_accuweather_api_key

# Notion Server
NOTION_MCP_NAME=notion
NOTION_MCP_SERVER_URL=http://127.0.0.1:4005/mcp
NOTION_MCP_API_KEY=your_notion_api_token
```

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
}
ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
defer cancel()
if err := mcpManager.RegisterServers(ctx, servers); err != nil {
    slog.Error("failed to register some MCP servers", "error", err)
}
// Inspect registration stack (order)
slog.Info("MCP servers registered", "order", mcpManager.ListServersInOrder())
```

Under the hood, `RegisterServers` uses `errgroup` for parallelism and a simple exponential backoff (3 attempts) per server. Successful registrations are kept even if others fail.

## 🎯 Usage

Once the application starts, you'll see:

```
========Chatbot with Memory=========
Hello with Memory Chatbot
🧔🏻‍♂️ You: 
```

### Example Interactions

**Weather Queries:**
```
🧔🏻‍♂️ You: What's the weather like in New York today?
🤖 Chatbot: [Provides detailed weather information using AccuWeather API]

🧔🏻‍♂️ You: Will it rain in London tomorrow?
🤖 Chatbot: [Gives precipitation forecast for London]

🧔🏻‍♂️ You: Give me the 5-day forecast for Tokyo
🤖 Chatbot: [Shows 5-day weather forecast for Tokyo]
```

**Notion Integration:**
```
🧔🏻‍♂️ You: Create a new Notion page with my meeting notes
🤖 Chatbot: [Creates a new Notion page with structured content]

🧔🏻‍♂️ You: Search for pages about project planning
🤖 Chatbot: [Searches and returns relevant Notion pages]

🧔🏻‍♂️ You: Update my task database with new items
🤖 Chatbot: [Adds new tasks to your Notion database]
```

**General Conversation:**
```
🧔🏻‍♂️ You: Hello! How are you today?
🤖 Chatbot: Hello! I'm doing well, thank you for asking! I'm here and ready to help you with any questions you might have, including weather information, Notion pages, and more.

🧔🏻‍♂️ You: exit
🤖 Chatbot: Bye. Thanks for chatting with me.
```

### Available Commands

- `exit`, `quit`, or `bye`: Gracefully exit the application
- Any other text: Send as a message to the chatbot

## 🔧 Configuration

### Environment Variables

| Variable | Description | Required | Default |
|----------|-------------|----------|---------|
| `OPENAI_API_KEY` | Your OpenAI API key | Yes | - |
| `MAX_TOKENS` | Maximum tokens per response | Yes | - |
| `TEMPERATURE` | OpenAI temperature setting (0-1) | Yes | - |
| `SYSTEM_MESSAGE` | System prompt for the AI | Yes | - |
| `ACCUWEATHER_MCP_NAME` | Name for the MCP weather server | Yes | - |
| `ACCUWEATHER_MCP_SERVER_URL` | MCP weather server endpoint | Yes | - |
| `ACCUWEATHER_API_KEY` | AccuWeather API key | Yes | - |
| `NOTION_MCP_NAME` | Name for the MCP Notion server | Yes | - |
| `NOTION_MCP_SERVER_URL` | MCP Notion server endpoint | Yes | - |
| `NOTION_MCP_API_KEY` | Notion API token | Yes | - |

### OpenAI Configuration

The application supports various OpenAI settings:

- **Model**: Uses GPT-4 by default
- **Temperature**: Configurable for response creativity
- **Max Tokens**: Controls response length
- **History**: Maintains conversation context (configurable size)

## 🛠️ Development

### Building the Application

```bash
# Build for current platform
go build -o chatbot main.go

# Build for specific platforms
GOOS=linux GOARCH=amd64 go build -o chatbot-linux main.go
GOOS=windows GOARCH=amd64 go build -o chatbot.exe main.go
```

### Running Tests

```bash
go test ./...
```

## 📊 Logging

The application uses structured JSON logging:

- **Location**: `logs/app.log`
- **Format**: JSON with timestamp, level, and message
- **Level**: Info and above
- **Rotation**: Manual (logs append to existing file)

Example log entry:
```json
{
  "time": "2024-01-15T10:30:45Z",
  "level": "INFO",
  "msg": "Message sent to reciever channel"
}
```

## 🔌 MCP Integration

This chatbot integrates with the MCP (Model Context Protocol) ecosystem, supporting multiple servers simultaneously.

### Currently Integrated Services

- Weather (AccuWeather) via `@timlukahorstmann/mcp-weather`
- Notion via `@notionhq/notion-mcp-server`

### Architecture Benefits

- **Modular Design**: Each MCP server runs independently
- **Scalable**: Easy to add new integrations
- **Transport Flexibility**: Supports HTTP/SSE and stdio transports
- **Tool Schema Validation**: Automatic schema normalization for OpenAI compatibility

## 🚀 Upcoming Integrations

We're actively working on expanding the MCP ecosystem integration. Here's our roadmap:

### 📅 Calendar & Scheduling
- **Google Calendar**: Create, update, and manage calendar events

### 📧 Communication
- **Gmail**: Send, read, and manage emails
- **Slack**: Send messages and manage channels

### 📊 Productivity & Data
- **Google Sheets**: Read and write spreadsheet data
- **Jira**: Issue tracking and project management

### 🛒 E-commerce & Services
- **GitHub**: Repository and issue management

### 📈 Analytics & Monitoring
- **Google Analytics**: Website traffic analysis
- **Datadog**: Infrastructure monitoring
- **New Relic**: Application performance monitoring

### 🔐 Security & Authentication
- **Auth0**: User authentication management

### Contributing to Integrations

Want to contribute a new MCP integration? Here's how:

1. **Find or Create an MCP Server**: Look for existing MCP servers or create your own
2. **Add Configuration**: Update environment variables and server configs
3. **Test Integration**: Ensure proper schema validation and error handling
4. **Update Documentation**: Add examples and usage instructions
5. **Submit PR**: Follow our contribution guidelines

**Priority Integrations** (Next 3 months):
- Google Calendar
- Gmail
- Google Sheets
- GitHub
- Slack

## 🚨 Troubleshooting

- Ensure MCP servers are running on their configured ports and tokens are set
- Check logs for detailed request/step traces and stack traces on panic

## 📝 License

This project is licensed under the MIT License - see the LICENSE file for details.
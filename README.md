# OpenAI ChatBot with MCP Weather Integration

A sophisticated Go-based chatbot application that integrates OpenAI's GPT-4 with Model Context Protocol (MCP) servers to provide intelligent conversations with weather forecasting capabilities using AccuWeather API.

## 🌟 Features

- **OpenAI GPT-4 Integration**: Powered by OpenAI's latest GPT-4 model for intelligent conversations
- **Memory-enabled Chat**: Maintains conversation history for context-aware responses
- **MCP Weather Server**: Integrates with [TimLukaHorstmann/mcp-weather](https://github.com/TimLukaHorstmann/mcp-weather) for real-time weather data
- **AccuWeather API**: Provides accurate weather forecasts and current conditions
- **Structured Architecture**: Clean separation of concerns with modular design
- **Concurrent Processing**: Uses goroutines and channels for efficient message handling
- **Comprehensive Logging**: JSON-structured logging with file output
- **Environment Configuration**: Flexible configuration through environment variables

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

1. **MCP Manager**: Manages connections to MCP servers and tool schemas
2. **OpenAI Client**: Singleton wrapper for OpenAI API interactions
3. **Chatbot Service**: Handles conversation flow and user interactions
4. **Send/Receive Strategies**: Implements different message handling patterns
5. **Logger**: Structured JSON logging with file output

## 🚀 Quick Start

### Prerequisites

- Go 1.25.0 or later
- OpenAI API key
- AccuWeather API key (free tier available)
- Node.js (for MCP weather server)

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
SYSTEM_MESSAGE="You are a helpful AI assistant with access to weather information. You can provide weather forecasts and current conditions for any location."

# MCP Weather Server Configuration
ACCUWEATHER_MCP_NAME=weather
ACCUWEATHER_MCP_SERVER_URL=http://127.0.0.1:4004/messages
```

### 4. Start the MCP Weather Server

Before running the chatbot, start the MCP weather server using the provided command:

```bash
 $env:ACCUWEATHER_API_KEY = "<ACCUWEATHER_API_KEY>"
npx -y supergateway --stdio "npx -y @timlukahorstmann/mcp-weather" \
  --port 4004 \
  --baseUrl http://127.0.0.1:4004 \
  --ssePath /messages \
  --messagePath /message \
  --cors "*" \
  --env ACCUWEATHER_API_KEY="$ACCUWEATHER_API_KEY"
```

### 5. Run the Application

```bash
go run main.go
```

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

**General Conversation:**
```
🧔🏻‍♂️ You: Hello! How are you today?
🤖 Chatbot: Hello! I'm doing well, thank you for asking! I'm here and ready to help you with any questions you might have, including weather information if you need it.

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
| `ACCUWEATHER_MCP_SERVER_URL` | MCP server endpoint | Yes | - |

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

### Code Structure

The application follows clean architecture principles:

- **External Layer**: Handles external API clients (OpenAI, MCP)
- **Internal Layer**: Contains business logic and services
- **Package Layer**: Shared utilities and helpers

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

This chatbot integrates with the MCP (Model Context Protocol) ecosystem:

### Weather Tools Available

- **Hourly Forecast**: Get weather for the next 12 hours
- **Daily Forecast**: Get weather for up to 15 days
- **Location Support**: Any city or location worldwide
- **Unit Support**: Metric (°C) and Imperial (°F) units

### MCP Server Details

The weather functionality is provided by [@timlukahorstmann/mcp-weather](https://github.com/TimLukaHorstmann/mcp-weather), which:

- Uses AccuWeather API for accurate weather data
- Provides both hourly and daily forecasts
- Supports multiple locations and units
- Runs as a separate MCP server process

## 🚨 Troubleshooting

### Common Issues

1. **MCP Server Connection Failed**
   - Ensure the MCP weather server is running on port 4004
   - Check that `ACCUWEATHER_API_KEY` is set correctly
   - Verify the server URL in environment variables

2. **OpenAI API Errors**
   - Verify your OpenAI API key is valid and has sufficient credits
   - Check the `MAX_TOKENS` setting isn't too high
   - Ensure you have access to GPT-4 model

3. **Environment Variables Not Loaded**
   - Make sure `.env` file is in the project root
   - Check variable names match exactly (case-sensitive)
   - Restart the application after changing `.env`

4. **Log File Issues**
   - Ensure the `logs/` directory is writable
   - Check disk space availability
   - Verify file permissions

### Debug Mode

Enable debug logging by modifying the logger configuration in `pkg/logger/setup-logger.go`:

```go
handler := slog.NewJSONHandler(file, &slog.HandlerOptions{
    Level: slog.LevelDebug, // Change from Info to Debug
})
```

## 📝 License

This project is licensed under the MIT License - see the LICENSE file for details.

## 🤝 Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

### Development Guidelines

1. Follow Go best practices and conventions
2. Add appropriate tests for new features
3. Update documentation for any API changes
4. Ensure all environment variables are documented

## 🔗 Related Projects

- [MCP Weather Server](https://github.com/TimLukaHorstmann/mcp-weather) - Weather MCP server implementation
- [OpenAI Go SDK](https://github.com/openai/openai-go) - Official OpenAI Go client
- [Model Context Protocol](https://github.com/modelcontextprotocol) - MCP specification and tools

## 📞 Support

For issues and questions:

1. Check the troubleshooting section above
2. Review the logs in `logs/app.log`
3. Open an issue on the GitHub repository
4. Ensure all dependencies are up to date

---

**Happy Chatting! 🤖✨**
package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/openai/openai-go/v2"
)

// ServerConfig describes an MCP server to register
type MCPServerConfig struct {
	Name     string // unique name used as map key
	Endpoint string // SSE / HTTP endpoint or base url
	APIKey   string // optional auth token (if required)
	// add other options as needed (timeout, transport type, etc.)
}

// Manager is the central singleton that holds multiple MCP sessions and tool schemas.
type Manager struct {
	mu sync.RWMutex

	// sessions map: serverName -> connected ClientSession
	sessions map[string]*mcp.ClientSession

	// tools map: serverName -> []*mcp.Tool (raw tool descriptors from server)
	tools map[string][]*mcp.Tool

	// schemas map: serverName -> []openai.ChatCompletionToolUnionParam (OpenAI tool schemas)
	schemas map[string][]openai.ChatCompletionToolUnionParam
}

var (
	managerInstance *Manager
	managerOnce     sync.Once
)

// GetManager returns the singleton Manager
func GetManager() *Manager {
	managerOnce.Do(func() {
		managerInstance = &Manager{
			sessions: make(map[string]*mcp.ClientSession),
			tools:    make(map[string][]*mcp.Tool),
			schemas:  make(map[string][]openai.ChatCompletionToolUnionParam),
		}
	})
	return managerInstance
}

// RegisterServer connects to an MCP server, lists its tools and stores session/schema.
// If a session with the same name exists, it is closed/replaced.
func (m *Manager) RegisterServer(ctx context.Context, cfg *MCPServerConfig) error {
	if cfg.Name == "" {
		return fmt.Errorf("server name required")
	}
	if cfg.Endpoint == "" {
		return fmt.Errorf("endpoint required")
	}

	// Create client and transport
	client := mcp.NewClient(&mcp.Implementation{Name: "mcp-client", Version: "v1.0.0"}, nil)

	// Build transport. Adjust headers/opts depending on your SDK version.
	transport := &mcp.SSEClientTransport{
		Endpoint: cfg.Endpoint,
		// If your SDK supports headers, set them here. If not, remove.
		// Some transports accept Headers map; if not available remove.
		// Headers: map[string]string{"Authorization": "Bearer " + cfg.APIKey},
	}

	// Connect (use ctx from caller; it should include a timeout)
	session, err := client.Connect(ctx, transport, nil)
	if err != nil {
		return fmt.Errorf("failed to connect to %s: %w", cfg.Name, err)
	}

	// List tools
	toolsResult, err := session.ListTools(ctx, nil)
	if err != nil {
		_ = session.Close()
		return fmt.Errorf("failed to list tools on %s: %w", cfg.Name, err)
	}

	// Build OpenAI schemas for this server
	openAISchemas := make([]openai.ChatCompletionToolUnionParam, 0, len(toolsResult.Tools))
	for _, tool := range toolsResult.Tools {
		// tool.InputSchema is (per your earlier usage) a *jsonschema.Schema-like type.
		// We'll marshal it to JSON and unmarshal to map[string]any so it fits openai.FunctionParameters.
		var params openai.FunctionParameters
		if tool.InputSchema != nil {
			b, err := json.Marshal(tool.InputSchema)
			if err != nil {
				// if marshaling fails, fallback to small default schema
				log.Printf("warning: failed to marshal schema for tool %s on %s: %v", tool.Name, cfg.Name, err)
			} else {
				var m map[string]any
				if err := json.Unmarshal(b, &m); err != nil {
					log.Printf("warning: failed to unmarshal schema for tool %s on %s: %v", tool.Name, cfg.Name, err)
				} else {
					// optionally remove $defs / $schema to reduce size
					delete(m, "$defs")
					delete(m, "$schema")
					delete(m, "$id")
					params = openai.FunctionParameters(m)
				}
			}
		}

		fd := openai.FunctionDefinitionParam{
			Name:        cfg.Name + "__" + tool.Name,
			Description: openai.String(tool.Description),
			Parameters:  params,
		}
		openAISchemas = append(openAISchemas, openai.ChatCompletionFunctionTool(fd))
	}

	// Save into maps atomically: if an old session exists close it first
	m.mu.Lock()
	if oldSess, ok := m.sessions[cfg.Name]; ok {
		_ = oldSess.Close()
	}
	m.sessions[cfg.Name] = session
	m.tools[cfg.Name] = toolsResult.Tools
	m.schemas[cfg.Name] = openAISchemas
	m.mu.Unlock()

	log.Printf("registered MCP server %s with %d tools", cfg.Name, len(openAISchemas))
	return nil
}

// UnregisterServer closes and removes the session and schema for the given server name.
func (m *Manager) UnregisterServer(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if sess, ok := m.sessions[name]; ok {
		_ = sess.Close()
		delete(m.sessions, name)
	}
	delete(m.tools, name)
	delete(m.schemas, name)
	log.Printf("unregistered MCP server %s", name)
	return nil
}

// GetSession returns the ClientSession for a given server name (or nil if not found).
func (m *Manager) GetSession(name string) *mcp.ClientSession {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.sessions[name]
}

// GetSession returns the ClientSession for a given server name (or nil if not found).
func (m *Manager) GetAllSession() map[string]*mcp.ClientSession {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.sessions
}

// GetSchemas returns the OpenAI tool schemas for a given server name.
func (m *Manager) GetSchemas(name string) []openai.ChatCompletionToolUnionParam {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if s, ok := m.schemas[name]; ok {
		// return a copy to avoid caller mutating internal slice
		cpy := make([]openai.ChatCompletionToolUnionParam, len(s))
		copy(cpy, s)
		return cpy
	}
	return nil
}

// GetSchemas returns the OpenAI tool schemas for a given server name.
func (m *Manager) GetAllSchemas() map[string][]openai.ChatCompletionToolUnionParam {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.schemas
}

// ListServers returns the registered server names.
func (m *Manager) ListServers() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]string, 0, len(m.sessions))
	for k := range m.sessions {
		out = append(out, k)
	}
	return out
}

func (m *Manager) CallTool(ToolID string, ToolName string, args map[string]any) (string, error) {
	split := strings.Split(ToolName, "__")
	toolResp, err := m.GetSession(split[0]).CallTool(context.Background(), &mcp.CallToolParams{
		Name:      split[1],
		Arguments: args,
	})

	if err != nil {
		return err.Error(), err
	}

	//    We marshal the whole response to JSON so we preserve structured data.
	respBytes, err := json.Marshal(toolResp)
	var respStr string
	if err != nil {
		// fallback: use fmt.Sprintf
		respStr = fmt.Sprintf("%+v", toolResp)
	} else {
		respStr = string(respBytes)
	}

	return respStr, nil
}

func (m *Manager) Close(ToolName string) {
	split := strings.Split(ToolName, "__")
	m.GetSession(split[0]).Close()
}

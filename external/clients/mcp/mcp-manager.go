package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/openai/openai-go/v2"
	"golang.org/x/sync/errgroup"
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

	// order keeps server names in registration order
	order []string
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
			order:    make([]string, 0),
		}
	})
	return managerInstance
}

// ensureObjectSchema normalizes an arbitrary map into a valid JSON Schema object
// suitable for OpenAI tool function parameters. It guarantees the presence of
// "type":"object" and at least an empty "properties" map. It also strips
// unsupported or unnecessary keys that can cause rejections.
func ensureObjectSchema(schema map[string]any) map[string]any {
	if schema == nil {
		return map[string]any{
			"type":       "object",
			"properties": map[string]any{},
		}
	}

	// Remove noisy top-level keys often present in exported schemas
	delete(schema, "$defs")
	delete(schema, "$schema")
	delete(schema, "$id")

	// Enforce object type schema expected by OpenAI for tool parameters
	if t, ok := schema["type"].(string); !ok || t == "" {
		schema["type"] = "object"
	}

	// Ensure properties exists; OpenAI requires an object schema with properties
	if _, ok := schema["properties"]; !ok {
		schema["properties"] = map[string]any{}
	}

	// If someone provided an array/object of required fields, keep as-is;
	// otherwise do not add a "required" key to avoid invalid references.

	return schema
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
	transport := &mcp.StreamableClientTransport{
		Endpoint: cfg.Endpoint,
	}

	// Connect (use ctx from caller; it should include a timeout)
	session, err := client.Connect(ctx, transport, nil)
	if err != nil {
		slog.Error("failed to connect", "server", cfg.Name, "error", err)
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
				slog.Warn("failed to marshal schema", "tool", tool.Name, "server", cfg.Name, "error", err)
			} else {
				var m map[string]any
				if err := json.Unmarshal(b, &m); err != nil {
					slog.Warn("failed to unmarshal schema", "tool", tool.Name, "server", cfg.Name, "error", err)
				} else {
					normalized := ensureObjectSchema(m)
					params = openai.FunctionParameters(normalized)
				}
			}
		}

		// If there was no schema provided or normalization led to empty params,
		// enforce a minimal valid object schema to satisfy OpenAI validation.
		if params == nil {
			params = openai.FunctionParameters(map[string]any{
				"type":       "object",
				"properties": map[string]any{},
			})
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
	// append to order if not already there
	found := false
	for _, n := range m.order {
		if n == cfg.Name {
			found = true
			break
		}
	}
	if !found {
		m.order = append(m.order, cfg.Name)
	}
	m.mu.Unlock()

	slog.Info("registered MCP server", "server", cfg.Name, "tool_count", len(openAISchemas))
	return nil
}

// RegisterServers registers multiple servers concurrently with retry/backoff.
// Returns first error encountered, if any. Successful registrations remain.
func (m *Manager) RegisterServers(ctx context.Context, cfgs []MCPServerConfig) error {
	g, ctx := errgroup.WithContext(ctx)
	for i := range cfgs {
		cfg := cfgs[i]
		g.Go(func() error {
			var lastErr error
			backoff := 300 * time.Millisecond
			for attempt := 1; attempt <= 3; attempt++ {
				if err := m.RegisterServer(ctx, &cfg); err != nil {
					lastErr = err
					slog.Warn("register server failed; retrying", "server", cfg.Name, "attempt", attempt, "error", err)
					select {
					case <-time.After(backoff):
						backoff *= 2
						continue
					case <-ctx.Done():
						return ctx.Err()
					}
				}
				slog.Info("server registered", "server", cfg.Name)
				return nil
			}
			return lastErr
		})
	}
	return g.Wait()
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
	slog.Info("unregistered MCP server", "server", name)
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

// ListServersInOrder returns the registration order list.
func (m *Manager) ListServersInOrder() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]string, len(m.order))
	copy(out, m.order)
	return out
}

func (m *Manager) CallTool(ToolID string, ToolName string, args map[string]any) (string, error) {
	split := strings.Split(ToolName, "__")
	if len(split) != 2 {
		return "", fmt.Errorf("invalid tool name format: %s", ToolName)
	}

	// Log the full arguments for debugging
	argsBytes, _ := json.Marshal(args)
	slog.Info("calling tool", "tool", ToolName, "args", string(argsBytes))

	toolResp, err := m.GetSession(split[0]).CallTool(context.Background(), &mcp.CallToolParams{
		Name:      split[1],
		Arguments: args,
	})

	if err != nil {
		slog.Error("tool call failed", "tool", ToolName, "error", err)
		return err.Error(), err
	}

	// Marshal the whole response to JSON so we preserve structured data.
	respBytes, err := json.Marshal(toolResp)
	var respStr string
	if err != nil {
		slog.Error("failed to marshal tool response", "tool", ToolName, "error", err)
		// fallback: use fmt.Sprintf
		respStr = fmt.Sprintf("%+v", toolResp)
	} else {
		respStr = string(respBytes)
	}

	slog.Info("tool completed", "tool", ToolName)
	return respStr, nil
}

func (m *Manager) Close(ToolName string) {
	split := strings.Split(ToolName, "__")
	m.GetSession(split[0]).Close()
}

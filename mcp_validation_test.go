package main

import (
	"encoding/json"
	"testing"
)

// Prevent malformed MCP tool schemas from being exposed to ChatGPT.
// The previous version serialized nil variadic required parameters as JSON null.
func TestAllMCPToolSchemasAreValid(t *testing.T) {
	tlist := tools()
	if len(tlist) == 0 {
		t.Fatal("no tools")
	}
	seen := map[string]bool{}
	for _, tool := range tlist {
		if tool.Name == "" || seen[tool.Name] {
			t.Fatalf("duplicate/empty name %q", tool.Name)
		}
		seen[tool.Name] = true
		b, err := json.Marshal(tool)
		if err != nil {
			t.Fatal(err)
		}
		var parsed map[string]any
		if err := json.Unmarshal(b, &parsed); err != nil {
			t.Fatal(err)
		}
		s, ok := parsed["inputSchema"].(map[string]any)
		if !ok || s["type"] != "object" {
			t.Fatalf("invalid schema for %s: %s", tool.Name, b)
		}
		if s["properties"] == nil {
			t.Fatalf("null properties for %s", tool.Name)
		}
		if r, exists := s["required"]; exists {
			a, ok := r.([]any)
			if !ok {
				t.Fatalf("required is not array for %s: %v", tool.Name, r)
			}
			props := s["properties"].(map[string]any)
			for _, field := range a {
				name, ok := field.(string)
				if !ok || name == "" || props[name] == nil {
					t.Fatalf("invalid required member %s: %v", tool.Name, field)
				}
			}
		}
	}
}

func TestToolListSchemaSerialization(t *testing.T) {
	a, err := NewApp(t.TempDir() + "/config.json")
	if err != nil {
		t.Fatal(err)
	}
	response, ok := a.handleRPC([]byte(`{"jsonrpc":"2.0","id":42,"method":"tools/list"}`))
	if !ok {
		t.Fatal("tools/list omitted response")
	}
	b, err := json.Marshal(response)
	if err != nil {
		t.Fatal(err)
	}
	var obj map[string]any
	if err := json.Unmarshal(b, &obj); err != nil {
		t.Fatal(err)
	}
	result := obj["result"].(map[string]any)
	tools := result["tools"].([]any)
	if len(tools) != 72 {
		t.Fatalf("tool count %d", len(tools))
	}
	for _, it := range tools {
		tool := it.(map[string]any)
		schema := tool["inputSchema"].(map[string]any)
		if v, exists := schema["required"]; exists && v == nil {
			t.Fatalf("null required on %s", tool["name"])
		}
	}
}

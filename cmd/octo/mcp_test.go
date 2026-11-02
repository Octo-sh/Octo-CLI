package main

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestMcpCommandStartsAndResponds(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	cmd := mcpCmd
	var in bytes.Buffer
	var out bytes.Buffer

	in.WriteString(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"1.0"}}}` + "\n")

	cmd.SetIn(&in)
	cmd.SetOut(&out)
	cmd.SetErr(&out)

	cmd.SetContext(ctx)
	if err := cmd.RunE(cmd, nil); err != nil {
		t.Fatal(err)
	}

	output := out.String()
	if !strings.Contains(output, `"protocolVersion":"2024-11-05"`) {
		t.Fatalf("expected protocolVersion in response, got: %s", output)
	}
	if !strings.Contains(output, `"name":"octo"`) {
		t.Fatalf("expected server name 'octo' in response, got: %s", output)
	}
}

func TestMcpCommandToolsListAndCall(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/mcptest\ngo 1.24\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "main.go"), []byte("package main\nimport \"os\"\nfunc main() { _ = os.Getenv(\"DATABASE_URL\") }\n"), 0644); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cmd := mcpCmd
	var in bytes.Buffer
	var out bytes.Buffer

	// 1. Initialize
	in.WriteString(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test-agent","version":"1.0"}}}` + "\n")

	// 2. tools/list
	in.WriteString(`{"jsonrpc":"2.0","id":2,"method":"tools/list"}` + "\n")

	// 3. tools/call octo_env_check
	envCall := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      3,
		"method":  "tools/call",
		"params": map[string]interface{}{
			"name": "octo_env_check",
			"arguments": map[string]interface{}{
				"path": root,
			},
		},
	}
	envData, _ := json.Marshal(envCall)
	in.Write(append(envData, '\n'))

	// 4. resources/list
	in.WriteString(`{"jsonrpc":"2.0","id":4,"method":"resources/list"}` + "\n")

	// 5. resources/read octo://env
	resCall := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      5,
		"method":  "resources/read",
		"params": map[string]interface{}{
			"uri": "octo://env?path=" + root,
		},
	}
	resData, _ := json.Marshal(resCall)
	in.Write(append(resData, '\n'))

	cmd.SetIn(&in)
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetContext(ctx)

	if err := cmd.RunE(cmd, nil); err != nil {
		t.Fatal(err)
	}

	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	if len(lines) < 5 {
		t.Fatalf("expected at least 5 JSON-RPC response lines, got %d: %s", len(lines), out.String())
	}

	// Verify tools/list has octo_env_check
	if !strings.Contains(lines[1], "octo_env_check") {
		t.Fatalf("expected tools/list to include 'octo_env_check', got: %s", lines[1])
	}

	// Verify env_check result has database_url
	if !strings.Contains(lines[2], "DATABASE_URL") {
		t.Fatalf("expected octo_env_check output to detect DATABASE_URL, got: %s", lines[2])
	}

	// Verify resources/list has octo://env
	if !strings.Contains(lines[3], "octo://env") {
		t.Fatalf("expected resources/list to include 'octo://env', got: %s", lines[3])
	}

	// Verify resources/read returns valid env resource
	if !strings.Contains(lines[4], "total_variables") {
		t.Fatalf("expected resources/read to return total_variables, got: %s", lines[4])
	}
}

func TestMcpConfigCommand(t *testing.T) {
	cmd := mcpConfigCmd
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)

	if err := cmd.RunE(cmd, nil); err != nil {
		t.Fatalf("mcp config failed: %v", err)
	}

	output := out.String()
	var cfg map[string]interface{}
	if err := json.Unmarshal([]byte(output), &cfg); err != nil {
		t.Fatalf("expected valid JSON output from mcp config, got: %s (err: %v)", output, err)
	}

	servers, ok := cfg["mcpServers"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected mcpServers object in output, got: %v", cfg)
	}

	octoServer, ok := servers["octo"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected octo server in mcpServers, got: %v", servers)
	}

	command, ok := octoServer["command"].(string)
	if !ok || command == "" {
		t.Fatalf("expected non-empty command string in octo server, got: %v", octoServer["command"])
	}

	args, ok := octoServer["args"].([]interface{})
	if !ok || len(args) != 1 || args[0] != "mcp" {
		t.Fatalf("expected args [\"mcp\"], got: %v", octoServer["args"])
	}
}

func TestMcpInitCommand_NewRepository(t *testing.T) {
	root := t.TempDir()

	cmd := mcpInitCmd
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)

	if err := cmd.RunE(cmd, []string{root}); err != nil {
		t.Fatalf("mcp init failed: %v", err)
	}

	mcpPath := filepath.Join(root, ".mcp.json")
	data, err := os.ReadFile(mcpPath)
	if err != nil {
		t.Fatalf("expected .mcp.json to be created, got error: %v", err)
	}

	var rootCfg map[string]interface{}
	if err := json.Unmarshal(data, &rootCfg); err != nil {
		t.Fatalf("expected valid JSON in .mcp.json, got: %s (err: %v)", string(data), err)
	}

	servers, ok := rootCfg["mcpServers"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected mcpServers map, got: %v", rootCfg)
	}
	if _, ok := servers["octo"]; !ok {
		t.Fatalf("expected octo server in mcpServers, got: %v", servers)
	}

	cursorPath := filepath.Join(root, ".cursor", "mcp.json")
	if _, err := os.Stat(cursorPath); !os.IsNotExist(err) {
		t.Fatalf("did not expect .cursor/mcp.json when .cursor does not exist")
	}

	outStr := out.String()
	if !strings.Contains(outStr, ".mcp.json") {
		t.Fatalf("expected output to mention .mcp.json, got: %s", outStr)
	}
}

func TestMcpInitCommand_WithCursorDir(t *testing.T) {
	root := t.TempDir()
	cursorDir := filepath.Join(root, ".cursor")
	if err := os.Mkdir(cursorDir, 0755); err != nil {
		t.Fatal(err)
	}

	cmd := mcpInitCmd
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)

	if err := cmd.RunE(cmd, []string{root}); err != nil {
		t.Fatalf("mcp init failed: %v", err)
	}

	// Verify both .mcp.json and .cursor/mcp.json exist
	mcpPath := filepath.Join(root, ".mcp.json")
	if _, err := os.Stat(mcpPath); err != nil {
		t.Fatalf("expected .mcp.json to exist: %v", err)
	}

	cursorMcpPath := filepath.Join(cursorDir, "mcp.json")
	data, err := os.ReadFile(cursorMcpPath)
	if err != nil {
		t.Fatalf("expected .cursor/mcp.json to exist: %v", err)
	}

	var cursorCfg map[string]interface{}
	if err := json.Unmarshal(data, &cursorCfg); err != nil {
		t.Fatalf("expected valid JSON in .cursor/mcp.json: %v", err)
	}
	servers := cursorCfg["mcpServers"].(map[string]interface{})
	if _, ok := servers["octo"]; !ok {
		t.Fatalf("expected octo in .cursor/mcp.json, got: %v", servers)
	}
}

func TestMcpInitCommand_SafeMergeExistingServers(t *testing.T) {
	root := t.TempDir()

	initialMcp := `{
  "mcpServers": {
    "github": {
      "command": "npx",
      "args": ["-y", "@modelcontextprotocol/server-github"]
    }
  },
  "customSetting": "custom-value"
}
`
	if err := os.WriteFile(filepath.Join(root, ".mcp.json"), []byte(initialMcp), 0644); err != nil {
		t.Fatal(err)
	}

	cursorDir := filepath.Join(root, ".cursor")
	if err := os.Mkdir(cursorDir, 0755); err != nil {
		t.Fatal(err)
	}
	initialCursorMcp := `{
  "mcpServers": {
    "postgres": {
      "command": "npx",
      "args": ["-y", "@modelcontextprotocol/server-postgres"]
    }
  }
}
`
	if err := os.WriteFile(filepath.Join(cursorDir, "mcp.json"), []byte(initialCursorMcp), 0644); err != nil {
		t.Fatal(err)
	}

	cmd := mcpInitCmd
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)

	if err := cmd.RunE(cmd, []string{root}); err != nil {
		t.Fatalf("mcp init failed: %v", err)
	}

	// Verify root .mcp.json preserved "github" and "customSetting"
	mcpData, err := os.ReadFile(filepath.Join(root, ".mcp.json"))
	if err != nil {
		t.Fatal(err)
	}
	var mcpCfg map[string]interface{}
	if err := json.Unmarshal(mcpData, &mcpCfg); err != nil {
		t.Fatal(err)
	}
	if mcpCfg["customSetting"] != "custom-value" {
		t.Fatalf("expected customSetting to be preserved, got: %v", mcpCfg["customSetting"])
	}
	mcpServers := mcpCfg["mcpServers"].(map[string]interface{})
	if _, ok := mcpServers["github"]; !ok {
		t.Fatalf("expected github server to be preserved in .mcp.json")
	}
	if _, ok := mcpServers["octo"]; !ok {
		t.Fatalf("expected octo server to be added to .mcp.json")
	}

	// Verify .cursor/mcp.json preserved "postgres"
	cursorData, err := os.ReadFile(filepath.Join(cursorDir, "mcp.json"))
	if err != nil {
		t.Fatal(err)
	}
	var cursorCfg map[string]interface{}
	if err := json.Unmarshal(cursorData, &cursorCfg); err != nil {
		t.Fatal(err)
	}
	cursorServers := cursorCfg["mcpServers"].(map[string]interface{})
	if _, ok := cursorServers["postgres"]; !ok {
		t.Fatalf("expected postgres server to be preserved in .cursor/mcp.json")
	}
	if _, ok := cursorServers["octo"]; !ok {
		t.Fatalf("expected octo server to be added to .cursor/mcp.json")
	}
}

func TestMcpInitCommand_InvalidJSONError(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, ".mcp.json"), []byte("{ broken json "), 0644); err != nil {
		t.Fatal(err)
	}

	cmd := mcpInitCmd
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)

	err := cmd.RunE(cmd, []string{root})
	if err == nil {
		t.Fatal("expected error when existing .mcp.json has invalid JSON, got nil")
	}
	if !strings.Contains(err.Error(), "valid JSON") {
		t.Fatalf("expected actionable error message, got: %v", err)
	}
}

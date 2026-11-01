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

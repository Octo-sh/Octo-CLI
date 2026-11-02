package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/Octo-sh/Octo-CLI/internal/mcp"
	"github.com/spf13/cobra"
)

var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "Start the Model Context Protocol (MCP) server over stdio",
	Long: `The mcp command starts a JSON-RPC 2.0 Model Context Protocol (MCP) server
over standard input and output (stdio).

AI coding agents (such as Claude Code, Cursor, Antigravity, Cline, and Windsurf)
can connect to this server to discover repository technologies, generate
deterministic execution plans, execute and verify stacks, and diagnose failures.`,
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		server := mcp.NewServer()
		return server.Serve(cmd.Context(), cmd.InOrStdin(), cmd.OutOrStdout())
	},
}

var mcpConfigCmd = &cobra.Command{
	Use:     "config",
	Aliases: []string{"show", "generate"},
	Short:   "Print copy-pasteable MCP server configuration JSON",
	Long: `The config command prints the JSON block needed to configure Octo as an MCP
server in .mcp.json, Cursor, Claude Desktop, or Windsurf.`,
	Args:         cobra.NoArgs,
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		exe := resolveOctoExecutable()
		cfg, err := generateMCPConfig(exe)
		if err != nil {
			return err
		}
		fmt.Fprint(cmd.OutOrStdout(), cfg)
		return nil
	},
}

var mcpInitCmd = &cobra.Command{
	Use:     "init [path]",
	Aliases: []string{"setup", "install"},
	Short:   "Initialize or merge Octo MCP server into project configuration",
	Long: `The init command safely writes or merges the Octo MCP server entry into
.mcp.json (and .cursor/mcp.json if a .cursor directory exists) without overwriting
existing MCP servers.`,
	Args:         cobra.MaximumNArgs(1),
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		targetDir := "."
		if len(args) > 0 {
			targetDir = args[0]
		}

		info, err := os.Stat(targetDir)
		if err != nil {
			return fmt.Errorf("target directory %q not found: %w", targetDir, err)
		}
		if !info.IsDir() {
			return fmt.Errorf("target path %q is not a directory", targetDir)
		}

		exe := resolveOctoExecutable()
		configuredFiles, err := initMCPConfigs(targetDir, exe)
		if err != nil {
			return err
		}

		for _, file := range configuredFiles {
			displayPath := file
			if rel, err := filepath.Rel(targetDir, file); err == nil && !strings.HasPrefix(rel, "..") {
				displayPath = rel
			}
			fmt.Fprintf(cmd.OutOrStdout(), "==> Configured Octo MCP server in %s\n", displayPath)
		}
		fmt.Fprintln(cmd.OutOrStdout(), "\nAI coding agents (Cursor, Claude Code, Windsurf) can now use Octo tools.")
		return nil
	},
}

func init() {
	mcpCmd.GroupID = "main"
	mcpCmd.AddCommand(mcpConfigCmd)
	mcpCmd.AddCommand(mcpInitCmd)
	rootCmd.AddCommand(mcpCmd)
}

// resolveOctoExecutable returns the absolute path to the current octo binary,
// falling back to "octo" on PATH if executable resolution fails.
func resolveOctoExecutable() string {
	exe, err := os.Executable()
	if err == nil && exe != "" {
		if abs, err := filepath.Abs(exe); err == nil {
			return abs
		}
		return filepath.Clean(exe)
	}
	if path, err := exec.LookPath("octo"); err == nil {
		if abs, err := filepath.Abs(path); err == nil {
			return abs
		}
		return path
	}
	return "octo"
}

// generateMCPConfig returns the formatted JSON string for an mcpServers configuration.
func generateMCPConfig(exePath string) (string, error) {
	config := map[string]interface{}{
		"mcpServers": map[string]interface{}{
			"octo": map[string]interface{}{
				"command": exePath,
				"args":    []string{"mcp"},
			},
		},
	}
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshal mcp config: %w", err)
	}
	return string(data) + "\n", nil
}

// initMCPConfigs creates or merges .mcp.json and .cursor/mcp.json in targetDir.
func initMCPConfigs(targetDir, exePath string) ([]string, error) {
	var configured []string

	// 1. Root .mcp.json
	rootMcpPath := filepath.Join(targetDir, ".mcp.json")
	if err := mergeMCPConfigFile(rootMcpPath, exePath); err != nil {
		return nil, fmt.Errorf("failed to configure %s: %w", rootMcpPath, err)
	}
	configured = append(configured, rootMcpPath)

	// 2. .cursor/mcp.json (if .cursor/ directory or file exists)
	cursorDir := filepath.Join(targetDir, ".cursor")
	cursorMcpPath := filepath.Join(cursorDir, "mcp.json")
	stat, err := os.Stat(cursorDir)
	cursorExists := (err == nil && stat.IsDir())
	if !cursorExists {
		if _, err := os.Stat(cursorMcpPath); err == nil {
			cursorExists = true
		}
	}

	if cursorExists {
		if err := mergeMCPConfigFile(cursorMcpPath, exePath); err != nil {
			return nil, fmt.Errorf("failed to configure %s: %w", cursorMcpPath, err)
		}
		configured = append(configured, cursorMcpPath)
	}

	return configured, nil
}

// mergeMCPConfigFile safely writes or merges the "octo" server entry into filePath
// preserving all existing MCP servers and other JSON attributes.
func mergeMCPConfigFile(filePath, exePath string) error {
	var root map[string]interface{}

	data, err := os.ReadFile(filePath)
	if err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("read %s: %w", filePath, err)
		}
		root = make(map[string]interface{})
	} else {
		trimmed := bytes.TrimSpace(data)
		if len(trimmed) == 0 {
			root = make(map[string]interface{})
		} else {
			if err := json.Unmarshal(trimmed, &root); err != nil {
				return fmt.Errorf("failed to parse existing JSON in %s: %w (ensure file is valid JSON)", filePath, err)
			}
		}
	}

	if root == nil {
		root = make(map[string]interface{})
	}

	var mcpServers map[string]interface{}
	if rawServers, exists := root["mcpServers"]; exists {
		if m, ok := rawServers.(map[string]interface{}); ok {
			mcpServers = m
		} else {
			mcpServers = make(map[string]interface{})
		}
	} else {
		mcpServers = make(map[string]interface{})
	}

	mcpServers["octo"] = map[string]interface{}{
		"command": exePath,
		"args":    []interface{}{"mcp"},
	}
	root["mcpServers"] = mcpServers

	formatted, err := json.MarshalIndent(root, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal %s: %w", filePath, err)
	}
	formatted = append(formatted, '\n')

	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create directory %s: %w", dir, err)
	}

	if err := os.WriteFile(filePath, formatted, 0644); err != nil {
		return fmt.Errorf("write %s: %w", filePath, err)
	}

	return nil
}

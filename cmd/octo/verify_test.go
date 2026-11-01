package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Octo-sh/Octo-CLI/internal/intelligence"
)

func resetVerifyFlags() {
	_ = verifyCmd.Flags().Set("json", "false")
	_ = verifyCmd.Flags().Set("strict", "false")
}

func TestVerifyCommandJSON(t *testing.T) {
	resetVerifyFlags()
	tempDir := t.TempDir()
	packageJSON := `{
		"name": "verify-test",
		"version": "1.0.0",
		"scripts": {
			"start": "node index.js"
		}
	}`
	if err := os.WriteFile(filepath.Join(tempDir, "package.json"), []byte(packageJSON), 0644); err != nil {
		t.Fatalf("failed to write package.json: %v", err)
	}

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"verify", tempDir, "--json"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error executing verify: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, `"project_name": "verify-test"`) {
		t.Fatalf("expected JSON output to contain project name 'verify-test', got: %s", out)
	}
}

func TestVerifyCommandHumanOutputCaptured(t *testing.T) {
	resetVerifyFlags()
	tempDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(tempDir, "go.mod"), []byte("module example.com/demo\ngo 1.24\n"), 0644); err != nil {
		t.Fatalf("failed to write go.mod: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tempDir, "main.go"), []byte("package main\nfunc main() {}\n"), 0644); err != nil {
		t.Fatalf("failed to write main.go: %v", err)
	}

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"verify", tempDir})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error executing verify: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Verification report for demo") {
		t.Fatalf("expected human output to contain project report header, got: %s", out)
	}
	if !strings.Contains(out, ".octo.lock:") {
		t.Fatalf("expected human output to contain .octo.lock status, got: %s", out)
	}
}

func TestVerifyCommandStrictFailsWithoutLock(t *testing.T) {
	resetVerifyFlags()
	tempDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(tempDir, "go.mod"), []byte("module example.com/strict-test\ngo 1.24\n"), 0644); err != nil {
		t.Fatalf("failed to write go.mod: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tempDir, "main.go"), []byte("package main\nfunc main() {}\n"), 0644); err != nil {
		t.Fatalf("failed to write main.go: %v", err)
	}

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"verify", tempDir, "--strict"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error executing verify with --strict when .octo.lock is missing")
	}

	if !strings.Contains(err.Error(), "strict verification failed") {
		t.Fatalf("expected strict verification error message, got: %v", err)
	}
}

func TestVerifyCommandStrictSucceedsWithLock(t *testing.T) {
	resetVerifyFlags()
	tempDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(tempDir, "go.mod"), []byte("module example.com/strict-success\ngo 1.24\n"), 0644); err != nil {
		t.Fatalf("failed to write go.mod: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tempDir, "main.go"), []byte("package main\nfunc main() {}\n"), 0644); err != nil {
		t.Fatalf("failed to write main.go: %v", err)
	}

	model, err := intelligence.Analyze(tempDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(model.Components) == 0 || len(model.Components[0].ExecutionCandidates) == 0 {
		t.Fatal("expected execution candidates")
	}

	comp := model.Components[0]
	cand := comp.ExecutionCandidates[0]

	fp, err := intelligence.ExecutionFingerprint(tempDir, comp, cand)
	if err != nil {
		t.Fatal(err)
	}

	lock := intelligence.OctoLock{
		Spec:    intelligence.OctoLockSpec,
		Project: model.Name,
		Strategies: []intelligence.VerifiedStrategy{
			{
				Component:   model.Name,
				Candidate:   cand.ID,
				Command:     cand.Command,
				Fingerprint: fp,
				VerifiedAt:  time.Now().UTC(),
			},
		},
	}
	if err := intelligence.SaveOctoLock(tempDir, lock); err != nil {
		t.Fatal(err)
	}

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"verify", tempDir, "--strict"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error executing verify --strict with valid lock: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "fingerprint verified") {
		t.Fatalf("expected output to confirm fingerprint verified, got: %s", out)
	}
}

func TestVerifyCommandFailsWhenPortDown(t *testing.T) {
	resetVerifyFlags()
	tempDir := t.TempDir()
	packageJSON := `{
		"name": "verify-port-fail",
		"version": "1.0.0",
		"scripts": {
			"start": "node index.js"
		}
	}`
	if err := os.WriteFile(filepath.Join(tempDir, "package.json"), []byte(packageJSON), 0644); err != nil {
		t.Fatalf("failed to write package.json: %v", err)
	}
	octoYaml := `components:
  verify-port-fail:
    port: 54322
`
	if err := os.WriteFile(filepath.Join(tempDir, ".octo.yaml"), []byte(octoYaml), 0644); err != nil {
		t.Fatalf("failed to write .octo.yaml: %v", err)
	}

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"verify", tempDir, "--json"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected verify to return an error when target port is down")
	}

	out := buf.String()
	var report map[string]interface{}
	if jsonErr := json.Unmarshal([]byte(out), &report); jsonErr != nil {
		t.Fatalf("failed to unmarshal JSON report: %v, raw output: %s", jsonErr, out)
	}

	if report["success"] != false {
		t.Fatalf("expected success to be false, got: %v", report["success"])
	}
	if report["error_code"] == nil || report["error_code"] == "" {
		t.Fatalf("expected error_code in JSON report: %v", report)
	}
	if report["root_cause"] == nil || report["root_cause"] == "" {
		t.Fatalf("expected root_cause in JSON report: %v", report)
	}
	if report["suggested_fix"] == nil || report["suggested_fix"] == "" {
		t.Fatalf("expected suggested_fix in JSON report: %v", report)
	}
}

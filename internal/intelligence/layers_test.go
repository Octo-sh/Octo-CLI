package intelligence

import (
	"context"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"
)

func TestCorePipelineLayerInterfaces(t *testing.T) {
	// Compile-time interface assertion
	var _ Analyzer = StaticAnalyzer{}
	var _ Analyzer = DefaultAnalyzer()
	var _ Planner = DeterministicPlanner{}
	var _ Planner = DefaultPlanner()
	var _ Runner = PlanRunner{}
	var _ Runner = DefaultRunner()
	var _ Verifier = ReadinessVerifier{}
	var _ Verifier = DefaultVerifier()

	if DefaultAnalyzer() == nil {
		t.Fatal("expected DefaultAnalyzer to be non-nil")
	}
	if DefaultPlanner() == nil {
		t.Fatal("expected DefaultPlanner to be non-nil")
	}
	if DefaultRunner() == nil {
		t.Fatal("expected DefaultRunner to be non-nil")
	}
	if DefaultVerifier() == nil {
		t.Fatal("expected DefaultVerifier to be non-nil")
	}
}

func TestStaticAnalyzerDoesNotLeakOSRuntime(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/demo\ngo 1.25\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "main.go"), []byte("package main\n\nfunc main() {}\n"), 0644); err != nil {
		t.Fatal(err)
	}

	analyzer := DefaultAnalyzer()
	model, err := analyzer.Analyze(root)
	if err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}

	if model.Language != "Go" {
		t.Fatalf("expected Go language, got %q", model.Language)
	}
	if len(model.Components) != 1 {
		t.Fatalf("expected 1 component, got %d", len(model.Components))
	}
	// Verify static model does not contain runtime process state
	if model.Root != root {
		t.Fatalf("model.Root = %q, want %q", model.Root, root)
	}
}

func TestDeterministicPlannerInterface(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "package.json"), []byte(`{"name":"web","scripts":{"start":"node index.js"}}`), 0644); err != nil {
		t.Fatal(err)
	}

	analyzer := DefaultAnalyzer()
	model, err := analyzer.Analyze(root)
	if err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}

	planner := DefaultPlanner()
	plan, err := planner.Plan(context.Background(), model)
	if err != nil {
		t.Fatalf("Plan failed: %v", err)
	}

	if len(plan.Steps) == 0 {
		t.Fatal("expected plan steps to be generated")
	}
}

func TestReadinessVerifierInterface(t *testing.T) {
	// Start a dummy TCP listener
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()

	_, portStr, err := net.SplitHostPort(listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		t.Fatal(err)
	}

	verifier := DefaultVerifier()
	checks := []VerificationCheck{
		{
			ID:        "check-tcp-port",
			Kind:      VerificationPort,
			Component: "web",
			Host:      "127.0.0.1",
			Port:      port,
			Timeout:   500 * time.Millisecond,
		},
	}

	results, err := verifier.Verify(context.Background(), checks)
	if err != nil {
		t.Fatalf("Verify failed: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if !results[0].Passed {
		t.Fatalf("expected check to pass, got reason: %s", results[0].Reason)
	}
}

func TestPlanRunnerInterface(t *testing.T) {
	runner := DefaultRunner()
	if runner == nil {
		t.Fatal("expected DefaultRunner to be non-nil")
	}

	planRunner := NewPlanRunner(NewRuntimeResolver())
	if len(planRunner.Resolver.adapters) == 0 {
		t.Fatal("expected NewPlanRunner resolver adapters to be initialized")
	}
}

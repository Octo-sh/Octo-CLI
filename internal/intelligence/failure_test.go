package intelligence

import (
	"errors"
	"testing"
)

func TestClassifyExecutionFailure(t *testing.T) {
	tests := []struct {
		name     string
		reason   string
		err      error
		expected FailureClassification
	}{
		{
			name:     "port already in use",
			reason:   "listen tcp :8080: bind: address already in use",
			err:      nil,
			expected: FailureClassPortConflict,
		},
		{
			name:     "port unreachable verification",
			reason:   "Startup verification failed: port is not reachable",
			err:      errors.New("timeout connecting to 127.0.0.1:3000"),
			expected: FailureClassPortConflict,
		},
		{
			name:     "node missing module",
			reason:   "Error: Cannot find module 'express'",
			err:      nil,
			expected: FailureClassMissingDependency,
		},
		{
			name:     "python module not found",
			reason:   "ModuleNotFoundError: No module named 'fastapi'",
			err:      nil,
			expected: FailureClassMissingDependency,
		},
		{
			name:     "command not found",
			reason:   "sh: nodemon: command not found",
			err:      nil,
			expected: FailureClassMissingDependency,
		},
		{
			name:     "go syntax error",
			reason:   "syntax error: unexpected newline, expecting comma or }",
			err:      nil,
			expected: FailureClassCompilationSyntax,
		},
		{
			name:     "typescript type error",
			reason:   "TypeError: Property 'foo' does not exist on type 'Bar'",
			err:      nil,
			expected: FailureClassCompilationSyntax,
		},
		{
			name:     "context deadline exceeded",
			reason:   "context deadline exceeded",
			err:      errors.New("timeout waiting for healthcheck"),
			expected: FailureClassTimeout,
		},
		{
			name:     "process crash exit status",
			reason:   "process terminated with exit status 137",
			err:      nil,
			expected: FailureClassProcessCrash,
		},
		{
			name:     "segmentation fault",
			reason:   "segmentation fault (core dumped)",
			err:      nil,
			expected: FailureClassProcessCrash,
		},
		{
			name:     "docker daemon unavailable",
			reason:   "Cannot connect to the Docker daemon at unix:///var/run/docker.sock. Is the docker daemon running?",
			err:      nil,
			expected: FailureClassDaemonUnavailable,
		},
		{
			name:     "missing environment variable secret",
			reason:   "missing required environment variables: [DATABASE_URL]",
			err:      nil,
			expected: FailureClassMissingEnvSecret,
		},
		{
			name:     "toolchain version mismatch",
			reason:   "error: The engine \"node\" is incompatible with this module. Expected version \">=18.0.0\". Got \"16.14.0\"",
			err:      nil,
			expected: FailureClassToolchainMismatch,
		},
		{
			name:     "health check timeout after retries",
			reason:   "health check \"db.health\" still failing after 10 attempts: connection refused",
			err:      nil,
			expected: FailureClassHealthcheckTimeout,
		},
		{
			name:     "unknown generic failure",
			reason:   "something completely unrecognized happened",
			err:      nil,
			expected: FailureClassUnknown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ClassifyExecutionFailure(tt.reason, tt.err)
			if got != tt.expected {
				t.Fatalf("ClassifyExecutionFailure() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestDiagnoseExecutionFailure(t *testing.T) {
	t.Run("PortConflictDiagnosis", func(t *testing.T) {
		diag := DiagnoseExecutionFailure("listen tcp :8080: bind: address already in use", nil)
		if diag.Code != FailurePortConflict {
			t.Fatalf("expected FailurePortConflict, got %s", diag.Code)
		}
		if diag.Class != FailureClassPortConflict {
			t.Fatalf("expected FailureClassPortConflict, got %s", diag.Class)
		}
		if diag.Port != 8080 {
			t.Fatalf("expected port 8080, got %d", diag.Port)
		}
		if diag.RootCause == "" || diag.SuggestedFix == "" {
			t.Fatalf("expected non-empty root cause and suggested fix: %+v", diag)
		}
	})

	t.Run("MissingEnvSecretDiagnosisFromError", func(t *testing.T) {
		err := &MissingRequiredEnvError{Missing: []string{"DATABASE_URL"}}
		diag := DiagnoseExecutionFailure("", err)
		if diag.Code != FailureMissingEnvSecret {
			t.Fatalf("expected FailureMissingEnvSecret, got %s", diag.Code)
		}
		if diag.MissingKey != "DATABASE_URL" {
			t.Fatalf("expected DATABASE_URL, got %q", diag.MissingKey)
		}
		if diag.RootCause == "" || diag.SuggestedFix == "" {
			t.Fatalf("expected non-empty root cause and suggested fix: %+v", diag)
		}
	})

	t.Run("DaemonUnavailableDiagnosis", func(t *testing.T) {
		diag := DiagnoseExecutionFailure("Cannot connect to the Docker daemon at unix:///var/run/docker.sock. Is the docker daemon running?", nil)
		if diag.Code != FailureDaemonUnavailable {
			t.Fatalf("expected FailureDaemonUnavailable, got %s", diag.Code)
		}
		if diag.Class != FailureClassDaemonUnavailable {
			t.Fatalf("expected FailureClassDaemonUnavailable, got %s", diag.Class)
		}
		if diag.RootCause == "" || diag.SuggestedFix == "" {
			t.Fatalf("expected non-empty root cause and suggested fix: %+v", diag)
		}
	})

	t.Run("DependencyNotInstalledBinaryDiagnosis", func(t *testing.T) {
		diag := DiagnoseExecutionFailure("exec: \"pnpm\": executable file not found in $PATH", nil)
		if diag.Code != FailureDependencyNotInstalled {
			t.Fatalf("expected FailureDependencyNotInstalled, got %s", diag.Code)
		}
		if diag.Class != FailureClassMissingDependency {
			t.Fatalf("expected FailureClassMissingDependency, got %s", diag.Class)
		}
		if diag.RootCause == "" || diag.SuggestedFix == "" {
			t.Fatalf("expected non-empty root cause and suggested fix: %+v", diag)
		}
	})

	t.Run("ToolchainMismatchDiagnosis", func(t *testing.T) {
		diag := DiagnoseExecutionFailure("error: The engine \"node\" is incompatible with this module", nil)
		if diag.Code != FailureToolchainMismatch {
			t.Fatalf("expected FailureToolchainMismatch, got %s", diag.Code)
		}
		if diag.Class != FailureClassToolchainMismatch {
			t.Fatalf("expected FailureClassToolchainMismatch, got %s", diag.Class)
		}
		if diag.RootCause == "" || diag.SuggestedFix == "" {
			t.Fatalf("expected non-empty root cause and suggested fix: %+v", diag)
		}
	})

	t.Run("HealthcheckTimeoutDiagnosis", func(t *testing.T) {
		diag := DiagnoseExecutionFailure("health check \"svc\" still failing after 10 attempts: timeout", nil)
		if diag.Code != FailureHealthcheckTimeout {
			t.Fatalf("expected FailureHealthcheckTimeout, got %s", diag.Code)
		}
		if diag.Class != FailureClassHealthcheckTimeout {
			t.Fatalf("expected FailureClassHealthcheckTimeout, got %s", diag.Class)
		}
		if diag.RootCause == "" || diag.SuggestedFix == "" {
			t.Fatalf("expected non-empty root cause and suggested fix: %+v", diag)
		}
	})
}

func TestHelperExtractors(t *testing.T) {
	if got := ExtractMissingEnvKey("missing required environment variables: [API_KEY]"); got != "API_KEY" {
		t.Fatalf("expected API_KEY, got %q", got)
	}
	if got := ExtractMissingBinary("sh: nodemon: command not found"); got != "nodemon" {
		t.Fatalf("expected nodemon, got %q", got)
	}
	if got := ExtractPortFromText("listen tcp :4000: bind: address already in use"); got != 4000 {
		t.Fatalf("expected 4000, got %d", got)
	}
	if code := MapClassificationToCode(FailureClassPortConflict); code != FailurePortConflict {
		t.Fatalf("expected FailurePortConflict, got %s", code)
	}
}

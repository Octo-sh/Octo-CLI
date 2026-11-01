package intelligence

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/Octo-sh/Octo-CLI/internal/secrets"
)

// FailureCode identifies a stable, machine-readable execution failure.
type FailureCode string

const (
	FailureNoExecutableCandidates FailureCode = "NO_EXECUTABLE_CANDIDATES"
	FailureNoRunCommand           FailureCode = "NO_RUN_COMMAND"
	FailureInvalidProject         FailureCode = "INVALID_PROJECT"
	FailureInvalidComponent       FailureCode = "INVALID_COMPONENT"
	FailureInvalidService         FailureCode = "INVALID_SERVICE"
	FailureTopologyBuild          FailureCode = "TOPOLOGY_BUILD_FAILED"
	FailureDecision               FailureCode = "DECISION_FAILED"
	FailurePlanValidation         FailureCode = "PLAN_VALIDATION_FAILED"

	// Deterministic local runtime onboarding failures:
	FailurePortConflict           FailureCode = "PORT_CONFLICT"
	FailureMissingEnvSecret       FailureCode = "MISSING_ENV_SECRET"
	FailureDaemonUnavailable      FailureCode = "DAEMON_UNAVAILABLE"
	FailureDependencyNotInstalled FailureCode = "DEPENDENCY_NOT_INSTALLED"
	FailureToolchainMismatch      FailureCode = "TOOLCHAIN_MISMATCH"
	FailureHealthcheckTimeout     FailureCode = "HEALTHCHECK_TIMEOUT"
	FailureCompilationSyntax      FailureCode = "COMPILATION_SYNTAX"
	FailureProcessCrash           FailureCode = "PROCESS_CRASH"
	FailureMissingArguments       FailureCode = "MISSING_ARGUMENTS"
	FailureTimeout                FailureCode = "TIMEOUT"
	FailureUnknown                FailureCode = "UNKNOWN"
)

// FailureReason carries structured context while keeping CLI explanations human-readable.
type FailureReason struct {
	Code      FailureCode
	Component string
	Summary   string
	Details   []string
}

func (f *FailureReason) Error() string {
	if f == nil {
		return ""
	}
	var b strings.Builder
	b.WriteString(f.Summary)
	for _, detail := range f.Details {
		if strings.TrimSpace(detail) == "" {
			continue
		}
		b.WriteString("\n  ")
		b.WriteString(detail)
	}
	return b.String()
}

// AsFailure extracts structured failure data through wrapped errors.
func AsFailure(err error) (*FailureReason, bool) {
	var failure *FailureReason
	if errors.As(err, &failure) {
		return failure, true
	}
	return nil, false
}

func newFailure(code FailureCode, component, summary string, details ...string) error {
	return &FailureReason{Code: code, Component: component, Summary: summary, Details: details}
}

// FailureClassification categorizes execution and startup failures into structured classes.
type FailureClassification string

const (
	FailureClassPortConflict       FailureClassification = "port_conflict"
	FailureClassMissingDependency  FailureClassification = "missing_dependency"
	FailureClassCompilationSyntax  FailureClassification = "compilation_syntax"
	FailureClassProcessCrash       FailureClassification = "process_crash"
	FailureClassMissingArguments   FailureClassification = "missing_arguments"
	FailureClassTimeout            FailureClassification = "timeout"
	FailureClassDaemonUnavailable  FailureClassification = "daemon_unavailable"
	FailureClassMissingEnvSecret   FailureClassification = "missing_env_secret"
	FailureClassToolchainMismatch  FailureClassification = "toolchain_mismatch"
	FailureClassHealthcheckTimeout FailureClassification = "healthcheck_timeout"
	FailureClassUnknown            FailureClassification = "unknown"
)

// FailureDiagnostic represents a structured, deterministic root-cause diagnostic with remediation suggestions.
type FailureDiagnostic struct {
	Code         FailureCode           `json:"code" yaml:"code"`
	Class        FailureClassification `json:"class,omitempty" yaml:"class,omitempty"`
	RootCause    string                `json:"root_cause" yaml:"root_cause"`
	SuggestedFix string                `json:"suggested_fix" yaml:"suggested_fix"`
	Port         int                   `json:"port,omitempty" yaml:"port,omitempty"`
	MissingKey   string                `json:"missing_key,omitempty" yaml:"missing_key,omitempty"`
}

var (
	envMissingPatterns = []*regexp.Regexp{
		regexp.MustCompile(`missing required environment variables?:\s*\[?([A-Z0-9_]+)\]?`),
		regexp.MustCompile(`(?:environment variable|secret|variable)\s+['"]?([A-Z0-9_]{3,})['"]?\s+is (?:not set|required|missing)`),
		regexp.MustCompile(`KeyError:\s*['"]([A-Z0-9_]{3,})['"]`),
		regexp.MustCompile(`process\.env\.([A-Z0-9_]{3,})\b`),
		regexp.MustCompile(`process\.env\[['"]([A-Z0-9_]{3,})['"]\]`),
		regexp.MustCompile(`(?:missing|unset)\s+(?:required\s+)?([A-Z0-9_]{3,})\b`),
	}

	binMissingPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?:exec|executable file|binary):\s*["']?([a-zA-Z0-9_.-]+)["']?:\s*(?:executable file not found|not found)`),
		regexp.MustCompile(`(?:command not found|not found):\s*([a-zA-Z0-9_.-]+)`),
		regexp.MustCompile(`(?:sh|bash|zsh):\s*([a-zA-Z0-9_.-]+):\s*command not found`),
		regexp.MustCompile(`^([a-zA-Z0-9_.-]+):\s*(?:command )?not found`),
	}
)

// ExtractMissingEnvKey extracts a missing environment variable name from a failure message.
func ExtractMissingEnvKey(text string) string {
	if text == "" {
		return ""
	}
	for _, re := range envMissingPatterns {
		matches := re.FindStringSubmatch(text)
		if len(matches) >= 2 && matches[1] != "" {
			return matches[1]
		}
	}
	return ""
}

// ExtractMissingBinary extracts a missing executable name from command output.
func ExtractMissingBinary(text string) string {
	if text == "" {
		return ""
	}
	for _, re := range binMissingPatterns {
		matches := re.FindStringSubmatch(text)
		if len(matches) >= 2 && matches[1] != "" {
			return matches[1]
		}
	}
	return ""
}

// MapClassificationToCode maps a FailureClassification to its corresponding FailureCode.
func MapClassificationToCode(c FailureClassification) FailureCode {
	switch c {
	case FailureClassPortConflict:
		return FailurePortConflict
	case FailureClassMissingEnvSecret:
		return FailureMissingEnvSecret
	case FailureClassDaemonUnavailable:
		return FailureDaemonUnavailable
	case FailureClassMissingDependency:
		return FailureDependencyNotInstalled
	case FailureClassToolchainMismatch:
		return FailureToolchainMismatch
	case FailureClassHealthcheckTimeout:
		return FailureHealthcheckTimeout
	case FailureClassCompilationSyntax:
		return FailureCompilationSyntax
	case FailureClassProcessCrash:
		return FailureProcessCrash
	case FailureClassMissingArguments:
		return FailureMissingArguments
	case FailureClassTimeout:
		return FailureTimeout
	default:
		return FailureUnknown
	}
}

// DiagnoseExecutionFailure inspects failure messages and errors to determine the root cause,
// structured failure code, and actionable remediation command.
func DiagnoseExecutionFailure(reason string, err error) FailureDiagnostic {
	var missingEnvErr *MissingRequiredEnvError
	if errors.As(err, &missingEnvErr) {
		return missingEnvErr.FailureDiagnostic()
	}

	msg := reason
	if err != nil {
		if msg == "" {
			msg = err.Error()
		} else {
			msg = msg + " " + err.Error()
		}
	}
	lower := strings.ToLower(msg)

	// 1. Daemon unavailable (Docker/Podman daemon not running)
	if strings.Contains(lower, "cannot connect to the docker daemon") ||
		strings.Contains(lower, "is the docker daemon running") ||
		strings.Contains(lower, "docker daemon is not running") ||
		strings.Contains(lower, "failed to connect to docker daemon") ||
		strings.Contains(lower, "pipe/docker_engine") ||
		strings.Contains(lower, "unable to connect to podman") ||
		strings.Contains(lower, "podman service is not running") ||
		strings.Contains(lower, "docker desktop is not running") ||
		strings.Contains(lower, "failed to initialize container runtime") ||
		(strings.Contains(lower, "connection refused") && (strings.Contains(lower, "docker.sock") || strings.Contains(lower, "docker"))) {
		return FailureDiagnostic{
			Code:         FailureDaemonUnavailable,
			Class:        FailureClassDaemonUnavailable,
			RootCause:    "Docker daemon is not running or unreachable, but container/compose services are required.",
			SuggestedFix: "Start Docker Desktop or start the container daemon ('sudo systemctl start docker') and retry.",
		}
	}

	// 2. Missing environment secrets / configuration
	if strings.Contains(lower, "missing required environment variable") ||
		strings.Contains(lower, "missing required env") ||
		strings.Contains(lower, "required environment variables: [") ||
		(strings.Contains(lower, "environment variable") && strings.Contains(lower, "is not set")) ||
		(strings.Contains(lower, "is not set") && (strings.Contains(lower, "secret") || strings.Contains(lower, "key") || strings.Contains(lower, "token") || strings.Contains(lower, "database_url"))) {
		missingKey := ExtractMissingEnvKey(msg)
		if missingKey != "" {
			placeholder := placeholderFor(missingKey, ProjectModel{})
			return FailureDiagnostic{
				Code:         FailureMissingEnvSecret,
				Class:        FailureClassMissingEnvSecret,
				RootCause:    fmt.Sprintf("Required environment secret/variable %q is missing from local configuration.", missingKey),
				SuggestedFix: fmt.Sprintf("Run 'octo env init' or set %s=%s in your .env file.", missingKey, placeholder),
				MissingKey:   missingKey,
			}
		}
		return FailureDiagnostic{
			Code:         FailureMissingEnvSecret,
			Class:        FailureClassMissingEnvSecret,
			RootCause:    "Required environment secret(s) are missing from local configuration.",
			SuggestedFix: "Run 'octo env init' to initialize missing configuration or run with --skip-env to bypass validation.",
		}
	}

	// 3. Toolchain & runtime version mismatch
	if strings.Contains(lower, "incompatible with this module") ||
		strings.Contains(lower, "requires node version") ||
		strings.Contains(lower, "unsupported engine") ||
		strings.Contains(lower, "incompatible runtime version") ||
		strings.Contains(lower, "toolchain mismatch") ||
		strings.Contains(lower, "unsupported runtime") ||
		strings.Contains(lower, "version mismatch") ||
		(strings.Contains(lower, "requires go") && (strings.Contains(lower, ">=") || strings.Contains(lower, "version"))) ||
		(strings.Contains(lower, "requires python") && (strings.Contains(lower, ">=") || strings.Contains(lower, "version"))) {
		return FailureDiagnostic{
			Code:         FailureToolchainMismatch,
			Class:        FailureClassToolchainMismatch,
			RootCause:    "Local runtime or toolchain version does not satisfy project prerequisites.",
			SuggestedFix: "Switch to a compatible runtime version using your version manager (e.g., 'nvm use', 'pyenv local', or update Go).",
		}
	}

	// 4. Port conflicts
	if strings.Contains(lower, "address already in use") ||
		strings.Contains(lower, "port is not reachable") ||
		(strings.Contains(lower, "not reachable") && strings.Contains(lower, "port")) ||
		strings.Contains(lower, "bind: address already in use") ||
		strings.Contains(lower, "port conflict") ||
		strings.Contains(lower, "port collision") ||
		strings.Contains(lower, "eaddrinuse") ||
		(strings.Contains(lower, "requested tcp port") && strings.Contains(lower, "unavailable")) ||
		(strings.Contains(lower, "startup verification failed") && strings.Contains(lower, "port")) {
		port := ExtractPortFromText(msg)
		if port > 0 {
			nextPort := FindNextFreePort(port+1, nil)
			return FailureDiagnostic{
				Code:         FailurePortConflict,
				Class:        FailureClassPortConflict,
				RootCause:    fmt.Sprintf("TCP port %d is already in use by another process or strictly declared.", port),
				SuggestedFix: fmt.Sprintf("Free port %d (e.g., 'lsof -i :%d' or 'kill <pid>'), or run with port override: octo run --port %d", port, port, nextPort),
				Port:         port,
			}
		}
		return FailureDiagnostic{
			Code:         FailurePortConflict,
			Class:        FailureClassPortConflict,
			RootCause:    "A required TCP port is already in use by another process.",
			SuggestedFix: "Free the conflicting port or run 'octo run' without '--no-port-shift' to let Octo allocate an alternative port.",
		}
	}

	// 5. Healthcheck timeout (service started but health check failed / timed out)
	if (strings.Contains(lower, "health check") && strings.Contains(lower, "still failing after")) ||
		strings.Contains(lower, "healthcheck timeout") ||
		strings.Contains(lower, "health check timed out") ||
		strings.Contains(lower, "health readiness check timed out") ||
		(strings.Contains(lower, "health check") && strings.Contains(lower, "failed")) {
		return FailureDiagnostic{
			Code:         FailureHealthcheckTimeout,
			Class:        FailureClassHealthcheckTimeout,
			RootCause:    "Service process started but failed to become healthy before the verification deadline expired.",
			SuggestedFix: "Inspect service logs in .octo/logs/ to check for initialization delays or errors, or increase healthcheck timeout in octo.yaml.",
		}
	}

	// 6. Missing dependencies & missing executables
	if strings.Contains(lower, "cannot find module") ||
		strings.Contains(lower, "modulenotfounderror") ||
		strings.Contains(lower, "no module named") ||
		strings.Contains(lower, "python3-venv") ||
		strings.Contains(lower, "venv module") ||
		strings.Contains(lower, "ensurepip is not available") ||
		strings.Contains(lower, "package not found") ||
		strings.Contains(lower, "command not found") ||
		strings.Contains(lower, "not installed") ||
		strings.Contains(lower, "executable file not found") ||
		strings.Contains(lower, "missing dependency") ||
		strings.Contains(lower, "could not find a declaration file") ||
		strings.Contains(lower, "import error") ||
		strings.Contains(lower, "importerror") ||
		(strings.Contains(lower, "no such file or directory") && (strings.Contains(lower, "bin") || strings.Contains(lower, "node_modules"))) {
		tool := ExtractMissingBinary(msg)
		if tool != "" {
			var fix string
			switch strings.ToLower(tool) {
			case "pnpm":
				fix = "Install pnpm (e.g., 'npm install -g pnpm') and verify it is available in your PATH."
			case "yarn":
				fix = "Install Yarn (e.g., 'npm install -g yarn') and verify it is available in your PATH."
			case "docker":
				fix = "Install Docker Desktop or the docker CLI and verify it is in your PATH."
			case "mvn":
				fix = "Install Apache Maven (e.g., 'brew install maven' or 'apt install maven') and verify it is in your PATH."
			case "gradle":
				fix = "Install Gradle (e.g., 'brew install gradle') or use the project's ./gradlew wrapper."
			default:
				fix = fmt.Sprintf("Install %s and verify it is available in your system PATH.", tool)
			}
			return FailureDiagnostic{
				Code:         FailureDependencyNotInstalled,
				Class:        FailureClassMissingDependency,
				RootCause:    fmt.Sprintf("Required executable %q is not installed or not found in PATH.", tool),
				SuggestedFix: fix,
			}
		}
		if strings.Contains(lower, "python3-venv") || strings.Contains(lower, "venv module") || strings.Contains(lower, "ensurepip") {
			return FailureDiagnostic{
				Code:         FailureDependencyNotInstalled,
				Class:        FailureClassMissingDependency,
				RootCause:    "Python virtual environment module (venv) is not installed.",
				SuggestedFix: "Install python3-venv (e.g., 'sudo apt install python3-venv') or create a virtualenv.",
			}
		}
		if strings.Contains(lower, "modulenotfounderror") || strings.Contains(lower, "no module named") {
			return FailureDiagnostic{
				Code:         FailureDependencyNotInstalled,
				Class:        FailureClassMissingDependency,
				RootCause:    "Python package dependencies are not installed in the active environment.",
				SuggestedFix: "Install dependencies with 'pip install -r requirements.txt' or 'poetry install'.",
			}
		}
		return FailureDiagnostic{
			Code:         FailureDependencyNotInstalled,
			Class:        FailureClassMissingDependency,
			RootCause:    "Required package dependencies are not installed in the local environment.",
			SuggestedFix: "Install project dependencies using your package manager (e.g., 'npm install', 'pnpm install', 'pip install -r requirements.txt', or 'go mod download').",
		}
	}

	// 7. Compilation & syntax errors
	if strings.Contains(lower, "syntaxerror") ||
		strings.Contains(lower, "syntax error") ||
		strings.Contains(lower, "compilation error") ||
		strings.Contains(lower, "compile error") ||
		strings.Contains(lower, "build failed") ||
		strings.Contains(lower, "typeerror") ||
		strings.Contains(lower, "type error") ||
		strings.Contains(lower, "undefined reference") ||
		strings.Contains(lower, "cannot find symbol") ||
		strings.Contains(lower, "parse error") ||
		strings.Contains(lower, "parseerror") {
		return FailureDiagnostic{
			Code:         FailureCompilationSyntax,
			Class:        FailureClassCompilationSyntax,
			RootCause:    "Code compilation or syntax error prevented application execution.",
			SuggestedFix: "Fix syntax, type, or compilation errors in your source files before restarting.",
		}
	}

	// 8. Timeouts & deadline exceeded
	if strings.Contains(lower, "deadline exceeded") ||
		strings.Contains(lower, "timed out") ||
		strings.Contains(lower, "timeout") {
		return FailureDiagnostic{
			Code:         FailureTimeout,
			Class:        FailureClassTimeout,
			RootCause:    "Operation exceeded the allotted execution time limit.",
			SuggestedFix: "Increase timeout duration or check if system resources are constrained.",
		}
	}

	// 9. Missing CLI arguments
	if strings.Contains(lower, "required arguments were not provided") ||
		strings.Contains(lower, "the following required arguments were not provided") ||
		strings.Contains(lower, "missing required argument") ||
		(strings.Contains(lower, "usage:") && strings.Contains(lower, "argument")) {
		return FailureDiagnostic{
			Code:         FailureMissingArguments,
			Class:        FailureClassMissingArguments,
			RootCause:    "Application executable requires arguments to run.",
			SuggestedFix: "Pass arguments using passthrough: octo run -- [arguments...]",
		}
	}

	// 10. Process crashes & abrupt exits
	if strings.Contains(lower, "exit status") ||
		strings.Contains(lower, "segmentation fault") ||
		strings.Contains(lower, "sigsegv") ||
		strings.Contains(lower, "signal: killed") ||
		strings.Contains(lower, "signal: terminated") ||
		strings.Contains(lower, "crashed") ||
		strings.Contains(lower, "panic:") {
		return FailureDiagnostic{
			Code:         FailureProcessCrash,
			Class:        FailureClassProcessCrash,
			RootCause:    "Application process exited prematurely or crashed during startup.",
			SuggestedFix: "Check application logs in .octo/logs/ to review the crash trace or stack dump.",
		}
	}

	return FailureDiagnostic{
		Code:         FailureUnknown,
		Class:        FailureClassUnknown,
		RootCause:    "Execution failed due to an unclassified error.",
		SuggestedFix: "Inspect logs in .octo/logs/ and check process output for troubleshooting details.",
	}
}

// ClassifyExecutionFailure inspects failure messages and errors to determine the failure category.
func ClassifyExecutionFailure(reason string, err error) FailureClassification {
	return DiagnoseExecutionFailure(reason, err).Class
}

// Silence unused variable warning for secrets package import if only used conceptually
var _ = secrets.IsCriticalEnvVar

package intelligence

import "context"

// EvidenceKind identifies where an inference came from.
type EvidenceKind string

const (
	EvidenceSignalFile EvidenceKind = "signal_file"
	EvidenceLockfile   EvidenceKind = "lockfile"
	EvidenceManifest   EvidenceKind = "manifest"
	EvidenceScript     EvidenceKind = "script"
	EvidenceConfig     EvidenceKind = "config"
	EvidenceExecutionFailure EvidenceKind = "execution_failure"
)

// RelationshipKind identifies the semantics of a topology relationship.
// Execution engines MUST treat only DependsOn as an execution-order edge.
type RelationshipKind string

const (
	RelationshipDependsOn       RelationshipKind = "depends_on"
	RelationshipNetworkReference RelationshipKind = "network_reference"
)

func (k RelationshipKind) IsKnown() bool {
	switch k {
	case RelationshipDependsOn, RelationshipNetworkReference:
		return true
	default:
		return false
	}
}

type Evidence struct {
	Kind EvidenceKind `json:"kind" yaml:"kind"`
	Path string `json:"path" yaml:"path"`
	Detail string `json:"detail" yaml:"detail"`
	Strength float64 `json:"strength" yaml:"strength"`
}

type Reference struct {
	Target string `json:"target" yaml:"target"`
	Kind RelationshipKind `json:"kind" yaml:"kind"`
	Variable string `json:"variable,omitempty" yaml:"variable,omitempty"`
	Confidence float64 `json:"confidence" yaml:"confidence"`
	Evidence []Evidence `json:"evidence,omitempty" yaml:"evidence,omitempty"`
}

type HealthCheck struct {
	Command string `json:"command,omitempty" yaml:"command,omitempty"`
	Interval string `json:"interval,omitempty" yaml:"interval,omitempty"`
	Timeout string `json:"timeout,omitempty" yaml:"timeout,omitempty"`
	Retries int `json:"retries,omitempty" yaml:"retries,omitempty"`
}

type Service struct {
	Name string `json:"name" yaml:"name"`
	Image string `json:"image,omitempty" yaml:"image,omitempty"`
	Build string `json:"build,omitempty" yaml:"build,omitempty"`
	DependsOn []string `json:"depends_on,omitempty" yaml:"depends_on,omitempty"`
	References []Reference `json:"references,omitempty" yaml:"references,omitempty"`
	Ports []string `json:"ports,omitempty" yaml:"ports,omitempty"`
	HealthCheck *HealthCheck `json:"health_check,omitempty" yaml:"health_check,omitempty"`
	Confidence float64 `json:"confidence" yaml:"confidence"`
	Evidence []Evidence `json:"evidence,omitempty" yaml:"evidence,omitempty"`
}

type Component struct {
	Name string `json:"name" yaml:"name"`
	ExecutionCandidates []ExecutionCandidate `json:"execution_candidates,omitempty" yaml:"execution_candidates,omitempty"`
	Path string `json:"path" yaml:"path"`
	Language string `json:"language,omitempty" yaml:"language,omitempty"`
	Framework string `json:"framework,omitempty" yaml:"framework,omitempty"`
	PackageManager string `json:"package_manager,omitempty" yaml:"package_manager,omitempty"`
	RunCommand string `json:"run_command,omitempty" yaml:"run_command,omitempty"`
	DependsOn []string `json:"depends_on,omitempty" yaml:"depends_on,omitempty"`
	References []Reference `json:"references,omitempty" yaml:"references,omitempty"`
	Port int `json:"port,omitempty" yaml:"port,omitempty"`
	PortStrict bool `json:"port_strict,omitempty" yaml:"port_strict,omitempty"`
	Confidence float64 `json:"confidence" yaml:"confidence"`
	Evidence []Evidence `json:"evidence,omitempty" yaml:"evidence,omitempty"`
	Role string `json:"role,omitempty" yaml:"role,omitempty"`
	NonRunnableReason string `json:"non_runnable_reason,omitempty" yaml:"non_runnable_reason,omitempty"`
}

type EnvironmentVariable struct {
	Name string `json:"name" yaml:"name"`
	Required bool `json:"required" yaml:"required"`
	Sources []string `json:"sources,omitempty" yaml:"sources,omitempty"`
}

type EnvironmentResolution struct {
	Variable string `json:"variable" yaml:"variable"`
	Source string `json:"source" yaml:"source"`
	Target string `json:"target" yaml:"target"`
	Kind RelationshipKind `json:"kind" yaml:"kind"`
	Confidence float64 `json:"confidence" yaml:"confidence"`
	Evidence []Evidence `json:"evidence,omitempty" yaml:"evidence,omitempty"`
}

type EnvironmentModel struct {
	Variables []EnvironmentVariable `json:"variables,omitempty" yaml:"variables,omitempty"`
	Resolutions []EnvironmentResolution `json:"resolutions,omitempty" yaml:"resolutions,omitempty"`
}

type ProjectModel struct {
	Name string `json:"name" yaml:"name"`
	Root string `json:"root" yaml:"root"`
	Language string `json:"language,omitempty" yaml:"language,omitempty"`
	RuntimeVersion string `json:"runtime_version,omitempty" yaml:"runtime_version,omitempty"`
	Framework string `json:"framework,omitempty" yaml:"framework,omitempty"`
	PackageManager string `json:"package_manager,omitempty" yaml:"package_manager,omitempty"`
	RunCommand string `json:"run_command,omitempty" yaml:"run_command,omitempty"`
	SetupCommand string `json:"setup_command,omitempty" yaml:"setup_command,omitempty"`
	Monorepo bool `json:"monorepo" yaml:"monorepo"`
	Port int `json:"port,omitempty" yaml:"port,omitempty"`
	Confidence float64 `json:"confidence" yaml:"confidence"`
	Components []Component `json:"components,omitempty" yaml:"components,omitempty"`
	Services []Service `json:"services,omitempty" yaml:"services,omitempty"`
	Environment EnvironmentModel `json:"environment,omitempty" yaml:"environment,omitempty"`
	Evidence []Evidence `json:"evidence,omitempty" yaml:"evidence,omitempty"`
	CustomVerification []VerificationCheck `json:"custom_verification,omitempty" yaml:"custom_verification,omitempty"`
	Diagnostics []string `json:"diagnostics,omitempty" yaml:"diagnostics,omitempty"`
}

// --- Core Intelligence Pipeline Interfaces ---

// Analyzer performs static discovery and topology analysis on a repository without executing code or spawning OS processes.
type Analyzer interface {
	Analyze(path string) (ProjectModel, error)
}

// Runner executes an ExecutionPlan using runtime adapters, managing process lifecycles and multiplexed logs.
type Runner interface {
	Run(ctx context.Context, model ProjectModel, plan ExecutionPlan, env ResolvedEnvironment, opts ExecutionOptions) ExecutionReport
}

// Verifier deterministically verifies service readiness and health probes (TCP, HTTP, process liveness).
type Verifier interface {
	Verify(ctx context.Context, checks []VerificationCheck) ([]VerificationResult, error)
}

// StaticAnalyzer implements Analyzer using evidence-backed static analysis.
type StaticAnalyzer struct{}

func (StaticAnalyzer) Analyze(path string) (ProjectModel, error) {
	return Analyze(path)
}

// DefaultAnalyzer returns the standard StaticAnalyzer.
func DefaultAnalyzer() Analyzer {
	return StaticAnalyzer{}
}

// DefaultPlanner returns the standard DeterministicPlanner.
func DefaultPlanner() Planner {
	return DeterministicPlanner{}
}

// PlanRunner implements Runner by delegating to runtime adapters and execution resolvers.
type PlanRunner struct {
	Resolver RuntimeResolver
}

func NewPlanRunner(resolver RuntimeResolver) *PlanRunner {
	return &PlanRunner{Resolver: resolver}
}

func (r PlanRunner) Run(ctx context.Context, model ProjectModel, plan ExecutionPlan, env ResolvedEnvironment, opts ExecutionOptions) ExecutionReport {
	resolver := r.Resolver
	if len(resolver.adapters) == 0 {
		resolver = NewRuntimeResolver()
	}
	return ExecutePlanReportWithOptions(ctx, model, plan, resolver, env, opts)
}

// DefaultRunner returns the standard PlanRunner.
func DefaultRunner() Runner {
	return PlanRunner{Resolver: NewRuntimeResolver()}
}

// ReadinessVerifier implements Verifier using deterministic protocol checks.
type ReadinessVerifier struct{}

func (ReadinessVerifier) Verify(ctx context.Context, checks []VerificationCheck) ([]VerificationResult, error) {
	return Verify(ctx, checks)
}

// DefaultVerifier returns the standard ReadinessVerifier.
func DefaultVerifier() Verifier {
	return ReadinessVerifier{}
}

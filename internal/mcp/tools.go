package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/Octo-sh/Octo-CLI/internal/benchmark"
	"github.com/Octo-sh/Octo-CLI/internal/intelligence"
	"github.com/Octo-sh/Octo-CLI/internal/secrets"
)

func DefaultTools() []Tool {
	return []Tool{
		{
			Name:        "octo_inspect",
			Description: "Analyze a repository to detect language, framework, runtime version, package manager, services, and execution confidence.",
			InputSchema: ToolInputSchema{
				Type: "object",
				Properties: map[string]Property{
					"path": {
						Type:        "string",
						Description: "Path to the repository directory (defaults to current working directory).",
					},
				},
			},
		},
		{
			Name:        "octo_topology",
			Description: "Return the validated dependency and relationship graph of components, infrastructure services, and cross-service references.",
			InputSchema: ToolInputSchema{
				Type: "object",
				Properties: map[string]Property{
					"path": {
						Type:        "string",
						Description: "Path to the repository directory (defaults to current working directory).",
					},
				},
			},
		},
		{
			Name:        "octo_plan",
			Description: "Generate the deterministic multi-step execution plan and decision trace for a repository without executing it.",
			InputSchema: ToolInputSchema{
				Type: "object",
				Properties: map[string]Property{
					"path": {
						Type:        "string",
						Description: "Path to the repository directory (defaults to current working directory).",
					},
					"only": {
						Type:        "array",
						Description: "Optional list of target selectors to execute (e.g. ['component:web', 'service:postgres']).",
						Items: &Property{
							Type: "string",
						},
					},
					"all": {
						Type:        "boolean",
						Description: "Run all detected runnable targets without selection.",
					},
				},
			},
		},
		{
			Name:        "octo_run_and_verify",
			Description: "Execute the deterministic plan for a repository, verifying startup and health, and capturing active processes with strict process-group teardown.",
			InputSchema: ToolInputSchema{
				Type: "object",
				Properties: map[string]Property{
					"path": {
						Type:        "string",
						Description: "Path to the repository directory (defaults to current working directory).",
					},
					"detach": {
						Type:        "boolean",
						Description: "Run long-running services in the background (detached mode).",
					},
					"sandbox": {
						Type:        "boolean",
						Description: "Run plan steps inside an ephemeral container (Docker or Podman) for environment isolation.",
					},
					"timeout": {
						Type:        "integer",
						Description: "Optional timeout in seconds for execution and verification (default: 30s in attached mode, 60s in detached mode).",
					},
					"only": {
						Type:        "array",
						Description: "Optional list of target selectors to execute (e.g. ['component:web', 'service:postgres']).",
						Items: &Property{
							Type: "string",
						},
					},
					"all": {
						Type:        "boolean",
						Description: "Run all detected runnable targets without selection.",
					},
				},
			},
		},
		{
			Name:        "octo_verify",
			Description: "Verify expected runtime state (ports, healthchecks) with bounded timeouts and structured failure diagnostics, inspecting .octo.lock verified state.",
			InputSchema: ToolInputSchema{
				Type: "object",
				Properties: map[string]Property{
					"path": {
						Type:        "string",
						Description: "Path to the repository directory (defaults to current working directory).",
					},
					"timeout": {
						Type:        "integer",
						Description: "Optional timeout in seconds for runtime verification checks (default: 10s).",
					},
				},
			},
		},
		{
			Name:        "octo_diagnose",
			Description: "Explain the evidence behind decisions and analyze failures for a repository.",
			InputSchema: ToolInputSchema{
				Type: "object",
				Properties: map[string]Property{
					"path": {
						Type:        "string",
						Description: "Path to the repository directory (defaults to current working directory).",
					},
				},
			},
		},
		{
			Name:        "octo_env_check",
			Description: "Validate required vs. missing environment variables against detected secrets, template definitions, and cross-service bindings.",
			InputSchema: ToolInputSchema{
				Type: "object",
				Properties: map[string]Property{
					"path": {
						Type:        "string",
						Description: "Path to the repository directory (defaults to current working directory).",
					},
				},
			},
		},
		{
			Name:        "octo_env",
			Description: "Alias for octo_env_check: Inspect present and missing environment variables, and generate safe non-secret configuration templates.",
			InputSchema: ToolInputSchema{
				Type: "object",
				Properties: map[string]Property{
					"path": {
						Type:        "string",
						Description: "Path to the repository directory (defaults to current working directory).",
					},
				},
			},
		},
		{
			Name:        "octo_preview",
			Description: "Preview all machine changes, filesystem mutations, port allocations, and processes before execution.",
			InputSchema: ToolInputSchema{
				Type: "object",
				Properties: map[string]Property{
					"path": {
						Type:        "string",
						Description: "Path to the repository directory (defaults to current working directory).",
					},
				},
			},
		},
		{
			Name:        "octo_benchmark",
			Description: "Evaluate and benchmark autonomous AI agent execution metrics (pass@1, hallucination rate, token savings, verified readiness) for a repository or canonical archetype suite.",
			InputSchema: ToolInputSchema{
				Type: "object",
				Properties: map[string]Property{
					"path": {
						Type:        "string",
						Description: "Path to the repository directory (defaults to current working directory).",
					},
					"suite": {
						Type:        "boolean",
						Description: "Whether to run the canonical SWE-bench archetype benchmark suite across multiple repository fixtures.",
					},
				},
			},
		},
	}
}

func ExecuteTool(ctx context.Context, name string, args map[string]interface{}) (CallToolResult, error) {
	switch name {
	case "octo_inspect":
		return handleInspect(ctx, args)
	case "octo_topology":
		return handleTopology(ctx, args)
	case "octo_plan":
		return handlePlan(ctx, args)
	case "octo_run_and_verify":
		return handleRunAndVerify(ctx, args)
	case "octo_verify":
		return handleVerify(ctx, args)
	case "octo_diagnose":
		return handleDiagnose(ctx, args)
	case "octo_env_check", "octo_env":
		return handleEnvCheck(ctx, args)
	case "octo_preview":
		return handlePreview(ctx, args)
	case "octo_benchmark":
		return handleBenchmark(ctx, args)
	default:
		return CallToolResult{
			IsError: true,
			Content: []ContentItem{{
				Type: "text",
				Text: fmt.Sprintf("unknown tool %q", name),
			}},
		}, nil
	}
}

func handleInspect(ctx context.Context, args map[string]interface{}) (CallToolResult, error) {
	target, err := resolvePath(args)
	if err != nil {
		return toolError(err.Error()), nil
	}

	model, err := intelligence.Analyze(target)
	if err != nil {
		return toolError(fmt.Sprintf("analysis failed: %v", err)), nil
	}

	data, err := json.MarshalIndent(model, "", "  ")
	if err != nil {
		return toolError(fmt.Sprintf("failed to serialize model: %v", err)), nil
	}

	return CallToolResult{
		Content: []ContentItem{{
			Type: "text",
			Text: string(data),
		}},
	}, nil
}

func parseSelectionOptions(args map[string]interface{}) (intelligence.SelectionOptions, error) {
	opts := intelligence.SelectionOptions{}
	if allVal, ok := args["all"].(bool); ok {
		opts.All = allVal
	}

	if onlyVal, ok := args["only"]; ok && onlyVal != nil {
		switch v := onlyVal.(type) {
		case []string:
			opts.Only = v
		case []interface{}:
			for _, item := range v {
				if s, ok := item.(string); ok {
					opts.Only = append(opts.Only, s)
				} else {
					return opts, fmt.Errorf("invalid 'only' selector: expected string, got %T", item)
				}
			}
		case string:
			if strings.TrimSpace(v) != "" {
				opts.Only = []string{strings.TrimSpace(v)}
			}
		default:
			return opts, fmt.Errorf("invalid 'only' argument: expected array of strings, got %T", onlyVal)
		}
	}

	if len(opts.Only) > 0 && opts.All {
		return opts, fmt.Errorf("cannot combine 'only' and 'all'; choose either specific targets or 'all'")
	}

	return opts, nil
}

type PlanResult struct {
	intelligence.ExecutionPlan
	Decisions []intelligence.DecisionTrace `json:"decisions,omitempty"`
}

func handlePlan(ctx context.Context, args map[string]interface{}) (CallToolResult, error) {
	target, err := resolvePath(args)
	if err != nil {
		return toolError(err.Error()), nil
	}

	model, err := intelligence.Analyze(target)
	if err != nil {
		return toolError(fmt.Sprintf("analysis failed: %v", err)), nil
	}

	selOpts, err := parseSelectionOptions(args)
	if err != nil {
		return toolError(err.Error()), nil
	}

	selection, err := intelligence.ResolveSelection(model, selOpts)
	if err != nil {
		return toolError(err.Error()), nil
	}
	model = selection.FilteredModel

	planner := intelligence.DeterministicPlanner{}
	plan, err := planner.Plan(ctx, model)
	if err != nil {
		return toolError(fmt.Sprintf("planning failed: %v", err)), nil
	}

	decisions := intelligence.BuildDecisionTrace(model, plan)
	result := PlanResult{
		ExecutionPlan: plan,
		Decisions:     decisions,
	}

	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return toolError(fmt.Sprintf("failed to serialize plan: %v", err)), nil
	}

	return CallToolResult{
		Content: []ContentItem{{
			Type: "text",
			Text: string(data),
		}},
	}, nil
}

func handleRunAndVerify(ctx context.Context, args map[string]interface{}) (CallToolResult, error) {
	target, err := resolvePath(args)
	if err != nil {
		return toolError(err.Error()), nil
	}

	detach := false
	if d, ok := args["detach"].(bool); ok {
		detach = d
	}

	// Bounded execution timeout for agent safety (default: 30s in attached mode, 60s in detached mode)
	timeoutSec := 30
	if detach {
		timeoutSec = 60
	}
	if tVal, ok := args["timeout"].(float64); ok && tVal > 0 {
		timeoutSec = int(tVal)
	} else if tVal, ok := args["timeout"].(int); ok && tVal > 0 {
		timeoutSec = tVal
	}

	execCtx, cancel := context.WithTimeout(ctx, time.Duration(timeoutSec)*time.Second)
	defer cancel()

	model, err := intelligence.Analyze(target)
	if err != nil {
		return toolError(fmt.Sprintf("analysis failed: %v", err)), nil
	}

	lock, err := intelligence.LoadOctoLock(target)
	if err != nil {
		return toolError(fmt.Sprintf("failed to read lockfile: %v", err)), nil
	}

	_, _ = intelligence.ApplyVerifiedStrategies(target, &model, lock)

	selOpts, err := parseSelectionOptions(args)
	if err != nil {
		return toolError(err.Error()), nil
	}

	selection, err := intelligence.ResolveSelection(model, selOpts)
	if err != nil {
		return toolError(err.Error()), nil
	}
	model = selection.FilteredModel

	planner := intelligence.DeterministicPlanner{}
	plan, err := planner.Plan(execCtx, model)
	if err != nil {
		return toolError(fmt.Sprintf("planning failed: %v", err)), nil
	}

	env, err := intelligence.ResolveProjectEnvironment(target, model)
	if err != nil {
		return toolError(fmt.Sprintf("environment resolution failed: %v", err)), nil
	}

	sandbox, _ := args["sandbox"].(bool)
	opts := intelligence.ExecutionOptions{
		Detach:  detach,
		Silent:  true,
		Sandbox: sandbox,
		// Record the lock at the same verified-startup checkpoint as the CLI.
		// A persistence failure fails the run instead of being swallowed.
		OnVerifiedStartup: func(rep intelligence.ExecutionReport) error {
			updated, err := intelligence.RecordVerifiedStrategiesWithLock(target, model, plan, rep, lock)
			if err != nil {
				return err
			}
			lock = updated
			return nil
		},
	}

	resolver := intelligence.NewRuntimeResolverWithOptions(intelligence.RuntimeResolverOptions{
		Sandbox: sandbox,
		Root:    target,
	})
	report := intelligence.ExecutePlanReportWithOptions(execCtx, model, plan, resolver, env, opts)

	if report.Success && detach {
		if err := intelligence.SaveDetachedState(target, report); err != nil {
			report.Success = false
			report.FailureReason = fmt.Sprintf("failed to save detached state: %v", err)
		}
	} else if !detach && len(report.ActiveProcesses) > 0 {
		// In attached/smoke-check verification mode, tear down active processes cleanly
		// using process groups so no background processes or child processes leak.
		teardownOpts := opts
		teardownOpts.OwnedServices = report.OwnedServices
		_ = intelligence.TeardownExecution(model, plan, report.RunningHandles(), teardownOpts)
	}

	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return toolError(fmt.Sprintf("failed to serialize report: %v", err)), nil
	}

	return CallToolResult{
		IsError: !report.Success,
		Content: []ContentItem{{
			Type: "text",
			Text: string(data),
		}},
	}, nil
}

type DiagnosisResult struct {
	ProjectName     string                         `json:"project_name"`
	Language        string                         `json:"language"`
	Framework       string                         `json:"framework,omitempty"`
	Confidence      float64                        `json:"confidence"`
	Decisions       []intelligence.DecisionTrace   `json:"decisions"`
	Evidence        []intelligence.Evidence        `json:"evidence"`
	ActiveProcesses []intelligence.ActiveProcess   `json:"active_processes,omitempty"`
}

func handleDiagnose(ctx context.Context, args map[string]interface{}) (CallToolResult, error) {
	target, err := resolvePath(args)
	if err != nil {
		return toolError(err.Error()), nil
	}

	model, err := intelligence.Analyze(target)
	if err != nil {
		return toolError(fmt.Sprintf("analysis failed: %v", err)), nil
	}

	planner := intelligence.DeterministicPlanner{}
	plan, err := planner.Plan(ctx, model)
	if err != nil {
		return toolError(fmt.Sprintf("planning failed: %v", err)), nil
	}

	decisions := intelligence.BuildDecisionTrace(model, plan)
	diagnosis := DiagnosisResult{
		ProjectName: model.Name,
		Language:    model.Language,
		Framework:   model.Framework,
		Confidence:  model.Confidence,
		Decisions:   decisions,
		Evidence:    model.Evidence,
	}

	data, err := json.MarshalIndent(diagnosis, "", "  ")
	if err != nil {
		return toolError(fmt.Sprintf("failed to serialize diagnosis: %v", err)), nil
	}

	return CallToolResult{
		Content: []ContentItem{{
			Type: "text",
			Text: string(data),
		}},
	}, nil
}

type TopologyResult struct {
	intelligence.TopologyGraph
	Components []intelligence.Component             `json:"components,omitempty"`
	Services   []intelligence.Service               `json:"services,omitempty"`
	Bindings   []intelligence.EnvironmentResolution `json:"bindings,omitempty"`
}

func handleTopology(ctx context.Context, args map[string]interface{}) (CallToolResult, error) {
	target, err := resolvePath(args)
	if err != nil {
		return toolError(err.Error()), nil
	}

	model, err := intelligence.Analyze(target)
	if err != nil {
		return toolError(fmt.Sprintf("analysis failed: %v", err)), nil
	}

	graph, err := intelligence.BuildTopologyGraph(model)
	if err != nil {
		return toolError(fmt.Sprintf("topology construction failed: %v", err)), nil
	}

	result := TopologyResult{
		TopologyGraph: graph,
		Components:    model.Components,
		Services:      model.Services,
		Bindings:      intelligence.ResolveEnvironmentBindings(model),
	}

	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return toolError(fmt.Sprintf("failed to serialize topology: %v", err)), nil
	}

	return CallToolResult{
		Content: []ContentItem{{
			Type: "text",
			Text: string(data),
		}},
	}, nil
}

type VerificationReport struct {
	ProjectName  string                            `json:"project_name"`
	Success      bool                              `json:"success"`
	Checks       []intelligence.VerificationResult `json:"checks,omitempty"`
	LockPresent  bool                              `json:"lock_present"`
	LockEntries  int                               `json:"lock_entries,omitempty"`
	Error        string                            `json:"error,omitempty"`
	ErrorCode    intelligence.FailureCode          `json:"error_code,omitempty"`
	RootCause    string                            `json:"root_cause,omitempty"`
	SuggestedFix string                            `json:"suggested_fix,omitempty"`
	Diagnostic   *intelligence.FailureDiagnostic   `json:"diagnostic,omitempty"`
}

func handleVerify(ctx context.Context, args map[string]interface{}) (CallToolResult, error) {
	target, err := resolvePath(args)
	if err != nil {
		return toolError(err.Error()), nil
	}

	timeoutSec := 10
	if tVal, ok := args["timeout"].(float64); ok && tVal > 0 {
		timeoutSec = int(tVal)
	} else if tVal, ok := args["timeout"].(int); ok && tVal > 0 {
		timeoutSec = tVal
	}

	verifyCtx, cancel := context.WithTimeout(ctx, time.Duration(timeoutSec)*time.Second)
	defer cancel()

	model, err := intelligence.Analyze(target)
	if err != nil {
		return toolError(fmt.Sprintf("analysis failed: %v", err)), nil
	}

	planner := intelligence.DeterministicPlanner{}
	plan, err := planner.Plan(verifyCtx, model)
	if err != nil {
		return toolError(fmt.Sprintf("planning failed: %v", err)), nil
	}

	checks := intelligence.BuildVerificationChecks(model, plan)
	results, verifyErr := intelligence.Verify(verifyCtx, checks)

	lock, _ := intelligence.LoadOctoLock(target)
	lockPresent := len(lock.Strategies) > 0

	report := VerificationReport{
		ProjectName: model.Name,
		Success:     verifyErr == nil,
		Checks:      results,
		LockPresent: lockPresent,
		LockEntries: len(lock.Strategies),
	}
	if verifyErr != nil {
		report.Success = false
		report.Error = verifyErr.Error()
		for _, res := range results {
			if !res.Passed && res.ErrorCode != "" {
				report.ErrorCode = res.ErrorCode
				report.RootCause = res.RootCause
				report.SuggestedFix = res.SuggestedFix
				break
			}
		}
		if report.ErrorCode == "" {
			diag := intelligence.DiagnoseExecutionFailure(verifyErr.Error(), verifyErr)
			report.ErrorCode = diag.Code
			report.RootCause = diag.RootCause
			report.SuggestedFix = diag.SuggestedFix
		}
		report.Diagnostic = &intelligence.FailureDiagnostic{
			Code:         report.ErrorCode,
			RootCause:    report.RootCause,
			SuggestedFix: report.SuggestedFix,
		}
	}

	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return toolError(fmt.Sprintf("failed to serialize verification report: %v", err)), nil
	}

	return CallToolResult{
		IsError: verifyErr != nil,
		Content: []ContentItem{{
			Type: "text",
			Text: string(data),
		}},
	}, nil
}

type EnvCheckResult struct {
	ProjectName     string                                  `json:"project_name"`
	TotalVariables  int                                     `json:"total_variables"`
	Present         []string                                `json:"present"`
	MissingRequired []string                                `json:"missing_required"`
	MissingOptional []string                                `json:"missing_optional"`
	CriticalMissing []string                                `json:"critical_missing,omitempty"`
	MissingBindings []intelligence.MissingBindingDiagnostic `json:"missing_bindings,omitempty"`
	Template        string                                  `json:"template,omitempty"`
}

func handleEnvCheck(ctx context.Context, args map[string]interface{}) (CallToolResult, error) {
	target, err := resolvePath(args)
	if err != nil {
		return toolError(err.Error()), nil
	}

	model, err := intelligence.Analyze(target)
	if err != nil {
		return toolError(fmt.Sprintf("analysis failed: %v", err)), nil
	}

	status := intelligence.InspectEnvironmentStatus(target, model)
	env, _ := intelligence.ResolveProjectEnvironment(target, model)

	criticalMap := make(map[string]bool)
	for _, v := range status.MissingRequired {
		if secrets.IsCriticalEnvVar(v) {
			criticalMap[v] = true
		}
	}
	for _, v := range status.MissingOptional {
		if secrets.IsCriticalEnvVar(v) {
			criticalMap[v] = true
		}
	}

	var criticalMissing []string
	for k := range criticalMap {
		criticalMissing = append(criticalMissing, k)
	}
	sort.Strings(criticalMissing)

	missingBindings := intelligence.DiagnoseEnvironmentBindings(model, env)

	result := EnvCheckResult{
		ProjectName:     model.Name,
		TotalVariables:  status.TotalVariables,
		Present:         status.Present,
		MissingRequired: status.MissingRequired,
		MissingOptional: status.MissingOptional,
		CriticalMissing: criticalMissing,
		MissingBindings: missingBindings,
		Template:        status.Template,
	}

	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return toolError(fmt.Sprintf("failed to serialize environment check result: %v", err)), nil
	}

	return CallToolResult{
		Content: []ContentItem{{
			Type: "text",
			Text: string(data),
		}},
	}, nil
}

func handlePreview(ctx context.Context, args map[string]interface{}) (CallToolResult, error) {
	target, err := resolvePath(args)
	if err != nil {
		return toolError(err.Error()), nil
	}

	model, err := intelligence.Analyze(target)
	if err != nil {
		return toolError(fmt.Sprintf("analysis failed: %v", err)), nil
	}

	planner := intelligence.DeterministicPlanner{}
	plan, err := planner.Plan(ctx, model)
	if err != nil {
		return toolError(fmt.Sprintf("planning failed: %v", err)), nil
	}

	preview := intelligence.BuildMachinePreview(ctx, model, plan)
	data, err := json.MarshalIndent(preview, "", "  ")
	if err != nil {
		return toolError(fmt.Sprintf("failed to serialize machine preview: %v", err)), nil
	}

	return CallToolResult{
		Content: []ContentItem{{
			Type: "text",
			Text: string(data),
		}},
	}, nil
}

func handleBenchmark(ctx context.Context, args map[string]interface{}) (CallToolResult, error) {
	if suite, ok := args["suite"].(bool); ok && suite {
		report, err := benchmark.RunBuiltinSuite(ctx)
		if err != nil {
			return toolError(fmt.Sprintf("benchmark suite failed: %v", err)), nil
		}
		data, err := json.MarshalIndent(report, "", "  ")
		if err != nil {
			return toolError(fmt.Sprintf("failed to serialize benchmark suite report: %v", err)), nil
		}
		return CallToolResult{
			Content: []ContentItem{{
				Type: "text",
				Text: string(data),
			}},
		}, nil
	}

	target, err := resolvePath(args)
	if err != nil {
		return toolError(err.Error()), nil
	}

	result, err := benchmark.EvaluateRepository(ctx, target, "")
	if err != nil {
		return toolError(fmt.Sprintf("benchmark evaluation failed: %v", err)), nil
	}

	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return toolError(fmt.Sprintf("failed to serialize benchmark result: %v", err)), nil
	}

	return CallToolResult{
		Content: []ContentItem{{
			Type: "text",
			Text: string(data),
		}},
	}, nil
}

func resolvePath(args map[string]interface{}) (string, error) {
	p := "."
	if val, ok := args["path"].(string); ok && val != "" {
		p = val
	}
	return filepath.Abs(p)
}

func toolError(msg string) CallToolResult {
	return CallToolResult{
		IsError: true,
		Content: []ContentItem{{
			Type: "text",
			Text: msg,
		}},
	}
}

package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"sort"

	"github.com/Octo-sh/Octo-CLI/internal/intelligence"
	"github.com/Octo-sh/Octo-CLI/internal/secrets"
)

func DefaultResources() []Resource {
	return []Resource{
		{
			URI:         "octo://topology",
			Name:        "Repository Topology",
			Description: "Unified dependency and relationship graph of components and infrastructure services.",
			MimeType:    "application/json",
		},
		{
			URI:         "octo://execution-plan",
			Name:        "Execution Plan",
			Description: "Deterministic step-by-step execution plan and topological order.",
			MimeType:    "application/json",
		},
		{
			URI:         "octo://env",
			Name:        "Environment Variables & Template",
			Description: "Status of required, optional, and missing environment variables, plus safe template.",
			MimeType:    "application/json",
		},
		{
			URI:         "octo://lock",
			Name:        "Strategy Lockfile",
			Description: "Cache of verified execution candidates and content fingerprints.",
			MimeType:    "application/json",
		},
	}
}

func ReadResource(ctx context.Context, uriStr string) (ReadResourceResult, error) {
	parsed, err := url.Parse(uriStr)
	if err != nil {
		return ReadResourceResult{}, fmt.Errorf("invalid resource URI %q: %w", uriStr, err)
	}

	target := "."
	if p := parsed.Query().Get("path"); p != "" {
		target = p
	}
	target, err = filepath.Abs(target)
	if err != nil {
		return ReadResourceResult{}, fmt.Errorf("resolve path %q: %w", target, err)
	}

	hostPath := parsed.Host + parsed.Path

	switch hostPath {
	case "topology":
		return readTopologyResource(target, uriStr)
	case "execution-plan":
		return readPlanResource(ctx, target, uriStr)
	case "env":
		return readEnvResource(target, uriStr)
	case "lock":
		return readLockResource(target, uriStr)
	default:
		return ReadResourceResult{}, fmt.Errorf("resource not found: %s", uriStr)
	}
}

func readTopologyResource(target, uri string) (ReadResourceResult, error) {
	model, err := intelligence.Analyze(target)
	if err != nil {
		return ReadResourceResult{}, fmt.Errorf("analyze failed: %w", err)
	}

	graph, err := intelligence.BuildTopologyGraph(model)
	if err != nil {
		return ReadResourceResult{}, fmt.Errorf("build topology failed: %w", err)
	}

	result := TopologyResult{
		TopologyGraph: graph,
		Components:    model.Components,
		Services:      model.Services,
		Bindings:      intelligence.ResolveEnvironmentBindings(model),
	}

	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return ReadResourceResult{}, fmt.Errorf("marshal topology: %w", err)
	}

	return ReadResourceResult{
		Contents: []ResourceContent{{
			URI:      uri,
			MimeType: "application/json",
			Text:     string(data),
		}},
	}, nil
}

func readPlanResource(ctx context.Context, target, uri string) (ReadResourceResult, error) {
	model, err := intelligence.Analyze(target)
	if err != nil {
		return ReadResourceResult{}, fmt.Errorf("analyze failed: %w", err)
	}

	planner := intelligence.DeterministicPlanner{}
	plan, err := planner.Plan(ctx, model)
	if err != nil {
		return ReadResourceResult{}, fmt.Errorf("plan failed: %w", err)
	}

	decisions := intelligence.BuildDecisionTrace(model, plan)
	result := PlanResult{
		ExecutionPlan: plan,
		Decisions:     decisions,
	}

	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return ReadResourceResult{}, fmt.Errorf("marshal plan: %w", err)
	}

	return ReadResourceResult{
		Contents: []ResourceContent{{
			URI:      uri,
			MimeType: "application/json",
			Text:     string(data),
		}},
	}, nil
}

func readEnvResource(target, uri string) (ReadResourceResult, error) {
	model, err := intelligence.Analyze(target)
	if err != nil {
		return ReadResourceResult{}, fmt.Errorf("analyze failed: %w", err)
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

	result := EnvCheckResult{
		ProjectName:     model.Name,
		TotalVariables:  status.TotalVariables,
		Present:         status.Present,
		MissingRequired: status.MissingRequired,
		MissingOptional: status.MissingOptional,
		CriticalMissing: criticalMissing,
		MissingBindings: intelligence.DiagnoseEnvironmentBindings(model, env),
		Template:        status.Template,
	}

	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return ReadResourceResult{}, fmt.Errorf("marshal env resource: %w", err)
	}

	return ReadResourceResult{
		Contents: []ResourceContent{{
			URI:      uri,
			MimeType: "application/json",
			Text:     string(data),
		}},
	}, nil
}

func readLockResource(target, uri string) (ReadResourceResult, error) {
	lockPath := filepath.Join(target, ".octo.lock")
	if _, statErr := os.Stat(lockPath); os.IsNotExist(statErr) {
		return ReadResourceResult{
			Contents: []ResourceContent{{
				URI:      uri,
				MimeType: "application/json",
				Text:     `{"status":"no_lockfile"}`,
			}},
		}, nil
	}

	lock, err := intelligence.LoadOctoLock(target)
	if err != nil {
		return ReadResourceResult{}, fmt.Errorf("load lockfile: %w", err)
	}

	data, err := json.MarshalIndent(lock, "", "  ")
	if err != nil {
		return ReadResourceResult{}, fmt.Errorf("marshal lockfile: %w", err)
	}

	return ReadResourceResult{
		Contents: []ResourceContent{{
			URI:      uri,
			MimeType: "application/json",
			Text:     string(data),
		}},
	}, nil
}

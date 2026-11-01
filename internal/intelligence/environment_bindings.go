package intelligence

import (
	"fmt"
	"sort"
)

// MissingBindingDiagnostic represents an unsatisfied topology reference binding.
type MissingBindingDiagnostic struct {
	Component  string            `json:"component" yaml:"component"`
	Variable   string            `json:"variable" yaml:"variable"`
	Target     string            `json:"target" yaml:"target"`
	Kind       RelationshipKind  `json:"kind" yaml:"kind"`
	Diagnostic FailureDiagnostic `json:"diagnostic" yaml:"diagnostic"`
}

// DiagnoseEnvironmentBindings checks whether required references have matching
// environment values or active backing services.
func DiagnoseEnvironmentBindings(model ProjectModel, env ResolvedEnvironment) []MissingBindingDiagnostic {
	var diagnostics []MissingBindingDiagnostic
	bindings := ResolveEnvironmentBindings(model)
	for _, b := range bindings {
		val := ""
		if env.Values != nil {
			val = env.Values[b.Variable]
		}
		if val == "" {
			for _, scoped := range env.ScopedValues {
				if v, ok := scoped[b.Variable]; ok && v != "" {
					val = v
					break
				}
			}
		}
		if val == "" {
			placeholder := placeholderFor(b.Variable, model)
			diag := FailureDiagnostic{
				Code:         FailureMissingEnvSecret,
				Class:        FailureClassMissingEnvSecret,
				RootCause:    fmt.Sprintf("Component %q requires %q to connect to %q (%s), but the variable is not set.", b.Source, b.Variable, b.Target, b.Kind),
				SuggestedFix: fmt.Sprintf("Run 'octo env init' or set %s=%s in your .env file.", b.Variable, placeholder),
				MissingKey:   b.Variable,
			}
			diagnostics = append(diagnostics, MissingBindingDiagnostic{
				Component:  b.Source,
				Variable:   b.Variable,
				Target:     b.Target,
				Kind:       b.Kind,
				Diagnostic: diag,
			})
		}
	}
	return diagnostics
}

// ResolveEnvironmentBindings derives secret-safe environment bindings from
// explicit topology references. It never reads or stores environment values.
func ResolveEnvironmentBindings(model ProjectModel) []EnvironmentResolution {
	resolutions := make([]EnvironmentResolution, 0)
	for _, component := range model.Components {
		for _, ref := range component.References {
			if ref.Variable == "" || !ref.Kind.IsKnown() {
				continue
			}
			resolutions = append(resolutions, EnvironmentResolution{
				Variable:   ref.Variable,
				Source:     topologyID(NodeComponent, component.Name),
				Target:     ref.Target,
				Kind:       ref.Kind,
				Confidence: ref.Confidence,
				Evidence:   append([]Evidence(nil), ref.Evidence...),
			})
		}
	}
	for _, service := range model.Services {
		for _, ref := range service.References {
			if ref.Variable == "" || !ref.Kind.IsKnown() {
				continue
			}
			resolutions = append(resolutions, EnvironmentResolution{
				Variable:   ref.Variable,
				Source:     topologyID(NodeService, service.Name),
				Target:     ref.Target,
				Kind:       ref.Kind,
				Confidence: ref.Confidence,
				Evidence:   append([]Evidence(nil), ref.Evidence...),
			})
		}
	}
	return sortEnvironmentResolutions(resolutions)
}

func sortEnvironmentResolutions(values []EnvironmentResolution) []EnvironmentResolution {
	sort.Slice(values, func(i, j int) bool {
		if values[i].Source != values[j].Source {
			return values[i].Source < values[j].Source
		}
		if values[i].Variable != values[j].Variable {
			return values[i].Variable < values[j].Variable
		}
		if values[i].Target != values[j].Target {
			return values[i].Target < values[j].Target
		}
		return values[i].Kind < values[j].Kind
	})
	return values
}

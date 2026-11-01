# Contributing to Octo CLI

Thanks for your interest in contributing to **Octo CLI**. We're excited to have you here!

Octo's mission is simple: **Make “how do I run this?” disappear.**

A repository should contain enough evidence to derive a safe, deterministic, explainable path toward execution—for both human developers and autonomous AI coding agents.

---

## Architectural North Star

Every contribution must preserve Octo's core execution philosophy:

1. **Evidence before inference**: Never guess flags, ports, or startup commands. Every decision must be traceable to observable facts in the repository (manifests, lockfiles, scripts, explicit configs).
2. **Bounded candidates**: When multiple entry points or commands are plausible, generate a bounded set of scored candidates. Bounded ambiguity is resolved through explicit decision providers or maintainer overrides (`.octo.yaml`), never by hallucinating unsupported commands.
3. **Deterministic planning**: Repository evidence and decisions must compile into a repeatable, topological execution plan.
4. **Machine-readable core**: Every capability must be accessible to AI coding agents via strongly typed JSON (`--json`) and native Model Context Protocol (MCP) tools (`octo mcp`).
5. **Hermetic testing**: All unit tests must execute in milliseconds without external network calls or running daemon dependencies.

---

## The Contributor Recipe: Adding a New Ecosystem

Octo provides a clean, declarative **Ecosystem Registry** in `internal/intelligence/ecosystems.go`. Contributors can add support for a new language, framework, or runtime with an isolated, ~10-line declarative definition without modifying core detection control flow, graph traversal, or planning algorithms.

### Step 1: Declare the Ecosystem Specification

Add a self-contained definition to `builtInEcosystems` in `internal/intelligence/ecosystems.go` (or call `RegisterEcosystem`):

```go
RegisterEcosystem(EcosystemSpec{
    Name:                  "Dart",
    Files:                 []string{"pubspec.yaml"},
    DefaultPackageManager: "dart",
    DefaultPort:           8080,
    DefaultStartCommand:   "dart run",
    SetupCommand:          "dart pub get",
    SetupRequired:         true,
})
```

Registering an `EcosystemSpec` automatically:
- Registers file signal markers with `ProjectDetectorRegistry`.
- Binds candidate generation with `CandidateProviderRegistry`.
- Enables monorepo and multi-component fallback discovery.
- Sets default ports, setup commands, and package managers.

#### Adding Framework Rules (Optional)
If your ecosystem has popular frameworks (like Phoenix for Elixir, Laravel for PHP, or Shelf for Dart), declare them inside `Frameworks`:

```go
RegisterFramework("Dart", FrameworkRule{
    Name:               "Shelf",
    DependencyPatterns: []string{"shelf"},
    DefaultPort:        8080,
    RunCommand:         "dart run bin/server.dart",
    CandidateID:        "dart.shelf.server",
    Confidence:         0.96,
})
```

### Step 2: Write Hermetic Unit Tests in `ecosystems_expansion_test.go`

Add a table-driven or temp-dir unit test in `internal/intelligence/ecosystems_expansion_test.go`:

```go
func TestDetectAndCandidateDart(t *testing.T) {
    tmp := t.TempDir()
    if err := os.WriteFile(filepath.Join(tmp, "pubspec.yaml"), []byte("name: my_app\n"), 0644); err != nil {
        t.Fatal(err)
    }

    detected, err := detectProject(tmp)
    if err != nil || detected.Language != "Dart" {
        t.Fatalf("expected language Dart, got %q (err: %v)", detected.Language, err)
    }

    model, err := Analyze(tmp)
    if err != nil || model.Language != "Dart" {
        t.Fatalf("expected model language Dart, got %q (err: %v)", model.Language, err)
    }

    planner := DeterministicPlanner{DecisionProvider: DeterministicDecisionProvider{}}
    plan, err := planner.Plan(context.Background(), model)
    if err != nil {
        t.Fatalf("Plan failed: %v", err)
    }

    foundRun := false
    for _, step := range plan.Steps {
        if step.Phase == PhaseStart && step.Command == "dart run" {
            foundRun = true
            break
        }
    }
    if !foundRun {
        t.Fatalf("expected 'dart run' step in plan, got: %+v", plan.Steps)
    }
}
```

### Step 3: Custom Detectors or Providers (Advanced)
For complex language manifests requiring bespoke parsing (e.g. Go module paths, Node engines version constraints, Python virtualenv resolution), assign `Detect` or `CandidateProvider` on the `EcosystemSpec`:

```go
EcosystemSpec{
    Name:              "CustomLang",
    Files:             []string{"custom.manifest"},
    Detect:            detectCustomProject,
    CandidateProvider: CustomCandidateProvider{},
}
```

### Step 4: Verification (if custom health probe protocol needed)
If the ecosystem requires specialized health checking (e.g. custom protocol probe):
```go
RegisterVerificationProvider(&CustomVerificationProvider{})
```

---

## Good First Issues

If you're making your first open-source contribution to Octo, look for:
- **Ecosystem Manifest Refinements**: Improving version parsing or lockfile attribution for existing languages.
- **Port Discovery Patterns**: Adding explicit port regex patterns for popular frameworks.
- **Explainability Tracing**: Adding clearer reason descriptions to `BuildDecisionTrace`.
- **CLI & Diagnostic Polish**: Improving error messages in `octo verify` or `octo explain`.
- **Regression Tests**: Submitting test cases based on popular open-source repositories to `regression_repos_test.go`.

---

## Development Setup

### 1. Fork and Clone
```bash
git clone https://github.com/<your-username>/octo-cli.git
cd octo-cli
```

### 2. Build and Test
Octo requires Go 1.25+ and compiles with standard Go toolchains:
```bash
# Run all tests
go test -count=1 ./...

# Build local binary
go build -o octo ./cmd/octo
```

---

## Branching & Commit Discipline

### Branch Naming
- `feat/<feature-name>` (e.g. `feat/elixir-phoenix-detection`)
- `fix/<bug-name>` (e.g. `fix/port-parsing-regex`)
- `docs/<description>` (e.g. `docs/update-architecture`)
- `test/<description>` (e.g. `test/add-monorepo-regression`)

### Conventional Commits
Use semantic, descriptive commit messages:
```
feat(intelligence): add candidate provider for Phoenix framework
fix(ports): avoid conflicting with reserved compose ports
docs(architecture): document extension provider registries
test(intelligence): add regression suite for Next.js app router
```

---

## Pull Request Guidelines

Before submitting your pull request:
1. Ensure all tests pass: `go test -count=1 ./...`.
2. Confirm the binary builds without errors: `go build -o /dev/null ./cmd/octo`.
3. Fill out the PR template (`.github/pull_request_template.md`), explaining what changed and the evidence supporting it.
4. Keep pull requests focused on a single logical change.

---

## Release Discipline & Semantic Versioning

Octo adheres to strict Semantic Versioning (`vMAJOR.MINOR.PATCH`):
- **MAJOR**: Breaking changes to CLI schemas, MCP tool inputs/outputs, or core contracts.
- **MINOR**: New capabilities, new language/ecosystem support, new MCP tools/resources.
- **PATCH**: Bug fixes, performance improvements, and non-breaking detection refinements.

The canonical Homebrew formula lives in Octo-sh/homebrew-tap and is updated by scripts/bump-tap.sh after each release.

---

## Code of Conduct

Be welcoming, constructive, and respectful. Focus on technical clarity, evidence-based solutions, and collaborative progress.

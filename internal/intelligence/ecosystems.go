package intelligence

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// PackageManagerHint defines a lockfile or marker file and its corresponding package manager name.
type PackageManagerHint struct {
	File     string `json:"file" yaml:"file"`
	Manager  string `json:"manager" yaml:"manager"`
	Lockfile string `json:"lockfile,omitempty" yaml:"lockfile,omitempty"`
}

// FrameworkRule defines declarative rules to detect a framework within an ecosystem.
type FrameworkRule struct {
	Name               string   `json:"name" yaml:"name"`
	Indicators         []string `json:"indicators,omitempty" yaml:"indicators,omitempty"`
	DependencyPatterns []string `json:"dependency_patterns,omitempty" yaml:"dependency_patterns,omitempty"`
	ManifestFiles      []string `json:"manifest_files,omitempty" yaml:"manifest_files,omitempty"`
	DefaultPort        int      `json:"default_port,omitempty" yaml:"default_port,omitempty"`
	RunCommand         string   `json:"run_command,omitempty" yaml:"run_command,omitempty"`
	CandidateID        string   `json:"candidate_id,omitempty" yaml:"candidate_id,omitempty"`
	Confidence         float64  `json:"confidence,omitempty" yaml:"confidence,omitempty"`
}

// CandidateRule defines a declarative rule to generate an ExecutionCandidate.
type CandidateRule struct {
	ID             string  `json:"id" yaml:"id"`
	Command        string  `json:"command" yaml:"command"`
	Confidence     float64 `json:"confidence" yaml:"confidence"`
	Kind           string  `json:"kind,omitempty" yaml:"kind,omitempty"`
	FileIndicator  string  `json:"file_indicator,omitempty" yaml:"file_indicator,omitempty"`
	ContentPattern string  `json:"content_pattern,omitempty" yaml:"content_pattern,omitempty"`
	FileSuffix     string  `json:"file_suffix,omitempty" yaml:"file_suffix,omitempty"`
	Detail         string  `json:"detail,omitempty" yaml:"detail,omitempty"`
}

// EcosystemSpec defines the declarative configuration for a programming language or runtime ecosystem.
type EcosystemSpec struct {
	// Canonical ecosystem / language name (e.g., "Go", "Node", "Python", "Rust", "Java", "Ruby", "PHP", "Elixir", "C#", "Swift", "Kotlin")
	Name string `json:"name" yaml:"name"`

	// Aliases for language matching (e.g., ["dotnet"] for "C#")
	Aliases []string `json:"aliases,omitempty" yaml:"aliases,omitempty"`

	// Manifest or signal files indicating this ecosystem (e.g., ["package.json"], ["go.mod", "go.work"])
	Files []string `json:"files,omitempty" yaml:"files,omitempty"`

	// File suffix or pattern indicators (e.g., ["*.csproj", "*.sln"])
	FilePatterns []string `json:"file_patterns,omitempty" yaml:"file_patterns,omitempty"`

	// Default package manager name (e.g., "npm", "go", "cargo", "mix", "swift", "composer")
	DefaultPackageManager string `json:"default_package_manager,omitempty" yaml:"default_package_manager,omitempty"`

	// Package manager detection rules based on lockfiles or markers
	PackageManagers []PackageManagerHint `json:"package_managers,omitempty" yaml:"package_managers,omitempty"`

	// Default port for applications in this ecosystem
	DefaultPort int `json:"default_port,omitempty" yaml:"default_port,omitempty"`

	// Default development, build, and start commands
	DefaultDevCommand   string `json:"default_dev_command,omitempty" yaml:"default_dev_command,omitempty"`
	DefaultBuildCommand string `json:"default_build_command,omitempty" yaml:"default_build_command,omitempty"`
	DefaultStartCommand string `json:"default_start_command,omitempty" yaml:"default_start_command,omitempty"`

	// Setup command and whether setup is required by default
	SetupCommand  string `json:"setup_command,omitempty" yaml:"setup_command,omitempty"`
	SetupRequired bool   `json:"setup_required,omitempty" yaml:"setup_required,omitempty"`

	// Framework detection rules
	Frameworks []FrameworkRule `json:"frameworks,omitempty" yaml:"frameworks,omitempty"`

	// Declarative candidate generation rules
	CandidateRules []CandidateRule `json:"candidate_rules,omitempty" yaml:"candidate_rules,omitempty"`

	// Custom detection function for complex manifests (e.g., Node package.json engines, Go version, Python pyproject)
	// If nil, the declarative detector handles detection automatically.
	Detect func(root string) (DetectedProject, error) `json:"-" yaml:"-"`

	// Custom candidate provider for complex multi-candidate heuristics (e.g., Go nested packages, Python virtualenvs)
	// If nil, declarative candidate generation uses CandidateRules, Frameworks, and DefaultStartCommand.
	CandidateProvider CandidateProvider `json:"-" yaml:"-"`
}

// Supports reports whether this ecosystem supports the given component.
func (s EcosystemSpec) Supports(comp Component) bool {
	if strings.EqualFold(comp.Language, s.Name) {
		return true
	}
	for _, alias := range s.Aliases {
		if strings.EqualFold(comp.Language, alias) {
			return true
		}
	}
	return false
}

// DetectProject performs declarative project detection on the given root directory.
func (s EcosystemSpec) DetectProject(root string) (DetectedProject, error) {
	if s.Detect != nil {
		return s.Detect(root)
	}

	name := filepath.Base(root)
	pm := s.DetectPackageManager(root)
	runCmd := s.DefaultStartCommand
	port := s.DefaultPort

	// Check if any framework matches to override run command or port
	for _, fw := range s.Frameworks {
		if s.matchesFramework(root, fw) {
			if fw.RunCommand != "" {
				runCmd = fw.RunCommand
			}
			if fw.DefaultPort != 0 {
				port = fw.DefaultPort
			}
			break
		}
	}

	// If run command is still empty, try matching candidate rules
	if runCmd == "" && len(s.CandidateRules) > 0 {
		for _, r := range s.CandidateRules {
			if s.matchesCandidateRule(root, r) {
				runCmd = r.Command
				break
			}
		}
	}

	return DetectedProject{
		Name:           name,
		Language:       s.Name,
		RunCommand:     runCmd,
		Port:           port,
		PackageManager: pm,
		SetupCommand:   s.SetupCommand,
		SetupRequired:  s.SetupRequired,
	}, nil
}

// DetectPackageManager detects the active package manager for this ecosystem in root.
func (s EcosystemSpec) DetectPackageManager(root string) string {
	for _, hint := range s.PackageManagers {
		if hint.File != "" {
			if _, err := os.Stat(filepath.Join(root, hint.File)); err == nil {
				return hint.Manager
			}
		}
	}
	return s.DefaultPackageManager
}

// GenerateCandidates generates execution candidates using declarative rules or custom provider.
func (s EcosystemSpec) GenerateCandidates(ctx context.Context, root string, comp Component) ([]ExecutionCandidate, error) {
	if s.CandidateProvider != nil {
		return s.CandidateProvider.Candidates(ctx, root, comp)
	}

	compDir := root
	if comp.Path != "" && comp.Path != "." {
		compDir = filepath.Join(root, filepath.FromSlash(comp.Path))
	}

	var candidates []ExecutionCandidate

	// 1. Framework-derived candidates
	for _, fw := range s.Frameworks {
		matches := false
		if comp.Framework != "" && strings.EqualFold(comp.Framework, fw.Name) {
			matches = true
		} else if s.matchesFramework(compDir, fw) {
			matches = true
		}
		if matches && fw.RunCommand != "" {
			candID := fw.CandidateID
			if candID == "" {
				candID = fmt.Sprintf("%s.%s", strings.ToLower(s.Name), strings.ToLower(strings.ReplaceAll(fw.Name, " ", ".")))
			}
			confidence := fw.Confidence
			if confidence == 0 {
				confidence = 0.95
			}
			evFile := ""
			if len(fw.ManifestFiles) > 0 {
				evFile = fw.ManifestFiles[0]
			} else if len(s.Files) > 0 {
				evFile = s.Files[0]
			}
			evPath := comp.Path
			if evFile != "" {
				evPath = filepath.Join(comp.Path, evFile)
			}
			candidates = append(candidates, ExecutionCandidate{
				Kind:       "application",
				ID:         candID,
				Command:    fw.RunCommand,
				Confidence: confidence,
				Evidence: []Evidence{{
					Kind:     EvidenceManifest,
					Path:     filepath.ToSlash(evPath),
					Detail:   fmt.Sprintf("%s framework detected for %s", fw.Name, s.Name),
					Strength: confidence,
				}},
			})
		}
	}

	// 2. Candidate rules
	for _, rule := range s.CandidateRules {
		if !s.matchesCandidateRule(compDir, rule) {
			continue
		}
		kind := rule.Kind
		if kind == "" {
			kind = "application"
		}
		confidence := rule.Confidence
		if confidence == 0 {
			confidence = 0.90
		}
		detail := rule.Detail
		if detail == "" {
			detail = fmt.Sprintf("%s execution candidate", s.Name)
		}
		evPath := comp.Path
		if rule.FileIndicator != "" {
			evPath = filepath.Join(comp.Path, rule.FileIndicator)
		}
		candidates = append(candidates, ExecutionCandidate{
			Kind:       kind,
			ID:         rule.ID,
			Command:    rule.Command,
			Confidence: confidence,
			Evidence: []Evidence{{
				Kind:     EvidenceConfig,
				Path:     filepath.ToSlash(evPath),
				Detail:   detail,
				Strength: confidence,
			}},
		})
	}

	// 3. Fallback default start command if no candidates generated yet
	if len(candidates) == 0 && s.DefaultStartCommand != "" {
		candidates = append(candidates, ExecutionCandidate{
			Kind:       "application",
			ID:         strings.ToLower(s.Name) + ".run",
			Command:    s.DefaultStartCommand,
			Confidence: 0.85,
			Evidence: []Evidence{{
				Kind:     EvidenceSignalFile,
				Path:     filepath.ToSlash(comp.Path),
				Detail:   fmt.Sprintf("%s standard runner", s.Name),
				Strength: 0.85,
			}},
		})
	}

	return candidates, nil
}

func (s EcosystemSpec) matchesFramework(root string, fw FrameworkRule) bool {
	for _, ind := range fw.Indicators {
		if _, err := os.Stat(filepath.Join(root, ind)); err == nil {
			return true
		}
	}
	manifests := fw.ManifestFiles
	if len(manifests) == 0 {
		manifests = s.Files
	}
	for _, m := range manifests {
		data, err := os.ReadFile(filepath.Join(root, m))
		if err != nil {
			continue
		}
		content := strings.ToLower(string(data))
		for _, pat := range fw.DependencyPatterns {
			if strings.Contains(content, strings.ToLower(pat)) {
				return true
			}
		}
	}
	return false
}

func (s EcosystemSpec) matchesCandidateRule(root string, rule CandidateRule) bool {
	if rule.FileIndicator != "" {
		target := filepath.Join(root, rule.FileIndicator)
		data, err := os.ReadFile(target)
		if err != nil {
			return false
		}
		if rule.ContentPattern != "" && !strings.Contains(strings.ToLower(string(data)), strings.ToLower(rule.ContentPattern)) {
			return false
		}
		return true
	}
	if rule.FileSuffix != "" {
		entries, err := os.ReadDir(root)
		if err != nil {
			return false
		}
		for _, e := range entries {
			if !e.IsDir() && strings.HasSuffix(e.Name(), rule.FileSuffix) {
				return true
			}
		}
		return false
	}
	return true
}

// EcosystemRegistry manages registered language and runtime ecosystems.
type EcosystemRegistry struct {
	mu         sync.RWMutex
	ecosystems []EcosystemSpec
}

// builtInEcosystems contains the declarative specifications for all standard ecosystems.
var builtInEcosystems = []EcosystemSpec{
	{
		Name:                  "Go",
		Files:                 []string{"go.work", "go.mod"},
		DefaultPackageManager: "go",
		CandidateProvider:     GoExecutionCandidateProvider{},
		Detect:                detectGoProject,
	},
	{
		Name:                  "Node",
		Files:                 []string{"package.json"},
		DefaultPackageManager: "npm",
		PackageManagers: []PackageManagerHint{
			{File: "bun.lockb", Manager: "bun"},
			{File: "bun.lock", Manager: "bun"},
			{File: "pnpm-lock.yaml", Manager: "pnpm"},
			{File: "yarn.lock", Manager: "yarn"},
			{File: "package-lock.json", Manager: "npm"},
		},
		DefaultPort: 3000,
		Frameworks: []FrameworkRule{
			{Name: "Next.js", DependencyPatterns: []string{"next"}, DefaultPort: 3000},
			{Name: "React", DependencyPatterns: []string{"react"}, DefaultPort: 3000},
			{Name: "Vue", DependencyPatterns: []string{"vue"}, DefaultPort: 5173},
			{Name: "Nuxt", DependencyPatterns: []string{"nuxt"}, DefaultPort: 3000},
			{Name: "Svelte", DependencyPatterns: []string{"svelte"}, DefaultPort: 5173},
			{Name: "Express", DependencyPatterns: []string{"express"}, DefaultPort: 3000},
			{Name: "Fastify", DependencyPatterns: []string{"fastify"}, DefaultPort: 3000},
			{Name: "NestJS", DependencyPatterns: []string{"@nestjs/core"}, DefaultPort: 3000},
		},
		CandidateProvider: NodeExecutionCandidateProvider{},
		Detect:            detectNodeProject,
	},
	{
		Name:                  "Python",
		Files:                 []string{"pyproject.toml", "requirements.txt", "setup.py", "Pipfile"},
		DefaultPackageManager: "pip",
		PackageManagers: []PackageManagerHint{
			{File: "uv.lock", Manager: "uv"},
			{File: "poetry.lock", Manager: "poetry"},
			{File: "Pipfile.lock", Manager: "pipenv"},
			{File: "Pipfile", Manager: "pipenv"},
			{File: "requirements.txt", Manager: "pip"},
		},
		DefaultPort: 8000,
		Frameworks: []FrameworkRule{
			{Name: "FastAPI", DependencyPatterns: []string{"fastapi"}, DefaultPort: 8000},
			{Name: "Django", Indicators: []string{"manage.py"}, DependencyPatterns: []string{"django"}, DefaultPort: 8000},
			{Name: "Flask", Indicators: []string{".flaskenv"}, DependencyPatterns: []string{"flask"}, DefaultPort: 5000},
		},
		CandidateProvider: PythonExecutionCandidateProvider{},
		Detect:            detectPythonProject,
	},
	{
		Name:                  "Java",
		Files:                 []string{"pom.xml", "build.gradle"},
		DefaultPackageManager: "maven",
		DefaultPort:           8080,
		Frameworks: []FrameworkRule{
			{Name: "Spring Boot", DependencyPatterns: []string{"spring-boot", "org.springframework.boot"}, DefaultPort: 8080},
		},
		CandidateProvider: JavaExecutionCandidateProvider{},
		Detect:            detectJavaProject,
	},
	{
		Name:                  "Ruby",
		Files:                 []string{"Gemfile"},
		DefaultPackageManager: "bundle",
		DefaultPort:           3000,
		Frameworks: []FrameworkRule{
			{Name: "Rails", Indicators: []string{"bin/rails"}, DefaultPort: 3000},
		},
		CandidateProvider: RubyExecutionCandidateProvider{},
		Detect:            detectRubyProject,
	},
	{
		Name:                  "Rust",
		Files:                 []string{"Cargo.toml"},
		DefaultPackageManager: "cargo",
		PackageManagers: []PackageManagerHint{
			{File: "Cargo.lock", Manager: "cargo"},
		},
		CandidateProvider: RustExecutionCandidateProvider{},
		Detect:            detectRustProject,
	},
	{
		Name:                  "PHP",
		Files:                 []string{"composer.json"},
		DefaultPackageManager: "composer",
		DefaultPort:           8000,
		SetupCommand:          "composer install",
		SetupRequired:         true,
		Frameworks: []FrameworkRule{
			{
				Name:        "Laravel",
				Indicators:  []string{"artisan"},
				DefaultPort: 8000,
				RunCommand:  "php artisan serve",
				CandidateID: "php.artisan.serve",
				Confidence:  0.95,
			},
		},
		CandidateProvider: PHPExecutionCandidateProvider{},
		Detect:            detectPHPProject,
	},
	{
		Name:                  "Elixir",
		Files:                 []string{"mix.exs"},
		DefaultPackageManager: "mix",
		DefaultPort:           4000,
		SetupCommand:          "mix deps.get",
		SetupRequired:         true,
		Frameworks: []FrameworkRule{
			{
				Name:               "Phoenix",
				DependencyPatterns: []string{":phoenix", "phoenix"},
				ManifestFiles:      []string{"mix.exs"},
				DefaultPort:        4000,
				RunCommand:         "mix phx.server",
				CandidateID:        "elixir.mix.phx.server",
				Confidence:         0.95,
			},
		},
		CandidateRules: []CandidateRule{
			{
				ID:             "elixir.mix.phx.server",
				Command:        "mix phx.server",
				Confidence:     0.95,
				FileIndicator:  "mix.exs",
				ContentPattern: "phoenix",
				Detail:         "Phoenix framework dependency found in mix.exs",
			},
			{
				ID:            "elixir.mix.run",
				Command:       "mix run --no-halt",
				Confidence:    0.75,
				FileIndicator: "mix.exs",
				Detail:        "Elixir mix standard runner",
			},
		},
		CandidateProvider: ElixirExecutionCandidateProvider{},
		Detect:            detectElixirProject,
	},
	{
		Name:                  "C#",
		Aliases:               []string{"dotnet"},
		Files:                 []string{"global.json"},
		FilePatterns:          []string{"*.csproj", "*.sln"},
		DefaultPackageManager: "dotnet",
		DefaultPort:           5000,
		SetupCommand:          "dotnet restore",
		CandidateProvider:     DotnetExecutionCandidateProvider{},
		Detect:                detectDotnetProject,
	},
	{
		Name:                  "Swift",
		Files:                 []string{"Package.swift"},
		DefaultPackageManager: "swift",
		DefaultPort:           8080,
		DefaultStartCommand:   "swift run",
		SetupCommand:          "swift package resolve",
		CandidateRules: []CandidateRule{
			{
				ID:            "swift.run",
				Command:       "swift run",
				Confidence:    0.95,
				FileIndicator: "Package.swift",
				Detail:        "Swift Package Manager standard run command",
			},
		},
		CandidateProvider: SwiftExecutionCandidateProvider{},
		Detect:            detectSwiftProject,
	},
	{
		Name:                  "Kotlin",
		Files:                 []string{"build.gradle.kts"},
		DefaultPackageManager: "gradle",
		DefaultPort:           8080,
		Frameworks: []FrameworkRule{
			{
				Name:               "Spring Boot",
				DependencyPatterns: []string{"spring", "boot"},
				ManifestFiles:      []string{"build.gradle.kts"},
				DefaultPort:        8080,
			},
		},
		CandidateProvider: KotlinExecutionCandidateProvider{},
		Detect:            detectKotlinProject,
	},
}

// NewEcosystemRegistry creates a registry initialized with all built-in ecosystems.
func NewEcosystemRegistry() *EcosystemRegistry {
	r := &EcosystemRegistry{}
	for _, spec := range builtInEcosystems {
		r.Register(spec)
	}
	return r
}

var defaultEcosystemRegistry = NewEcosystemRegistry()

// DefaultEcosystemRegistry returns the shared global ecosystem registry.
func DefaultEcosystemRegistry() *EcosystemRegistry {
	return defaultEcosystemRegistry
}

// Register adds an ecosystem specification to the registry.
func (r *EcosystemRegistry) Register(spec EcosystemSpec) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i, existing := range r.ecosystems {
		if strings.EqualFold(existing.Name, spec.Name) {
			r.ecosystems[i] = spec
			return
		}
	}
	r.ecosystems = append(r.ecosystems, spec)
}

// RegisterFramework adds a framework detection rule to a registered ecosystem.
func (r *EcosystemRegistry) RegisterFramework(language string, rule FrameworkRule) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i, eco := range r.ecosystems {
		if strings.EqualFold(eco.Name, language) {
			// Replace existing if same framework name
			for j, existing := range r.ecosystems[i].Frameworks {
				if strings.EqualFold(existing.Name, rule.Name) {
					r.ecosystems[i].Frameworks[j] = rule
					return nil
				}
			}
			r.ecosystems[i].Frameworks = append(r.ecosystems[i].Frameworks, rule)
			return nil
		}
	}
	return fmt.Errorf("ecosystem %q not found", language)
}

// RegisterFramework registers a framework detection rule in the default ecosystem registry.
func RegisterFramework(language string, rule FrameworkRule) error {
	return defaultEcosystemRegistry.RegisterFramework(language, rule)
}

// FindEcosystem looks up an ecosystem by language name or alias.
func (r *EcosystemRegistry) FindEcosystem(nameOrAlias string) (EcosystemSpec, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, spec := range r.ecosystems {
		if strings.EqualFold(spec.Name, nameOrAlias) {
			return spec, true
		}
		for _, alias := range spec.Aliases {
			if strings.EqualFold(alias, nameOrAlias) {
				return spec, true
			}
		}
	}
	return EcosystemSpec{}, false
}

// FindByFile finds an ecosystem whose indicator files match the given filename.
func (r *EcosystemRegistry) FindByFile(filename string) (EcosystemSpec, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, spec := range r.ecosystems {
		for _, f := range spec.Files {
			if f == filename {
				return spec, true
			}
		}
	}
	return EcosystemSpec{}, false
}

// Ecosystems returns a snapshot slice of all registered ecosystems.
func (r *EcosystemRegistry) Ecosystems() []EcosystemSpec {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]EcosystemSpec, len(r.ecosystems))
	copy(out, r.ecosystems)
	return out
}

// RegisterEcosystem registers an ecosystem into the default ecosystem, detector, and candidate registries.
func RegisterEcosystem(spec EcosystemSpec) {
	defaultEcosystemRegistry.Register(spec)

	// Register file marker signals in detector registry
	for _, file := range spec.Files {
		f := file
		lang := spec.Name
		RegisterProjectSignal(SignalDefinition{
			File:     f,
			Language: lang,
			Detect: func(root string) (DetectedProject, error) {
				if live, ok := DefaultEcosystemRegistry().FindEcosystem(lang); ok {
					return live.DetectProject(root)
				}
				return spec.DetectProject(root)
			},
		})
	}

	// Register pattern detector if ecosystem defines file patterns
	if len(spec.FilePatterns) > 0 {
		s := spec
		RegisterProjectDetector(ecosystemPatternDetector{spec: s})
	}

	// Register candidate provider
	if spec.CandidateProvider != nil {
		RegisterCandidateProvider(spec.CandidateProvider)
	} else {
		RegisterCandidateProvider(ecosystemCandidateProvider{spec: spec})
	}
}

type ecosystemCandidateProvider struct {
	spec EcosystemSpec
}

func (p ecosystemCandidateProvider) Name() string {
	return strings.ToLower(p.spec.Name)
}

func (p ecosystemCandidateProvider) Supports(component Component) bool {
	spec := p.spec
	if live, ok := DefaultEcosystemRegistry().FindEcosystem(p.spec.Name); ok {
		spec = live
	}
	return spec.Supports(component)
}

func (p ecosystemCandidateProvider) Candidates(ctx context.Context, root string, component Component) ([]ExecutionCandidate, error) {
	spec := p.spec
	if live, ok := DefaultEcosystemRegistry().FindEcosystem(p.spec.Name); ok {
		spec = live
	}
	return spec.GenerateCandidates(ctx, root, component)
}

type ecosystemPatternDetector struct {
	spec EcosystemSpec
}

func (d ecosystemPatternDetector) Detect(root string) (DetectedProject, error) {
	spec := d.spec
	if live, ok := DefaultEcosystemRegistry().FindEcosystem(d.spec.Name); ok {
		spec = live
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		return DetectedProject{}, err
	}
	matched := false
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		for _, pattern := range spec.FilePatterns {
			if matchedName, err := filepath.Match(pattern, entry.Name()); err == nil && matchedName {
				matched = true
				break
			}
		}
		if matched {
			break
		}
	}
	if !matched {
		return DetectedProject{}, fmt.Errorf("no %s pattern signal found", spec.Name)
	}
	return spec.DetectProject(root)
}


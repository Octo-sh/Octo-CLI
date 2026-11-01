package intelligence

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestDetectAndCandidateDotnet(t *testing.T) {
	tmp := t.TempDir()
	csproj := filepath.Join(tmp, "MyWeb.csproj")
	if err := os.WriteFile(csproj, []byte("<Project Sdk=\"Microsoft.NET.Sdk.Web\"></Project>"), 0644); err != nil {
		t.Fatal(err)
	}

	detected, err := detectProject(tmp)
	if err != nil {
		t.Fatalf("detectProject failed: %v", err)
	}
	if detected.Language != "C#" {
		t.Fatalf("expected language C#, got %q", detected.Language)
	}

	model, err := Analyze(tmp)
	if err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}
	if model.Language != "C#" {
		t.Fatalf("model language = %q, want C#", model.Language)
	}

	planner := DeterministicPlanner{DecisionProvider: DeterministicDecisionProvider{}}
	plan, err := planner.Plan(context.Background(), model)
	if err != nil {
		t.Fatalf("Plan failed: %v", err)
	}

	foundRun := false
	for _, step := range plan.Steps {
		if step.Phase == PhaseStart && (step.Command == "dotnet run --project MyWeb.csproj" || step.Command == "dotnet run") {
			foundRun = true
			break
		}
	}
	if !foundRun {
		t.Fatalf("expected dotnet run step in plan, got: %+v", plan.Steps)
	}
}

func TestDetectAndCandidateSwift(t *testing.T) {
	tmp := t.TempDir()
	pkgSwift := filepath.Join(tmp, "Package.swift")
	if err := os.WriteFile(pkgSwift, []byte("// swift-tools-version:5.9\nimport PackageDescription\nlet package = Package(name: \"MyCli\")"), 0644); err != nil {
		t.Fatal(err)
	}

	detected, err := detectProject(tmp)
	if err != nil {
		t.Fatalf("detectProject failed: %v", err)
	}
	if detected.Language != "Swift" {
		t.Fatalf("expected language Swift, got %q", detected.Language)
	}

	model, err := Analyze(tmp)
	if err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}
	if model.Language != "Swift" {
		t.Fatalf("model language = %q, want Swift", model.Language)
	}

	planner := DeterministicPlanner{DecisionProvider: DeterministicDecisionProvider{}}
	plan, err := planner.Plan(context.Background(), model)
	if err != nil {
		t.Fatalf("Plan failed: %v", err)
	}

	foundRun := false
	for _, step := range plan.Steps {
		if step.Phase == PhaseStart && step.Command == "swift run" {
			foundRun = true
			break
		}
	}
	if !foundRun {
		t.Fatalf("expected swift run step in plan, got: %+v", plan.Steps)
	}
}

func TestDetectAndCandidateKotlin(t *testing.T) {
	tmp := t.TempDir()
	buildGradle := filepath.Join(tmp, "build.gradle.kts")
	content := `plugins {
    id("org.springframework.boot") version "3.2.0"
    kotlin("jvm") version "1.9.20"
}`
	if err := os.WriteFile(buildGradle, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	// fake gradlew wrapper
	if err := os.WriteFile(filepath.Join(tmp, "gradlew"), []byte("#!/bin/sh\n"), 0755); err != nil {
		t.Fatal(err)
	}

	detected, err := detectProject(tmp)
	if err != nil {
		t.Fatalf("detectProject failed: %v", err)
	}
	if detected.Language != "Kotlin" {
		t.Fatalf("expected language Kotlin, got %q", detected.Language)
	}

	model, err := Analyze(tmp)
	if err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}
	if model.Language != "Kotlin" {
		t.Fatalf("model language = %q, want Kotlin", model.Language)
	}

	planner := DeterministicPlanner{DecisionProvider: DeterministicDecisionProvider{}}
	plan, err := planner.Plan(context.Background(), model)
	if err != nil {
		t.Fatalf("Plan failed: %v", err)
	}

	foundRun := false
	for _, step := range plan.Steps {
		if step.Phase == PhaseStart && (step.Command == "./gradlew bootRun" || step.Command == "./gradlew run") {
			foundRun = true
			break
		}
	}
	if !foundRun {
		t.Fatalf("expected ./gradlew bootRun in plan, got: %+v", plan.Steps)
	}
}

func TestDeclarativeEcosystemRegistration(t *testing.T) {
	RegisterEcosystem(EcosystemSpec{
		Name:                  "Dart",
		Files:                 []string{"pubspec.yaml"},
		DefaultPackageManager: "dart",
		DefaultPort:           8080,
		DefaultStartCommand:   "dart run",
		CandidateRules: []CandidateRule{
			{
				ID:            "dart.run",
				Command:       "dart run",
				Confidence:    0.95,
				FileIndicator: "pubspec.yaml",
				Detail:        "Dart standard application runner",
			},
		},
	})

	tmp := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmp, "pubspec.yaml"), []byte("name: my_app\n"), 0644); err != nil {
		t.Fatal(err)
	}

	detected, err := detectProject(tmp)
	if err != nil {
		t.Fatalf("detectProject failed: %v", err)
	}
	if detected.Language != "Dart" {
		t.Fatalf("expected language Dart, got %q", detected.Language)
	}
	if detected.PackageManager != "dart" {
		t.Fatalf("expected package manager dart, got %q", detected.PackageManager)
	}

	model, err := Analyze(tmp)
	if err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}
	if model.Language != "Dart" {
		t.Fatalf("model language = %q, want Dart", model.Language)
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
		t.Fatalf("expected dart run in plan, got: %+v", plan.Steps)
	}
}

func TestDeclarativeFrameworkDetection(t *testing.T) {
	RegisterEcosystem(EcosystemSpec{
		Name:                  "Dart",
		Files:                 []string{"pubspec.yaml"},
		DefaultPackageManager: "dart",
		DefaultPort:           8080,
		DefaultStartCommand:   "dart run",
	})

	err := RegisterFramework("Dart", FrameworkRule{
		Name:               "Shelf",
		DependencyPatterns: []string{"shelf"},
		DefaultPort:        8080,
		RunCommand:         "dart run bin/server.dart",
		CandidateID:        "dart.shelf.server",
		Confidence:         0.96,
	})
	if err != nil {
		t.Fatalf("RegisterFramework failed: %v", err)
	}

	tmp := t.TempDir()
	content := "name: my_shelf_app\ndependencies:\n  shelf: ^1.4.0\n"
	if err := os.WriteFile(filepath.Join(tmp, "pubspec.yaml"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	model, err := Analyze(tmp)
	if err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}
	if model.Framework != "Shelf" {
		t.Fatalf("expected framework Shelf, got %q", model.Framework)
	}

	planner := DeterministicPlanner{DecisionProvider: DeterministicDecisionProvider{}}
	plan, err := planner.Plan(context.Background(), model)
	if err != nil {
		t.Fatalf("Plan failed: %v", err)
	}

	foundRun := false
	for _, step := range plan.Steps {
		if step.Phase == PhaseStart && step.Command == "dart run bin/server.dart" {
			foundRun = true
			break
		}
	}
	if !foundRun {
		t.Fatalf("expected dart run bin/server.dart in plan, got: %+v", plan.Steps)
	}
}

func TestDeclarativeEcosystemMonorepoDiscovery(t *testing.T) {
	RegisterEcosystem(EcosystemSpec{
		Name:                  "Dart",
		Files:                 []string{"pubspec.yaml"},
		DefaultPackageManager: "dart",
		DefaultPort:           8080,
		DefaultStartCommand:   "dart run",
	})

	root := t.TempDir()
	clientDir := filepath.Join(root, "client")
	serverDir := filepath.Join(root, "server")
	if err := os.MkdirAll(clientDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(serverDir, 0755); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(clientDir, "package.json"), []byte(`{"name":"client","scripts":{"start":"react-scripts start"}}`), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(serverDir, "pubspec.yaml"), []byte("name: server\n"), 0644); err != nil {
		t.Fatal(err)
	}

	model, err := Analyze(root)
	if err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}
	if !model.Monorepo {
		t.Fatal("expected multi-component repository to be marked Monorepo")
	}
	if len(model.Components) != 2 {
		t.Fatalf("expected 2 components, got %d", len(model.Components))
	}

	dartComp := findComponent(model.Components, "server")
	if dartComp == nil {
		t.Fatal("server component not found")
	}
	if dartComp.Language != "Dart" {
		t.Fatalf("expected Dart language, got %q", dartComp.Language)
	}
	if dartComp.PackageManager != "dart" {
		t.Fatalf("expected dart package manager, got %q", dartComp.PackageManager)
	}
}

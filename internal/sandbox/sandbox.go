// Package sandbox manages YAML specifications, isolated Go module workspaces,
// sequential compiler/vet/race verification, and safe promotion to generated/.
package sandbox

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/PratikDhanave/agentic-software-factory-go/internal/state"
	"github.com/PratikDhanave/agentic-software-factory-go/internal/store"
)

type SpecIntent struct {
	Goal            string   `yaml:"goal" json:"goal"`
	Context         string   `yaml:"context,omitempty" json:"context,omitempty"`
	SuccessCriteria []string `yaml:"success_criteria" json:"success_criteria"`
	AntiGoals       []string `yaml:"anti_goals,omitempty" json:"anti_goals,omitempty"`
}

type SpecConstraint struct {
	ID          string `yaml:"id" json:"id"`
	Type        string `yaml:"type" json:"type"`
	Description string `yaml:"description" json:"description"`
}

type SpecStep struct {
	ID          string   `yaml:"id" json:"id"`
	Name        string   `yaml:"name" json:"name"`
	Description string   `yaml:"description" json:"description"`
	DependsOn   []string `yaml:"depends_on,omitempty" json:"depends_on,omitempty"`
}

type Spec struct {
	ID          string           `yaml:"id" json:"id"`
	Version     string           `yaml:"version" json:"version"`
	Title       string           `yaml:"title" json:"title"`
	Description string           `yaml:"description" json:"description"`
	Intent      SpecIntent       `yaml:"intent" json:"intent"`
	Constraints []SpecConstraint `yaml:"constraints,omitempty" json:"constraints,omitempty"`
	Steps       []SpecStep       `yaml:"steps" json:"steps"`
	Filename    string           `yaml:"-" json:"filename,omitempty"`
}

func LoadSpecs(specsDir string) ([]Spec, error) {
	entries, err := os.ReadDir(specsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []Spec{}, nil
		}
		return nil, err
	}
	var out []Spec
	for _, ent := range entries {
		if ent.IsDir() || (!strings.HasSuffix(ent.Name(), ".yaml") && !strings.HasSuffix(ent.Name(), ".yml")) {
			continue
		}
		b, err := os.ReadFile(filepath.Join(specsDir, ent.Name()))
		if err != nil {
			continue
		}
		var sp Spec
		if err := yaml.Unmarshal(b, &sp); err == nil && sp.ID != "" {
			sp.Filename = ent.Name()
			out = append(out, sp)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

func SaveSpec(specsDir string, sp Spec) (Spec, error) {
	if err := os.MkdirAll(specsDir, 0755); err != nil {
		return sp, err
	}
	slug := sanitizeSlug(sp.Title)
	if sp.ID == "" {
		sp.ID = "spec-" + slug
	}
	if sp.Version == "" {
		sp.Version = "1.0"
	}
	if sp.Intent.Goal == "" {
		sp.Intent.Goal = sp.Description
	}
	if len(sp.Intent.SuccessCriteria) == 0 {
		sp.Intent.SuccessCriteria = []string{
			"Compiles cleanly with go build ./...",
			"Passes go test -race -v ./...",
		}
	}
	if len(sp.Steps) == 0 {
		sp.Steps = []SpecStep{
			{ID: "step-model", Name: "Domain Types & Validation", Description: "Define types, constructor validation, and mutex safety"},
			{ID: "step-impl", Name: "Core Implementation", Description: "Implement core methods and invariants", DependsOn: []string{"step-model"}},
			{ID: "step-tests", Name: "Table-Driven & Race Tests", Description: "Verify edge cases and goroutine safety under -race", DependsOn: []string{"step-impl"}},
		}
	}
	filename := strings.TrimPrefix(sp.ID, "spec-") + ".yaml"
	sp.Filename = filename
	b, err := yaml.Marshal(&sp)
	if err != nil {
		return sp, err
	}
	return sp, os.WriteFile(filepath.Join(specsDir, filename), b, 0644)
}

var fencedBlockRe = regexp.MustCompile("(?s)```(?:go|golang)\\s+path=([^\\s\\n]+)\\s*\\n(.*?)```")

func ParseFiles(raw string) []store.FileArtifact {
	matches := fencedBlockRe.FindAllStringSubmatch(raw, -1)
	out := make([]store.FileArtifact, 0, len(matches))
	for _, m := range matches {
		rel := filepath.Clean(strings.TrimSpace(m[1]))
		if strings.HasPrefix(rel, "..") || filepath.IsAbs(rel) {
			continue
		}
		content := strings.TrimLeft(m[2], "\n")
		lines := strings.Count(strings.TrimRight(content, "\n"), "\n") + 1
		out = append(out, store.FileArtifact{
			Path:    rel,
			Lines:   lines,
			Content: content,
		})
	}
	return out
}

type VerifyReport struct {
	FinalState  state.State
	BuildPassed bool
	VetPassed   bool
	TestsPassed bool
	TestCases   []string
	Output      string
	Error       string
}

var testRunRe = regexp.MustCompile(`(?m)^--- PASS: (Test\S+)`)

func ProvisionAndVerify(ctx context.Context, workspaceDir, moduleSlug string, files []store.FileArtifact, constraints []SpecConstraint) VerifyReport {
	current := state.VerifyBuild
	var logBuf strings.Builder

	if len(files) == 0 {
		_ = state.SandboxVerifyMachine.Transition(&current, state.VerifyFailed)
		return VerifyReport{FinalState: current, Error: "synthesizer produced zero Go files"}
	}

	if err := os.MkdirAll(workspaceDir, 0755); err != nil {
		_ = state.SandboxVerifyMachine.Transition(&current, state.VerifyFailed)
		return VerifyReport{FinalState: current, Error: err.Error()}
	}

	// Clean stale .go files on self-healing retry
	if entries, err := os.ReadDir(workspaceDir); err == nil {
		for _, ent := range entries {
			if !ent.IsDir() && strings.HasSuffix(ent.Name(), ".go") {
				_ = os.Remove(filepath.Join(workspaceDir, ent.Name()))
			}
		}
	}

	goMod := fmt.Sprintf("module asf.local/%s\n\ngo 1.23\n", sanitizeSlug(moduleSlug))
	if err := os.WriteFile(filepath.Join(workspaceDir, "go.mod"), []byte(goMod), 0644); err != nil {
		_ = state.SandboxVerifyMachine.Transition(&current, state.VerifyFailed)
		return VerifyReport{FinalState: current, Error: err.Error()}
	}

	for _, f := range files {
		dest := filepath.Join(workspaceDir, f.Path)
		if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
			_ = state.SandboxVerifyMachine.Transition(&current, state.VerifyFailed)
			return VerifyReport{FinalState: current, Error: err.Error()}
		}
		if err := os.WriteFile(dest, []byte(f.Content), 0644); err != nil {
			_ = state.SandboxVerifyMachine.Transition(&current, state.VerifyFailed)
			return VerifyReport{FinalState: current, Error: err.Error()}
		}
	}

	runCmd := func(args ...string) (string, error) {
		cctx, cancel := context.WithTimeout(ctx, 45*time.Second)
		defer cancel()
		cmd := exec.CommandContext(cctx, "go", args...)
		cmd.Dir = workspaceDir
		out, err := cmd.CombinedOutput()
		return string(out), err
	}

	// 1. go build
	logBuf.WriteString("$ go build -buildvcs=false ./...\n")
	if out, err := runCmd("build", "-buildvcs=false", "./..."); err != nil {
		logBuf.WriteString(out)
		_ = state.SandboxVerifyMachine.Transition(&current, state.VerifyFailed)
		return VerifyReport{
			FinalState: current,
			Output:     logBuf.String(),
			Error:      "go build failed:\n" + out,
		}
	}
	logBuf.WriteString("ok (compiled cleanly)\n\n")

	// 2. go vet
	_ = state.SandboxVerifyMachine.Transition(&current, state.VerifyVet)
	logBuf.WriteString("$ go vet -buildvcs=false ./...\n")
	if out, err := runCmd("vet", "-buildvcs=false", "./..."); err != nil {
		logBuf.WriteString(out)
		_ = state.SandboxVerifyMachine.Transition(&current, state.VerifyFailed)
		return VerifyReport{
			FinalState:  current,
			BuildPassed: true,
			Output:      logBuf.String(),
			Error:       "go vet failed:\n" + out,
		}
	}
	logBuf.WriteString("ok (zero static analysis warnings)\n\n")

	// 3. go test -v -race
	_ = state.SandboxVerifyMachine.Transition(&current, state.VerifyTestRace)
	logBuf.WriteString("$ go test -v -race -buildvcs=false ./...\n")
	testOut, err := runCmd("test", "-v", "-race", "-buildvcs=false", "./...")
	if err != nil && strings.Contains(testOut, "-race") && strings.Contains(testOut, "not supported") {
		testOut, err = runCmd("test", "-v", "-buildvcs=false", "./...")
	}
	logBuf.WriteString(testOut)
	if err != nil {
		_ = state.SandboxVerifyMachine.Transition(&current, state.VerifyFailed)
		return VerifyReport{
			FinalState:  current,
			BuildPassed: true,
			VetPassed:   true,
			Output:      logBuf.String(),
			Error:       "go test failed:\n" + testOut,
		}
	}

	cases := make([]string, 0)
	for _, m := range testRunRe.FindAllStringSubmatch(testOut, -1) {
		cases = append(cases, m[1])
	}

	// 4. Specification constraints
	if len(constraints) > 0 {
		_ = state.SandboxVerifyMachine.Transition(&current, state.VerifyConstraints)
		for _, c := range constraints {
			if strings.EqualFold(c.Type, "must_not") && strings.TrimSpace(c.Description) != "" {
				for _, f := range files {
					if strings.Contains(f.Content, c.Description) {
						_ = state.SandboxVerifyMachine.Transition(&current, state.VerifyFailed)
						msg := fmt.Sprintf("violated must_not constraint %q (%s) in %s", c.ID, c.Description, f.Path)
						logBuf.WriteString("\nCONSTRAINT FAILURE: " + msg + "\n")
						return VerifyReport{
							FinalState:  current,
							BuildPassed: true,
							VetPassed:   true,
							TestsPassed: true,
							TestCases:   cases,
							Output:      logBuf.String(),
							Error:       msg,
						}
					}
				}
			}
		}
	}

	_ = state.SandboxVerifyMachine.Transition(&current, state.VerifyPassed)
	return VerifyReport{
		FinalState:  current,
		BuildPassed: true,
		VetPassed:   true,
		TestsPassed: true,
		TestCases:   cases,
		Output:      logBuf.String(),
	}
}

func Promote(workspaceDir, generatedRoot, targetSlug string) (string, error) {
	cleanSlug := sanitizeSlug(targetSlug)
	destDir := filepath.Join(generatedRoot, cleanSlug)

	absRoot, err := filepath.Abs(generatedRoot)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(absRoot, 0755); err != nil {
		return "", err
	}
	absDest, err := filepath.Abs(destDir)
	if err != nil {
		return "", err
	}
	rel, err := filepath.Rel(absRoot, absDest)
	if err != nil || strings.HasPrefix(rel, "..") {
		return "", fmt.Errorf("invalid target path outside generated/")
	}

	if err := os.MkdirAll(absDest, 0755); err != nil {
		return "", err
	}

	entries, err := os.ReadDir(workspaceDir)
	if err != nil {
		return "", err
	}
	copied := 0
	for _, ent := range entries {
		if ent.IsDir() || !strings.HasSuffix(ent.Name(), ".go") {
			continue
		}
		b, err := os.ReadFile(filepath.Join(workspaceDir, ent.Name()))
		if err != nil {
			return "", err
		}
		if err := os.WriteFile(filepath.Join(absDest, ent.Name()), b, 0644); err != nil {
			return "", err
		}
		copied++
	}
	if copied == 0 {
		return "", fmt.Errorf("no verified .go files found in workspace")
	}
	return filepath.ToSlash(filepath.Join("generated", cleanSlug)), nil
}

var slugRe = regexp.MustCompile(`[^a-z0-9-]+`)

func sanitizeSlug(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.TrimPrefix(s, "spec-")
	s = strings.ReplaceAll(s, " ", "-")
	s = strings.ReplaceAll(s, "_", "-")
	s = slugRe.ReplaceAllString(s, "")
	s = strings.Trim(s, "-")
	if s == "" {
		return "pkg"
	}
	if len(s) > 36 {
		s = strings.Trim(s[:36], "-")
	}
	return s
}

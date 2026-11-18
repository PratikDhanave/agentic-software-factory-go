package docs

import (
	"fmt"
	"log"

	"github.com/PratikDhanave/agentic-software-factory-go/pkg/llm"
)

type DocGenerator struct {
	ollama llm.Client
}

type Documentation struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	README      string `json:"readme"`
	APIDoc      string `json:"api_doc"`
	Examples    string `json:"examples"`
	Contributing string `json:"contributing"`
}

func NewDocGenerator(ollamaURL, model string) *DocGenerator {
	return &DocGenerator{
		ollama: llm.NewOllamaClient(ollamaURL, model),
	}
}

func NewDocGeneratorWithClient(client llm.Client) *DocGenerator {
	return &DocGenerator{ollama: client}
}

// GenerateREADME creates a README from code
func (dg *DocGenerator) GenerateREADME(projectName, description, code string) (string, error) {
	log.Printf("[DocGen] Generating README for: %s", projectName)

	prompt := fmt.Sprintf(`Generate a comprehensive README.md for this project:

Project: %s
Description: %s

Code overview:
%s

Include sections:
- Overview
- Installation
- Usage
- Features
- API Reference
- Examples
- Contributing
- License

Make it professional and clear.`, projectName, description, code[:min(len(code), 1000)])

	readme, err := dg.ollama.Analyze(prompt)
	if err != nil {
		return "", err
	}

	return readme, nil
}

// GenerateAPIDoc creates API documentation
func (dg *DocGenerator) GenerateAPIDoc(code, language string) (string, error) {
	log.Printf("[DocGen] Generating API documentation")

	prompt := fmt.Sprintf(`Generate comprehensive API documentation for this %s code:

%s

Format as markdown with:
- Function/method signatures
- Parameters and return types
- Examples for each function
- Error cases
- Performance notes`, language, code)

	apidoc, err := dg.ollama.Analyze(prompt)
	if err != nil {
		return "", err
	}

	return apidoc, nil
}

// GenerateExamples creates usage examples
func (dg *DocGenerator) GenerateExamples(code, language, useCase string) (string, error) {
	log.Printf("[DocGen] Generating examples for: %s", useCase)

	prompt := fmt.Sprintf(`Create 5 practical examples for using this %s code:

%s

Use case: %s

Format each example with:
- Description
- Code
- Expected output
- Comments`, language, code, useCase)

	examples, err := dg.ollama.Analyze(prompt)
	if err != nil {
		return "", err
	}

	return examples, nil
}

// GenerateFull creates complete documentation package
func (dg *DocGenerator) GenerateFull(projectName, description, code, language string) (*Documentation, error) {
	log.Printf("[DocGen] Generating full documentation for: %s", projectName)

	// Generate README
	readme, err := dg.GenerateREADME(projectName, description, code)
	if err != nil {
		return nil, fmt.Errorf("failed to generate README: %w", err)
	}

	// Generate API Doc
	apidoc, err := dg.GenerateAPIDoc(code, language)
	if err != nil {
		return nil, fmt.Errorf("failed to generate API doc: %w", err)
	}

	// Generate Examples
	examples, err := dg.GenerateExamples(code, language, projectName)
	if err != nil {
		return nil, fmt.Errorf("failed to generate examples: %w", err)
	}

	doc := &Documentation{
		Title:       projectName,
		Description: description,
		README:      readme,
		APIDoc:      apidoc,
		Examples:    examples,
		Contributing: generateContributingTemplate(projectName),
	}

	log.Printf("[DocGen] Documentation complete")
	return doc, nil
}

func generateContributingTemplate(projectName string) string {
	return fmt.Sprintf(`# Contributing to %s

We welcome contributions! Here's how to get started:

## Development Setup
1. Clone the repository
2. Install dependencies
3. Create a feature branch

## Code Style
- Follow the existing code style
- Add comments for complex logic
- Write tests for new features

## Testing
- Run tests: make test
- Ensure all tests pass before submitting PR
- Add tests for bug fixes

## Submitting Changes
1. Create a pull request with a clear description
2. Reference any related issues
3. Wait for code review
4. Address feedback

## Questions?
Feel free to open an issue for discussions.

Thank you for contributing!
`, projectName)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

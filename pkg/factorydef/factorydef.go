package factorydef

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// FactoryDefinition represents the complete factory configuration
type FactoryDefinition struct {
	Factory FactoryConfig `yaml:"factory"`
}

// FactoryConfig holds the factory configuration
type FactoryConfig struct {
	Name      string       `yaml:"name"`
	Version   string       `yaml:"version"`
	Agents    []AgentDef   `yaml:"agents"`
	Workflows []WorkflowDef `yaml:"workflows"`
	Runners   []RunnerDef  `yaml:"runners"`
}

// AgentDef defines an agent in the factory
type AgentDef struct {
	ID           string   `yaml:"id"`
	Role         string   `yaml:"role"`
	Capabilities []string `yaml:"capabilities"`
	Model        string   `yaml:"model"`
}

// WorkflowDef defines a workflow
type WorkflowDef struct {
	ID    string        `yaml:"id"`
	Steps []WorkflowStep `yaml:"steps"`
}

// WorkflowStep defines a step in a workflow
type WorkflowStep struct {
	Name       string `yaml:"name"`
	Agent      string `yaml:"agent"`
	Validation string `yaml:"validation,omitempty"`
}

// RunnerDef defines a runner/sandbox configuration
type RunnerDef struct {
	ID        string `yaml:"id"`
	Image     string `yaml:"image"`
	WorkDir   string `yaml:"work_dir"`
	ReadOnly  bool   `yaml:"read_only"`
	Timeout   string `yaml:"timeout"`
}

// LoadFactoryDefinition loads a factory definition from a YAML file
func LoadFactoryDefinition(path string) (*FactoryDefinition, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading factory definition: %w", err)
	}

	var def FactoryDefinition
	if err := yaml.Unmarshal(data, &def); err != nil {
		return nil, fmt.Errorf("parsing factory definition: %w", err)
	}

	return &def, nil
}

// FindFactoryDefinition searches for factory.yaml in standard locations
func FindFactoryDefinition(workingDir string) (string, error) {
	locations := []string{
		filepath.Join(workingDir, "specs", "factory.yaml"),
		filepath.Join(workingDir, "factory.yaml"),
		filepath.Join(workingDir, ".asf", "factory.yaml"),
	}

	for _, loc := range locations {
		if _, err := os.Stat(loc); err == nil {
			return loc, nil
		}
	}

	return "", fmt.Errorf("factory.yaml not found in standard locations")
}

// GetAgent returns an agent definition by ID
func (f *FactoryDefinition) GetAgent(id string) (*AgentDef, error) {
	for _, a := range f.Factory.Agents {
		if a.ID == id {
			return &a, nil
		}
	}
	return nil, fmt.Errorf("agent %s not found", id)
}

// GetWorkflow returns a workflow definition by ID
func (f *FactoryDefinition) GetWorkflow(id string) (*WorkflowDef, error) {
	for _, w := range f.Factory.Workflows {
		if w.ID == id {
			return &w, nil
		}
	}
	return nil, fmt.Errorf("workflow %s not found", id)
}

// GetRunner returns a runner definition by ID
func (f *FactoryDefinition) GetRunner(id string) (*RunnerDef, error) {
	for _, r := range f.Factory.Runners {
		if r.ID == id {
			return &r, nil
		}
	}
	return nil, fmt.Errorf("runner %s not found", id)
}
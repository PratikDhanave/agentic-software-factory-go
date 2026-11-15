package llm_test

import (
	"testing"

	"github.com/PratikDhanave/agentic-software-factory-go/pkg/eval"
	"github.com/PratikDhanave/agentic-software-factory-go/pkg/harness"
	"github.com/PratikDhanave/agentic-software-factory-go/pkg/llm"
)

// The client must satisfy both consumer interfaces.
var (
	_ harness.AgentClient = (*llm.Client)(nil)
	_ eval.LLMClient      = (*llm.Client)(nil)
)

func TestNewUsesDefaultModel(t *testing.T) {
	t.Setenv("ASF_MODEL", "")
	c := llm.New()
	if c.Model() != string(llm.DefaultModel) {
		t.Errorf("expected default model %s, got %s", llm.DefaultModel, c.Model())
	}
}

func TestModelOverride(t *testing.T) {
	t.Setenv("ASF_MODEL", "claude-opus-4-8")
	c := llm.New()
	if c.Model() != "claude-opus-4-8" {
		t.Errorf("expected overridden model, got %s", c.Model())
	}
}

func TestFromEnvWithoutCredentials(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "")
	t.Setenv("ANTHROPIC_AUTH_TOKEN", "")
	t.Setenv("ANTHROPIC_CONFIG_DIR", t.TempDir()) // empty: no credentials dir
	if c := llm.FromEnv(); c != nil {
		t.Error("expected nil client when no credentials are present")
	}
}

func TestFromEnvWithAPIKey(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "sk-ant-test")
	if c := llm.FromEnv(); c == nil {
		t.Error("expected client when ANTHROPIC_API_KEY is set")
	}
}

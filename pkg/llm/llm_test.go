package llm_test

import (
	"strings"
	"testing"

	"github.com/PratikDhanave/agentic-software-factory-go/pkg/eval"
	"github.com/PratikDhanave/agentic-software-factory-go/pkg/harness"
	"github.com/PratikDhanave/agentic-software-factory-go/pkg/llm"
)

// Every provider must satisfy both consumer interfaces.
var (
	_ harness.AgentClient = (llm.Client)(nil)
	_ eval.LLMClient      = (llm.Client)(nil)
)

// clearEnv isolates provider selection from the host machine.
func clearEnv(t *testing.T) {
	t.Helper()
	t.Setenv("ASF_LLM_PROVIDER", "")
	t.Setenv("ANTHROPIC_API_KEY", "")
	t.Setenv("ANTHROPIC_AUTH_TOKEN", "")
	t.Setenv("OPENAI_API_KEY", "")
	t.Setenv("ASF_LLM_BASE_URL", "")
	t.Setenv("ASF_LLM_API_KEY", "")
	t.Setenv("ASF_MODEL", "")
	t.Setenv("ASF_ENABLE_LLM", "")
	t.Setenv("ANTHROPIC_CONFIG_DIR", t.TempDir()) // empty: no ant profile
}

func TestNoConfigurationIsPromptOnly(t *testing.T) {
	clearEnv(t)
	if c := llm.FromEnv(); c != nil {
		t.Errorf("expected nil client, got provider %s", c.Provider())
	}
	if msg := llm.ConfigError(); msg != "" {
		t.Errorf("expected no config error, got %q", msg)
	}
}

func TestAnthropicAutoDetected(t *testing.T) {
	clearEnv(t)
	t.Setenv("ANTHROPIC_API_KEY", "sk-ant-test")
	c := llm.FromEnv()
	if c == nil || c.Provider() != "anthropic" {
		t.Fatalf("expected anthropic provider, got %v", c)
	}
	if c.Model() == "" {
		t.Error("anthropic provider should have a default model")
	}
}

func TestModelOverrideAppliesToAnthropic(t *testing.T) {
	clearEnv(t)
	t.Setenv("ANTHROPIC_API_KEY", "sk-ant-test")
	t.Setenv("ASF_MODEL", "claude-opus-4-8")
	if c := llm.FromEnv(); c.Model() != "claude-opus-4-8" {
		t.Errorf("expected overridden model, got %s", c.Model())
	}
}

func TestOpenAIAutoDetected(t *testing.T) {
	clearEnv(t)
	t.Setenv("OPENAI_API_KEY", "sk-test")
	t.Setenv("ASF_MODEL", "some-model")
	c := llm.FromEnv()
	if c == nil || c.Provider() != "openai" {
		t.Fatalf("expected openai provider, got %v", c)
	}
	if c.Model() != "some-model" {
		t.Errorf("expected ASF_MODEL to apply, got %s", c.Model())
	}
}

func TestOpenAIWithoutModelIsPromptOnlyWithExplanation(t *testing.T) {
	clearEnv(t)
	t.Setenv("OPENAI_API_KEY", "sk-test")
	if c := llm.FromEnv(); c != nil {
		t.Fatalf("expected nil client without ASF_MODEL, got %s", c.Provider())
	}
	if msg := llm.ConfigError(); !strings.Contains(msg, "ASF_MODEL") {
		t.Errorf("expected ConfigError to mention ASF_MODEL, got %q", msg)
	}
}

func TestOpenAICompatibleNeedsNoKey(t *testing.T) {
	clearEnv(t)
	t.Setenv("ASF_LLM_PROVIDER", "openai-compatible")
	t.Setenv("ASF_LLM_BASE_URL", "http://localhost:11434/v1")
	t.Setenv("ASF_MODEL", "local-model")
	c := llm.FromEnv()
	if c == nil || c.Provider() != "openai-compatible" {
		t.Fatalf("expected openai-compatible provider, got %v", c)
	}
}

func TestOpenAICompatibleWithoutBaseURL(t *testing.T) {
	clearEnv(t)
	t.Setenv("ASF_LLM_PROVIDER", "openai-compatible")
	t.Setenv("ASF_MODEL", "local-model")
	if c := llm.FromEnv(); c != nil {
		t.Fatal("expected nil client without ASF_LLM_BASE_URL")
	}
	if msg := llm.ConfigError(); !strings.Contains(msg, "ASF_LLM_BASE_URL") {
		t.Errorf("expected ConfigError to mention ASF_LLM_BASE_URL, got %q", msg)
	}
}

func TestUnknownProviderIsExplained(t *testing.T) {
	clearEnv(t)
	t.Setenv("ASF_LLM_PROVIDER", "mystery")
	if c := llm.FromEnv(); c != nil {
		t.Fatal("expected nil client for unknown provider")
	}
	if msg := llm.ConfigError(); !strings.Contains(msg, "mystery") {
		t.Errorf("expected ConfigError to name the unknown provider, got %q", msg)
	}
}

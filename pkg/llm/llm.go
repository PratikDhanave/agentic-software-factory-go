// Package llm provides the language-model layer for the factory. It is
// provider-agnostic: every provider implements Client (which satisfies both
// harness.AgentClient and eval.LLMClient), and FromEnv selects one from
// configuration.
//
// Provider selection (ASF_LLM_PROVIDER, or auto-detected when unset):
//
//	anthropic          Anthropic Messages API via the official SDK.
//	                   Credentials: ANTHROPIC_API_KEY / ANTHROPIC_AUTH_TOKEN,
//	                   or an `ant auth login` profile (requires the explicit
//	                   opt-in ASF_ENABLE_LLM=1, since profiles are ambient).
//	openai             OpenAI Chat Completions API. Credentials: OPENAI_API_KEY.
//	ollama             A local Ollama server. Zero config: defaults to
//	                   http://localhost:11434 (override with OLLAMA_HOST or
//	                   ASF_LLM_BASE_URL) and auto-picks the first installed
//	                   chat model when ASF_MODEL is unset. A running Ollama
//	                   is also auto-detected when no other credentials exist.
//	openai-compatible  Any other server speaking the Chat Completions wire
//	                   format (vLLM, Groq, Mistral, LM Studio, ...).
//	                   Config: ASF_LLM_BASE_URL (required), ASF_LLM_API_KEY
//	                   (optional — local servers often need none).
//
// ASF_MODEL selects the model for any provider. Without it, the anthropic
// provider has a default; the others require it explicitly.
//
// When nothing is configured, FromEnv returns nil and the factory runs in
// prompt-only mode — no behavior breaks without credentials.
package llm

import (
	"context"
	"os"
	"path/filepath"
)

// systemPrompt is intentionally stable (byte-identical across requests) so
// provider-side prompt caches stay warm; per-task content goes in the user
// message.
const systemPrompt = `You are a software engineering agent inside the Agentic Software Factory.
You receive a machine-readable specification and project context, and produce
the requested artifact (code, review findings, or a refactoring plan).
Follow the specification's constraints exactly. Output only the artifact —
no preamble.`

// Client is a language-model provider. It satisfies harness.AgentClient and
// eval.LLMClient (both declare Generate(ctx, prompt) (string, error)).
type Client interface {
	Generate(ctx context.Context, prompt string) (string, error)
	Model() string
	Provider() string
}

// FromEnv selects and constructs a provider from the environment, or returns
// nil when the factory should run in prompt-only mode.
func FromEnv() Client {
	switch os.Getenv("ASF_LLM_PROVIDER") {
	case "anthropic":
		return newAnthropic()
	case "openai":
		return newOpenAICompatible("openai", "https://api.openai.com/v1", os.Getenv("OPENAI_API_KEY"))
	case "openai-compatible":
		return newOpenAICompatible("openai-compatible", os.Getenv("ASF_LLM_BASE_URL"), os.Getenv("ASF_LLM_API_KEY"))
	case "ollama":
		return newOllama()
	case "":
		// Auto-detect: explicit credentials first, then a locally running
		// Ollama (free, so safe to auto-enable), then the ambient profile
		// behind its opt-in.
		if os.Getenv("ANTHROPIC_API_KEY") != "" || os.Getenv("ANTHROPIC_AUTH_TOKEN") != "" {
			return newAnthropic()
		}
		if os.Getenv("OPENAI_API_KEY") != "" {
			return newOpenAICompatible("openai", "https://api.openai.com/v1", os.Getenv("OPENAI_API_KEY"))
		}
		if c := detectOllama(); c != nil {
			return c
		}
		if profileCredentialsExist() && os.Getenv("ASF_ENABLE_LLM") == "1" {
			return newAnthropic()
		}
		return nil
	default:
		// Unknown provider name: fail safe into prompt-only mode; the
		// factory logs the configuration problem via ConfigError.
		return nil
	}
}

// ConfigError explains why FromEnv returned nil despite partial
// configuration, or returns "" when the state is unsurprising.
func ConfigError() string {
	switch p := os.Getenv("ASF_LLM_PROVIDER"); p {
	case "", "anthropic", "openai":
		openaiSelected := p == "openai" ||
			(p == "" && os.Getenv("ANTHROPIC_API_KEY") == "" && os.Getenv("ANTHROPIC_AUTH_TOKEN") == "" && os.Getenv("OPENAI_API_KEY") != "")
		if openaiSelected && os.Getenv("ASF_MODEL") == "" {
			return "the openai provider requires ASF_MODEL (no default model is assumed)"
		}
		if ProfileRequiresOptIn() {
			return "found an `ant auth login` profile but ASF_ENABLE_LLM is not set — set ASF_ENABLE_LLM=1 to spend those credentials"
		}
		return ""
	case "openai-compatible":
		if os.Getenv("ASF_LLM_BASE_URL") == "" {
			return "ASF_LLM_PROVIDER=openai-compatible requires ASF_LLM_BASE_URL"
		}
		if os.Getenv("ASF_MODEL") == "" {
			return "ASF_LLM_PROVIDER=openai-compatible requires ASF_MODEL"
		}
		return ""
	case "ollama":
		base := ollamaBaseURL()
		if !ollamaReachable(base) {
			return "ollama is not reachable at " + base + " — start it with `ollama serve` (or set OLLAMA_HOST / ASF_LLM_BASE_URL)"
		}
		if os.Getenv("ASF_MODEL") == "" && firstOllamaChatModel(base) == "" {
			return "ollama is running but has no chat-capable model — `ollama pull <model>` or set ASF_MODEL"
		}
		return ""
	default:
		return "unknown ASF_LLM_PROVIDER " + p + " (expected anthropic, openai, or openai-compatible)"
	}
}

// ProfileRequiresOptIn reports whether an ambient `ant auth login` profile
// exists that would be used if ASF_ENABLE_LLM=1 were set.
func ProfileRequiresOptIn() bool {
	return os.Getenv("ANTHROPIC_API_KEY") == "" &&
		os.Getenv("ANTHROPIC_AUTH_TOKEN") == "" &&
		os.Getenv("OPENAI_API_KEY") == "" &&
		os.Getenv("ASF_ENABLE_LLM") != "1" &&
		profileCredentialsExist()
}

// profileCredentialsExist reports whether an `ant auth login` profile is on
// disk. A heuristic — the definitive check is the first API call.
func profileCredentialsExist() bool {
	configDir := os.Getenv("ANTHROPIC_CONFIG_DIR")
	if configDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return false
		}
		configDir = filepath.Join(home, ".config", "anthropic")
	}
	entries, err := os.ReadDir(filepath.Join(configDir, "credentials"))
	return err == nil && len(entries) > 0
}

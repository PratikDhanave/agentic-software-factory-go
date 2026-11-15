// Package llm provides the Anthropic-backed language model client for the
// factory. It satisfies both harness.AgentClient and eval.LLMClient (both
// declare Generate(ctx, prompt) (string, error)).
//
// Credentials resolve through the SDK's standard chain (ANTHROPIC_API_KEY,
// ANTHROPIC_AUTH_TOKEN, or an `ant auth login` profile). When none are
// present, FromEnv returns nil and the factory falls back to prompt-only
// mode — no behavior breaks without a key.
package llm

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
)

// DefaultModel is used unless ASF_MODEL overrides it.
const DefaultModel = anthropic.ModelClaudeOpus4_8

// systemPrompt is intentionally stable (byte-identical across requests) so
// the prefix stays cacheable; per-task content goes in the user message.
const systemPrompt = `You are a software engineering agent inside the Agentic Software Factory.
You receive a machine-readable specification and project context, and produce
the requested artifact (code, review findings, or a refactoring plan).
Follow the specification's constraints exactly. Output only the artifact —
no preamble.`

// Client calls the Anthropic Messages API.
type Client struct {
	client    anthropic.Client
	model     anthropic.Model
	maxTokens int64
}

// New constructs a client using the SDK's default credential resolution.
func New() *Client {
	model := DefaultModel
	if m := os.Getenv("ASF_MODEL"); m != "" {
		model = anthropic.Model(m)
	}
	return &Client{
		client:    anthropic.NewClient(),
		model:     model,
		maxTokens: 16000,
	}
}

// FromEnv returns a ready client when credentials are available and their
// use is authorized, or nil when the factory should run in prompt-only mode.
//
// An env var (ANTHROPIC_API_KEY / ANTHROPIC_AUTH_TOKEN) is an explicit act
// for this process and enables the LLM directly. An `ant auth login` profile
// on disk is ambient — it was created for other tools, so spending it here
// additionally requires the explicit opt-in ASF_ENABLE_LLM=1.
func FromEnv() *Client {
	if os.Getenv("ANTHROPIC_API_KEY") != "" || os.Getenv("ANTHROPIC_AUTH_TOKEN") != "" {
		return New()
	}
	if profileCredentialsExist() {
		if os.Getenv("ASF_ENABLE_LLM") == "1" {
			return New()
		}
		// Ambient profile present but not opted in — stay in prompt-only
		// mode. The factory logs how to enable it.
		return nil
	}
	return nil
}

// ProfileRequiresOptIn reports whether an ambient profile credential exists
// that would be used if ASF_ENABLE_LLM=1 were set.
func ProfileRequiresOptIn() bool {
	return os.Getenv("ANTHROPIC_API_KEY") == "" &&
		os.Getenv("ANTHROPIC_AUTH_TOKEN") == "" &&
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

// Model returns the configured model ID.
func (c *Client) Model() string { return string(c.model) }

// Generate sends a prompt and returns the model's text output. It satisfies
// harness.AgentClient and eval.LLMClient.
func (c *Client) Generate(ctx context.Context, prompt string) (string, error) {
	adaptive := anthropic.ThinkingConfigAdaptiveParam{}
	resp, err := c.client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     c.model,
		MaxTokens: c.maxTokens,
		Thinking:  anthropic.ThinkingConfigParamUnion{OfAdaptive: &adaptive},
		System: []anthropic.TextBlockParam{{
			Text:         systemPrompt,
			CacheControl: anthropic.NewCacheControlEphemeralParam(),
		}},
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(prompt)),
		},
	})
	if err != nil {
		var apierr *anthropic.Error
		if errors.As(err, &apierr) {
			switch apierr.StatusCode {
			case 401:
				return "", fmt.Errorf("anthropic auth failed (check ANTHROPIC_API_KEY or `ant auth login`): %w", err)
			case 429:
				return "", fmt.Errorf("anthropic rate limited: %w", err)
			default:
				return "", fmt.Errorf("anthropic API error (HTTP %d): %w", apierr.StatusCode, err)
			}
		}
		return "", fmt.Errorf("anthropic request failed: %w", err)
	}

	if resp.StopReason == anthropic.StopReasonRefusal {
		return "", fmt.Errorf("model declined the request (category: %s)", resp.StopDetails.Category)
	}

	var out strings.Builder
	for _, block := range resp.Content {
		if text, ok := block.AsAny().(anthropic.TextBlock); ok {
			out.WriteString(text.Text)
		}
	}
	if resp.StopReason == anthropic.StopReasonMaxTokens {
		return out.String(), fmt.Errorf("output truncated at %d tokens", c.maxTokens)
	}
	return out.String(), nil
}

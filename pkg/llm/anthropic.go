package llm

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
)

// defaultAnthropicModel is used when ASF_MODEL is unset.
const defaultAnthropicModel = anthropic.ModelClaudeOpus4_8

// anthropicClient implements Client against the Anthropic Messages API.
type anthropicClient struct {
	client    anthropic.Client
	model     anthropic.Model
	maxTokens int64
}

func newAnthropic() Client {
	model := defaultAnthropicModel
	if m := os.Getenv("ASF_MODEL"); m != "" {
		model = anthropic.Model(m)
	}
	return &anthropicClient{
		client:    anthropic.NewClient(), // SDK resolves env vars / ant profile
		model:     model,
		maxTokens: 16000,
	}
}

func (c *anthropicClient) Provider() string { return "anthropic" }
func (c *anthropicClient) Model() string    { return string(c.model) }

func (c *anthropicClient) Generate(ctx context.Context, prompt string) (string, error) {
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

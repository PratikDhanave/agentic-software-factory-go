# 🤖 Model-Agnostic: Use ANY LLM Provider

Your Factory Stack now supports **any LLM provider**. Switch between them with one config change.

## Supported Providers

| Provider | Model | Setup | Cost | Latency |
|----------|-------|-------|------|---------|
| **Claude** | claude-3-5-sonnet | API key | $ | Medium |
| **OpenAI** | GPT-4 Turbo | API key | $$ | Medium |
| **Ollama** | llama2, mistral, neural-chat | Local | Free | Low |
| **Mock** | None | Built-in | Free | Instant |

## Quick Start: Choose Your Provider

### Option 1: Use Ollama (Local, Free, Private)

```bash
# 1. Install Ollama: https://ollama.ai

# 2. Download a model
ollama pull llama2
# or: ollama pull mistral

# 3. Start Ollama
ollama serve

# 4. Update .env
LLM_PROVIDER=ollama
OLLAMA_MODEL=llama2

# 5. Run Phase 2
./bin/phase2-simulator
```

**Result:** Complete local control, no API keys, no costs, fully private.

### Option 2: Use Claude (Powerful, Fast)

```bash
# 1. Get API key: https://console.anthropic.com/account/keys

# 2. Update .env
LLM_PROVIDER=claude
CLAUDE_API_KEY=sk-ant-YOUR_KEY_HERE
CLAUDE_MODEL=claude-3-5-sonnet-20241022

# 3. Run Phase 2
./bin/phase2-server
```

### Option 3: Use OpenAI (GPT-4)

```bash
# 1. Get API key: https://platform.openai.com/api-keys

# 2. Update .env
LLM_PROVIDER=openai
OPENAI_API_KEY=sk-YOUR_KEY_HERE
OPENAI_MODEL=gpt-4-turbo

# 3. Run Phase 2
./bin/phase2-server
```

### Option 4: Use Mock (Testing, No Dependencies)

```bash
# 1. Update .env
LLM_PROVIDER=mock

# 2. Run Phase 2
./bin/phase2-simulator
# Generates mock decisions instantly, stores to PostgreSQL
```

## Architecture

```go
// Your code is provider-agnostic
llmClient := services.NewLLMProvider(config)

// Works with ANY provider
decision, err := llmClient.GenerateDecision(ctx, request)
```

```
┌─────────────────────────────────────────────────┐
│          Your Application (Generic)             │
│       Uses LLMProvider interface only           │
└─────────────┬───────────────────────────────────┘
              │
    ┌─────────┼─────────┬────────────┐
    │         │         │            │
    ▼         ▼         ▼            ▼
  Claude    OpenAI   Ollama       Mock
  (API)     (API)   (Local)    (Testing)
```

## Switching Providers (1 Change)

Your entire system automatically adapts. Just update `.env`:

```bash
# Was using Claude
LLM_PROVIDER=claude

# Switch to Ollama (no code changes!)
LLM_PROVIDER=ollama

# Back to Claude (no code changes!)
LLM_PROVIDER=claude
```

## Model Comparison

### For Development/Testing: Ollama
- Run locally on your machine
- No API keys needed
- Zero latency (instant feedback)
- Models: llama2, mistral, neural-chat
- Perfect for rapid iteration

### For Production: Claude or OpenAI
- Claude: fastest, most reliable
- OpenAI: GPT-4 power, well-known
- Both have enterprise support
- Appropriate for real workflows

### For CI/CD: Mock Provider
- No external dependencies
- Instant responses (testing)
- Perfect for automated testing
- Deterministic output

## Real-World Example: Multi-Model Workflow

```go
// Monday morning: Use mock for fast CI/CD
os.Setenv("LLM_PROVIDER", "mock")
test_suite()  // Instant feedback

// Development: Use Ollama locally
os.Setenv("LLM_PROVIDER", "ollama")
dev_server()  // Fast iteration, free

// Demo: Use Claude for best quality
os.Setenv("LLM_PROVIDER", "claude")
demo()  // Impressive, production-ready

// Production: Choose based on cost vs quality
os.Setenv("LLM_PROVIDER", "claude")  // Best quality
os.Setenv("LLM_PROVIDER", "openai")  // Alternative
```

## Configuration Reference

### Claude
```env
LLM_PROVIDER=claude
CLAUDE_API_KEY=sk-ant-...
CLAUDE_MODEL=claude-3-5-sonnet-20241022
```

### OpenAI
```env
LLM_PROVIDER=openai
OPENAI_API_KEY=sk-...
OPENAI_MODEL=gpt-4-turbo
# Optional:
# OPENAI_BASE_URL=https://api.openai.com/v1
```

### Ollama
```env
LLM_PROVIDER=ollama
OLLAMA_MODEL=llama2
# Optional (defaults to localhost:11434):
# OLLAMA_BASE_URL=http://localhost:11434
```

### Mock
```env
LLM_PROVIDER=mock
# No other configuration needed
```

## Installing Local Models (Ollama)

```bash
# Install Ollama (macOS, Linux, Windows)
# https://ollama.ai

# Start Ollama server
ollama serve

# In another terminal, download models
ollama pull llama2           # 4B params, 3.8GB
ollama pull mistral          # 7B params, 4.1GB  
ollama pull neural-chat      # 7B params, 4.0GB
ollama pull openchat         # 7B params, 3.9GB

# List installed models
ollama list

# Run specific model
ollama run mistral
```

**Models ranked by speed:** openchat > neural-chat > mistral > llama2  
**Models ranked by quality:** mistral ≈ neural-chat > openchat > llama2

## Extending: Add Your Own Provider

Want to support another model? Add a provider adapter:

```go
// internal/domain/services/provider_custom.go

type CustomProvider struct {
    apiKey string
    model  string
}

func (c *CustomProvider) GenerateDecision(ctx context.Context, req *DecisionRequest) (*DecisionResponse, error) {
    // Implement your provider's API
    // Must return: decision, reasoning, confidence, alternatives
}

// Register in llm_provider.go
case "custom":
    return NewCustomProvider(config)
```

Then use it:

```env
LLM_PROVIDER=custom
CUSTOM_API_KEY=...
CUSTOM_MODEL=...
```

## Performance Characteristics

```
Provider     Latency    Cost      Quality   Setup
─────────────────────────────────────────────────
Mock         <10ms      Free      N/A       Built-in
Ollama       50-500ms   Free      Good      Local
Claude       200-800ms  $         Excellent API
OpenAI       200-800ms  $$        Excellent API
```

## Recommended Setup By Use Case

### I'm Building & Testing (Local Development)
```env
LLM_PROVIDER=mock  # or ollama for more realistic
```

### I Want Privacy & Control (Self-Hosted)
```env
LLM_PROVIDER=ollama
OLLAMA_MODEL=mistral
```

### I Want Best Quality (Production)
```env
LLM_PROVIDER=claude
CLAUDE_MODEL=claude-3-5-sonnet-20241022
```

### I Want Cost-Effective (Alternative)
```env
LLM_PROVIDER=openai
OPENAI_MODEL=gpt-4-turbo
```

## Cost Estimates (Monthly)

| Provider | Usage | Cost |
|----------|-------|------|
| Ollama | Unlimited | $0 |
| Mock | Unlimited | $0 |
| Claude | 10k decisions | ~$1 |
| OpenAI | 10k decisions | ~$2 |

**At scale, costs are minimal compared to development speed gains.**

## Troubleshooting

### "Connection refused" with Ollama
```bash
# Make sure Ollama is running
ollama serve

# Check it's listening
curl http://localhost:11434/api/tags
```

### "API key invalid" with Claude/OpenAI
```bash
# Verify key format
# Claude: sk-ant-...
# OpenAI: sk-...

# Test key directly
curl -H "Authorization: Bearer YOUR_KEY" https://api.anthropic.com/v1/...
```

### Mock provider always returns same response
```bash
# That's by design! It's for testing.
# Use ollama or claude for realistic variety.
```

## Next: Try It

1. **Pick a provider** (I recommend Ollama for local dev)
2. **Update .env** with one line
3. **Run:** `./bin/phase2-simulator`
4. **Switch providers:** Just change one env var
5. **Feel the difference:** Compare speed, quality, cost

---

**Your Factory Stack is now truly vendor-agnostic. 🚀**

No lock-in. Full flexibility. Any model. Instant switching.

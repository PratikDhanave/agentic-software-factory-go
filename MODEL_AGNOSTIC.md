# Model-Agnostic LLM Architecture

Your Factory Stack supports any LLM provider with zero code changes.

## Supported Providers

| Provider | Model | Setup | Cost | Status |
|----------|-------|-------|------|--------|
| Claude | claude-3-5-sonnet | API key | $ | Supported |
| OpenAI | gpt-4-turbo | API key | $$ | Supported |
| Ollama | llama2, mistral | Local | Free | Supported |
| Mock | None | Built-in | Free | Supported |

## How to Use

Update `.env` with your chosen provider:

```bash
# Option 1: Local (Ollama)
LLM_PROVIDER=ollama
OLLAMA_MODEL=llama2

# Option 2: Claude API
LLM_PROVIDER=claude
CLAUDE_API_KEY=sk-ant-YOUR_KEY
CLAUDE_MODEL=claude-3-5-sonnet-20241022

# Option 3: OpenAI API
LLM_PROVIDER=openai
OPENAI_API_KEY=sk-YOUR_KEY
OPENAI_MODEL=gpt-4-turbo

# Option 4: Mock (testing)
LLM_PROVIDER=mock
```

Then run:
```bash
./bin/phase2-simulator
```

## Switching Providers

Change one environment variable and restart:

```bash
# Was using Ollama
LLM_PROVIDER=ollama

# Switch to Claude (no code changes)
LLM_PROVIDER=claude

# Switch to OpenAI (no code changes)
LLM_PROVIDER=openai

# Switch back to local (no code changes)
LLM_PROVIDER=ollama
```

**All application code remains unchanged.**

## Architecture

```go
// Your code
llmClient := services.NewLLMProvider(config)
decision, _ := llmClient.GenerateDecision(ctx, request)

// Works with ANY provider automatically
```

## Supported Models

### Ollama (Local)
```bash
ollama pull llama2           # 4B, ~3.8GB
ollama pull mistral          # 7B, ~4.1GB
ollama pull neural-chat      # 7B, ~4.0GB
ollama pull openchat         # 7B, ~3.9GB
```

### Cloud APIs
- Claude: claude-3-5-sonnet-20241022
- OpenAI: gpt-4-turbo, gpt-4, gpt-3.5-turbo

## Implementation

Provider interface (vendor-independent):
```go
type LLMProvider interface {
    GenerateDecision(ctx context.Context, req *DecisionRequest) (*DecisionResponse, error)
    AnalyzeCode(ctx context.Context, code string) (map[string]interface{}, error)
    ReviewSpecification(ctx context.Context, spec string) (map[string]interface{}, error)
    TriageIssue(ctx context.Context, issue string) (map[string]interface{}, error)
}
```

Supported implementations:
- `ClaudeProvider` - Claude API
- `OpenAIProvider` - OpenAI API
- `OllamaProvider` - Local Ollama
- `MockProvider` - Testing

## Configuration Reference

### Ollama
```env
LLM_PROVIDER=ollama
OLLAMA_MODEL=llama2
OLLAMA_BASE_URL=http://localhost:11434  # optional
```

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
```

### Mock
```env
LLM_PROVIDER=mock
```

## Adding Custom Providers

To add a new provider:

1. Create `internal/domain/services/provider_mymodel.go`
2. Implement the `LLMProvider` interface
3. Register in `llm_provider.go`:
   ```go
   case "mymodel":
       return NewMyModelProvider(config)
   ```
4. Use it:
   ```env
   LLM_PROVIDER=mymodel
   MYMODEL_API_KEY=...
   ```

## Files

- `internal/domain/services/llm_provider.go` - Base interface
- `internal/domain/services/provider_claude.go` - Claude implementation
- `internal/domain/services/provider_openai.go` - OpenAI implementation
- `internal/domain/services/provider_ollama.go` - Ollama implementation

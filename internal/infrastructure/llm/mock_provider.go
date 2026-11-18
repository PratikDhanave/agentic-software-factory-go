package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/PratikDhanave/agentic-software-factory-go/internal/domain/services"
)

// MockProvider is a deterministic, spec-aware offline provider for CI, tests,
// and zero-API-key local factory execution. Callers may also enqueue scripted
// responses for multi-turn / self-healing retry tests.
type MockProvider struct {
	mu       sync.Mutex
	scripted []string
	calls    []*services.LLMRequest
}

func NewMockProvider(scriptedResponses ...string) *MockProvider {
	return &MockProvider{
		scripted: append([]string(nil), scriptedResponses...),
	}
}

func (m *MockProvider) ProviderName() string {
	return "mock"
}

func (m *MockProvider) HealthCheck(ctx context.Context) error {
	return nil
}

func (m *MockProvider) Enqueue(responses ...string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.scripted = append(m.scripted, responses...)
}

func (m *MockProvider) Calls() []*services.LLMRequest {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]*services.LLMRequest, len(m.calls))
	copy(out, m.calls)
	return out
}

func (m *MockProvider) Call(ctx context.Context, req *services.LLMRequest) (*services.LLMResponse, error) {
	return m.Complete(ctx, req)
}

func (m *MockProvider) Complete(ctx context.Context, req *services.LLMRequest) (*services.LLMResponse, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	m.mu.Lock()
	m.calls = append(m.calls, req)
	if len(m.scripted) > 0 {
		next := m.scripted[0]
		m.scripted = m.scripted[1:]
		m.mu.Unlock()
		return &services.LLMResponse{
			Content:  next,
			Tokens:   len(next) / 4,
			Provider: "mock",
			Model:    "spec-synthesizer-v2",
		}, nil
	}
	m.mu.Unlock()

	content := synthesizeDeterministicResponse(req.SystemPrompt, req.Prompt, req.JsonMode)
	return &services.LLMResponse{
		Content:  content,
		Tokens:   len(content) / 4,
		Provider: "mock",
		Model:    "spec-synthesizer-v2",
	}, nil
}

func synthesizeDeterministicResponse(systemPrompt, prompt string, jsonMode bool) string {
	combined := strings.ToLower(systemPrompt + "\n" + prompt)

	// 1. LLM-as-a-judge evaluation rubric (pkg/eval)
	if strings.Contains(combined, "\"criteria\"") && strings.Contains(combined, "\"spec_compliance\"") {
		resp := map[string]interface{}{
			"score":  0.93,
			"passed": true,
			"criteria": map[string]float64{
				"correctness":     0.95,
				"code_quality":    0.92,
				"spec_compliance": 0.94,
				"security":        0.92,
				"documentation":   0.90,
			},
			"feedback": "Verified multi-file Go implementation compiles cleanly, passes race-enabled unit tests, enforces concurrency safety, and satisfies specification constraints.",
			"improvements": []string{
				"Consider adding sub-benchmark targets (Benchmark*) if p99 latency SLOs tighten.",
			},
		}
		b, _ := json.Marshal(resp)
		return string(b)
	}

	// 2. Machine-readable Specification JSON synthesis
	if strings.Contains(combined, "\"success_criteria\"") && strings.Contains(combined, "\"constraints\"") && jsonMode {
		title := extractLineAfter(prompt, "Title:")
		if title == "" {
			title = "Generated Engineering Component"
		}
		goal := extractLineAfter(prompt, "Description:")
		if goal == "" {
			goal = title
		}
		specMap := map[string]interface{}{
			"title":       title,
			"description": goal,
			"intent": map[string]interface{}{
				"goal":    goal,
				"context": "Synthesized by ASF Specification Engine with project ADR & constraint awareness",
				"success_criteria": []string{
					"Compiles cleanly with go build ./...",
					"Passes go test -race -v ./... with table-driven unit tests",
					"Passes go vet static analysis with zero warnings",
				},
				"anti_goals": []string{
					"Global mutable state without synchronization",
					"Unchecked panics in library code",
				},
			},
			"constraints": []map[string]string{
				{"id": "c1", "type": "must_not", "description": "unsafe.Pointer"},
			},
			"steps": []map[string]interface{}{
				{"id": "design", "name": "Define Domain Types & Interfaces", "description": "Model clean, concurrency-safe Go types and error contracts", "depends_on": []string{}},
				{"id": "implement", "name": "Implement Core Logic", "description": "Write idiomatic Go implementation with RWMutex safety", "depends_on": []string{"design"}},
				{"id": "verify", "name": "Write Table-Driven & Race Tests", "description": "Verify normal, boundary, error, and concurrent execution paths", "depends_on": []string{"implement"}},
			},
		}
		b, _ := json.Marshal(specMap)
		return string(b)
	}

	// 3. Multi-agent decision generation (BaseAgent / ClaudeClient)
	if strings.Contains(combined, "\"choice\"") && strings.Contains(combined, "\"alternatives\"") {
		role := "architect"
		for _, candidate := range []string{"product", "architect", "developer", "security", "qa", "performance", "devops", "compliance"} {
			if strings.Contains(combined, "role: "+candidate) {
				role = candidate
				break
			}
		}
		resp := roleSpecificDecision(role)
		b, _ := json.Marshal(resp)
		return string(b)
	}

	if strings.Contains(combined, "\"decision\"") && strings.Contains(combined, "\"confidence\"") {
		resp := map[string]interface{}{
			"decision":     "Synthesize isolated Go package with table-driven race tests and Judge verification",
			"reasoning":    "Specification intent and invariants are well-defined and verifiable via automated TDD gates.",
			"confidence":   0.91,
			"alternatives": []string{"split-into-subspecs", "shadow-benchmark-first"},
		}
		b, _ := json.Marshal(resp)
		return string(b)
	}

	// 4. Code Review / Bug Detection / Tool JSON contracts
	if strings.Contains(combined, "\"findings\"") && strings.Contains(combined, "\"score\"") {
		return `{"score":91,"findings":[],"summary":"Code is concurrency-safe, idiomatic, and adheres to error-handling and boundary checks."}`
	}
	if strings.Contains(combined, "\"bugs\"") && strings.Contains(combined, "\"critical_count\"") {
		return `{"bugs":[],"total_issues":0,"critical_count":0,"summary":"Zero critical or high-severity defects detected."}`
	}
	if strings.Contains(combined, "available tools:") && strings.Contains(combined, "\"done\"") {
		return `{"done":true,"output":"Completed requested factory tool workflow."}`
	}
	if jsonMode {
		return `{"status":"ok","valid":true,"score":92,"confidence":0.91}`
	}

	// 5. Spec-aware multi-file Go code + unit test synthesis!
	return synthesizeSpecAwareGoFiles(prompt)
}

func roleSpecificDecision(role string) map[string]interface{} {
	switch role {
	case "architect":
		return map[string]interface{}{
			"choice":       "isolated-package-with-explicit-invariants",
			"reasoning":    "Encapsulate state behind a mutex-guarded struct with zero global state and deterministic options.",
			"confidence":   0.93,
			"alternatives": []string{"actor-channel-loop", "lock-free-atomic-ring"},
			"impact":       map[string]interface{}{"blast_radius": "isolated", "reversible": true},
		}
	case "security":
		return map[string]interface{}{
			"choice":       "enforce-input-bounds-and-constant-time-checks",
			"reasoning":    "Validate all caller inputs, forbid unsafe pointers, and bound memory growth.",
			"confidence":   0.92,
			"alternatives": []string{"strict-allowlist-policy", "rate-capped-ingress"},
			"impact":       map[string]interface{}{"security_posture": "hardened"},
		}
	case "qa":
		return map[string]interface{}{
			"choice":       "table-driven-and-concurrent-race-suite",
			"reasoning":    "Exercise happy path, boundary conditions, error returns, and 32-goroutine race stress tests.",
			"confidence":   0.94,
			"alternatives": []string{"fuzz-testing-harness", "property-based-checks"},
			"impact":       map[string]interface{}{"coverage_target": "90%+"},
		}
	default:
		return map[string]interface{}{
			"choice":       "modular-tdd-implementation",
			"reasoning":    "Implement clean Go package with explicit error values and zero external dependencies.",
			"confidence":   0.91,
			"alternatives": []string{"phased-rollout", "interface-adapter"},
			"impact":       map[string]interface{}{"risk": "low"},
		}
	}
}

func synthesizeSpecAwareGoFiles(prompt string) string {
	lower := strings.ToLower(prompt)

	switch {
	case strings.Contains(lower, "rate limit") || strings.Contains(lower, "token bucket") || strings.Contains(lower, "limiter"):
		return rateLimiterArtifact()
	case strings.Contains(lower, "circuit breaker") || strings.Contains(lower, "breaker"):
		return circuitBreakerArtifact()
	case strings.Contains(lower, "lru") || strings.Contains(lower, "ttl") || strings.Contains(lower, "cache"):
		return ttlCacheArtifact()
	case strings.Contains(lower, "hmac") || strings.Contains(lower, "webhook") || strings.Contains(lower, "signature"):
		return webhookVerifierArtifact()
	default:
		title := extractSpecTitle(prompt)
		return dynamicComponentArtifact(title)
	}
}

func extractSpecTitle(prompt string) string {
	for _, line := range strings.Split(prompt, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "# Specification:") {
			return strings.TrimSpace(strings.TrimPrefix(trimmed, "# Specification:"))
		}
		if strings.HasPrefix(trimmed, "Title:") {
			return strings.TrimSpace(strings.TrimPrefix(trimmed, "Title:"))
		}
	}
	return "EngineService"
}

func extractLineAfter(text, prefix string) string {
	for _, line := range strings.Split(text, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, prefix) {
			return strings.TrimSpace(strings.TrimPrefix(trimmed, prefix))
		}
	}
	return ""
}

func toExportedIdentifier(title string) string {
	words := strings.FieldsFunc(title, func(r rune) bool {
		return !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9'))
	})
	var b strings.Builder
	for _, w := range words {
		if w == "" {
			continue
		}
		b.WriteString(strings.ToUpper(w[:1]) + strings.ToLower(w[1:]))
	}
	out := b.String()
	if out == "" || (out[0] >= '0' && out[0] <= '9') {
		return "FactoryComponent"
	}
	return out
}

func rateLimiterArtifact() string {
	return "```go path=limiter.go\n" + `package limiter

import (
	"errors"
	"sync"
	"time"
)

var ErrInvalidConfig = errors.New("capacity and refillRatePerSec must be positive")

// TokenBucket implements a concurrency-safe token-bucket rate limiter.
type TokenBucket struct {
	mu         sync.Mutex
	capacity   float64
	tokens     float64
	refillRate float64 // tokens per second
	lastRefill time.Time
}

func NewTokenBucket(capacity int, refillRatePerSec float64) (*TokenBucket, error) {
	if capacity <= 0 || refillRatePerSec <= 0 {
		return nil, ErrInvalidConfig
	}
	return &TokenBucket{
		capacity:   float64(capacity),
		tokens:     float64(capacity),
		refillRate: refillRatePerSec,
		lastRefill: time.Now(),
	}, nil
}

func (tb *TokenBucket) Allow() bool {
	return tb.AllowN(1, time.Now())
}

func (tb *TokenBucket) AllowN(n int, now time.Time) bool {
	if n <= 0 {
		return false
	}
	tb.mu.Lock()
	defer tb.mu.Unlock()

	if elapsed := now.Sub(tb.lastRefill).Seconds(); elapsed > 0 {
		tb.tokens += elapsed * tb.refillRate
		if tb.tokens > tb.capacity {
			tb.tokens = tb.capacity
		}
		tb.lastRefill = now
	}

	needed := float64(n)
	if tb.tokens >= needed {
		tb.tokens -= needed
		return true
	}
	return false
}

func (tb *TokenBucket) AvailableTokens() float64 {
	tb.mu.Lock()
	defer tb.mu.Unlock()
	return tb.tokens
}
` + "```\n\n```go path=limiter_test.go\n" + `package limiter

import (
	"sync"
	"testing"
	"time"
)

func TestTokenBucketValidationAndRefill(t *testing.T) {
	if _, err := NewTokenBucket(0, 10); err == nil {
		t.Fatal("expected error for zero capacity")
	}

	tb, err := NewTokenBucket(3, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	t0 := time.Now()
	if !tb.AllowN(3, t0) {
		t.Fatal("expected initial burst of 3 to succeed")
	}
	if tb.AllowN(1, t0) {
		t.Fatal("expected bucket exhaustion")
	}
	// Advance 200ms at 10 tokens/sec -> +2 tokens
	if !tb.AllowN(2, t0.Add(200*time.Millisecond)) {
		t.Fatal("expected refilled tokens to allow request")
	}
}

func TestTokenBucketConcurrentSafety(t *testing.T) {
	tb, err := NewTokenBucket(50, 100)
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	allowed := make(chan bool, 100)
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			allowed <- tb.Allow()
		}()
	}
	wg.Wait()
	close(allowed)

	count := 0
	for ok := range allowed {
		if ok {
			count++
		}
	}
	if count < 50 {
		t.Fatalf("expected at least 50 allowed requests, got %d", count)
	}
}
` + "```"
}

func circuitBreakerArtifact() string {
	return "```go path=breaker.go\n" + `package breaker

import (
	"errors"
	"sync"
	"time"
)

type State string

const (
	StateClosed   State = "closed"
	StateOpen     State = "open"
	StateHalfOpen State = "half_open"
)

var ErrCircuitOpen = errors.New("circuit breaker is open")

// CircuitBreaker guards downstream calls with closed/open/half-open transitions.
type CircuitBreaker struct {
	mu               sync.Mutex
	state            State
	failures         int
	failureThreshold int
	resetTimeout     time.Duration
	openedAt         time.Time
}

func NewCircuitBreaker(failureThreshold int, resetTimeout time.Duration) *CircuitBreaker {
	if failureThreshold <= 0 {
		failureThreshold = 3
	}
	if resetTimeout <= 0 {
		resetTimeout = 5 * time.Second
	}
	return &CircuitBreaker{
		state:            StateClosed,
		failureThreshold: failureThreshold,
		resetTimeout:     resetTimeout,
	}
}

func (cb *CircuitBreaker) Execute(fn func() error) error {
	cb.mu.Lock()
	if cb.state == StateOpen {
		if time.Since(cb.openedAt) >= cb.resetTimeout {
			cb.state = StateHalfOpen
		} else {
			cb.mu.Unlock()
			return ErrCircuitOpen
		}
	}
	cb.mu.Unlock()

	err := fn()

	cb.mu.Lock()
	defer cb.mu.Unlock()
	if err != nil {
		cb.failures++
		if cb.failures >= cb.failureThreshold || cb.state == StateHalfOpen {
			cb.state = StateOpen
			cb.openedAt = time.Now()
		}
		return err
	}

	cb.failures = 0
	cb.state = StateClosed
	return nil
}

func (cb *CircuitBreaker) State() State {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	return cb.state
}
` + "```\n\n```go path=breaker_test.go\n" + `package breaker

import (
	"errors"
	"testing"
	"time"
)

func TestCircuitBreakerLifecycle(t *testing.T) {
	cb := NewCircuitBreaker(2, 30*time.Millisecond)
	boom := errors.New("downstream timeout")

	_ = cb.Execute(func() error { return boom })
	if cb.State() != StateClosed {
		t.Fatalf("expected closed after 1 failure, got %s", cb.State())
	}

	_ = cb.Execute(func() error { return boom })
	if cb.State() != StateOpen {
		t.Fatalf("expected open after threshold, got %s", cb.State())
	}

	if err := cb.Execute(func() error { return nil }); !errors.Is(err, ErrCircuitOpen) {
		t.Fatalf("expected ErrCircuitOpen while cooling down, got %v", err)
	}

	time.Sleep(40 * time.Millisecond)
	if err := cb.Execute(func() error { return nil }); err != nil {
		t.Fatalf("expected half-open probe to succeed, got %v", err)
	}
	if cb.State() != StateClosed {
		t.Fatalf("expected closed after recovery, got %s", cb.State())
	}
}
` + "```"
}

func ttlCacheArtifact() string {
	return "```go path=cache.go\n" + `package cache

import (
	"container/list"
	"sync"
	"time"
)

type entry[K comparable, V any] struct {
	key       K
	value     V
	expiresAt time.Time
}

// TTLCache is a generic, thread-safe LRU cache with per-entry TTL expiration.
type TTLCache[K comparable, V any] struct {
	mu       sync.Mutex
	capacity int
	ttl      time.Duration
	order    *list.List
	items    map[K]*list.Element
}

func NewTTLCache[K comparable, V any](capacity int, ttl time.Duration) *TTLCache[K, V] {
	if capacity <= 0 {
		capacity = 64
	}
	if ttl <= 0 {
		ttl = time.Minute
	}
	return &TTLCache[K, V]{
		capacity: capacity,
		ttl:      ttl,
		order:    list.New(),
		items:    make(map[K]*list.Element, capacity),
	}
}

func (c *TTLCache[K, V]) Set(key K, val V) {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	if elem, ok := c.items[key]; ok {
		c.order.MoveToFront(elem)
		ent := elem.Value.(*entry[K, V])
		ent.value = val
		ent.expiresAt = now.Add(c.ttl)
		return
	}

	if c.order.Len() >= c.capacity {
		oldest := c.order.Back()
		if oldest != nil {
			ent := oldest.Value.(*entry[K, V])
			delete(c.items, ent.key)
			c.order.Remove(oldest)
		}
	}

	elem := c.order.PushFront(&entry[K, V]{
		key:       key,
		value:     val,
		expiresAt: now.Add(c.ttl),
	})
	c.items[key] = elem
}

func (c *TTLCache[K, V]) Get(key K) (V, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	var zero V
	elem, ok := c.items[key]
	if !ok {
		return zero, false
	}
	ent := elem.Value.(*entry[K, V])
	if time.Now().After(ent.expiresAt) {
		delete(c.items, key)
		c.order.Remove(elem)
		return zero, false
	}
	c.order.MoveToFront(elem)
	return ent.value, true
}
` + "```\n\n```go path=cache_test.go\n" + `package cache

import (
	"sync"
	"testing"
	"time"
)

func TestTTLCacheEvictionAndExpiry(t *testing.T) {
	c := NewTTLCache[string, int](2, 40*time.Millisecond)
	c.Set("a", 1)
	c.Set("b", 2)
	_, _ = c.Get("a") // promote "a", making "b" LRU
	c.Set("c", 3)     // evicts "b"

	if _, ok := c.Get("b"); ok {
		t.Fatal("expected LRU key 'b' to be evicted")
	}
	if v, ok := c.Get("a"); !ok || v != 1 {
		t.Fatalf("expected a=1, got %v (ok=%v)", v, ok)
	}

	time.Sleep(50 * time.Millisecond)
	if _, ok := c.Get("a"); ok {
		t.Fatal("expected TTL expiration")
	}
}

func TestTTLCacheConcurrentAccess(t *testing.T) {
	c := NewTTLCache[int, int](32, time.Second)
	var wg sync.WaitGroup
	for i := 0; i < 64; i++ {
		wg.Add(1)
		go func(k int) {
			defer wg.Done()
			c.Set(k%16, k)
			_, _ = c.Get(k % 16)
		}(i)
	}
	wg.Wait()
}
` + "```"
}

func webhookVerifierArtifact() string {
	return "```go path=verifier.go\n" + `package webhookverifier

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"sync"
	"time"
)

var (
	ErrMissingSignature = errors.New("missing signature")
	ErrInvalidSignature = errors.New("invalid signature")
	ErrReplayDetected   = errors.New("replay delivery id detected")
)

type Verifier struct {
	secret []byte
	window time.Duration
	mu     sync.Mutex
	seen   map[string]time.Time
}

func NewVerifier(secret string, replayWindow time.Duration) *Verifier {
	if replayWindow <= 0 {
		replayWindow = 10 * time.Minute
	}
	return &Verifier{
		secret: []byte(secret),
		window: replayWindow,
		seen:   make(map[string]time.Time),
	}
}

func (v *Verifier) Sign(payload []byte) string {
	mac := hmac.New(sha256.New, v.secret)
	mac.Write(payload)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

func (v *Verifier) Verify(deliveryID string, payload []byte, signature string) error {
	if strings.TrimSpace(signature) == "" {
		return ErrMissingSignature
	}
	expected := v.Sign(payload)
	if !hmac.Equal([]byte(signature), []byte(expected)) {
		return ErrInvalidSignature
	}

	if deliveryID != "" {
		v.mu.Lock()
		defer v.mu.Unlock()
		now := time.Now()
		for k, ts := range v.seen {
			if now.Sub(ts) > v.window {
				delete(v.seen, k)
			}
		}
		if _, exists := v.seen[deliveryID]; exists {
			return ErrReplayDetected
		}
		v.seen[deliveryID] = now
	}
	return nil
}
` + "```\n\n```go path=verifier_test.go\n" + `package webhookverifier

import (
	"errors"
	"testing"
	"time"
)

func TestWebhookSignatureAndReplayGuard(t *testing.T) {
	v := NewVerifier("top-secret", time.Minute)
	body := []byte(` + "`" + `{"action":"opened"}` + "`" + `)
	sig := v.Sign(body)

	if err := v.Verify("deliv-1", body, sig); err != nil {
		t.Fatalf("expected valid signature, got %v", err)
	}
	if err := v.Verify("deliv-1", body, sig); !errors.Is(err, ErrReplayDetected) {
		t.Fatalf("expected replay detection, got %v", err)
	}
	if err := v.Verify("deliv-2", body, "sha256=deadbeef"); !errors.Is(err, ErrInvalidSignature) {
		t.Fatalf("expected invalid signature error, got %v", err)
	}
}
` + "```"
}

func dynamicComponentArtifact(title string) string {
	typeName := toExportedIdentifier(title)
	code := fmt.Sprintf(`package generated

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

var (
	ErrEmptyID  = errors.New("record id must not be empty")
	ErrNotFound = errors.New("record not found")
)

// Record represents a versioned entity managed by %s.
type Record struct {
	ID        string
	Payload   map[string]string
	Version   int
	UpdatedAt time.Time
}

// %s implements "%s" with RWMutex concurrency safety and optimistic versioning.
type %s struct {
	mu      sync.RWMutex
	records map[string]Record
}

func New%s() *%s {
	return &%s{
		records: make(map[string]Record),
	}
}

func (s *%s) Upsert(ctx context.Context, id string, payload map[string]string) (Record, error) {
	if err := ctx.Err(); err != nil {
		return Record{}, err
	}
	if id == "" {
		return Record{}, ErrEmptyID
	}

	copied := make(map[string]string, len(payload))
	for k, v := range payload {
		copied[k] = v
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	prev := s.records[id]
	rec := Record{
		ID:        id,
		Payload:   copied,
		Version:   prev.Version + 1,
		UpdatedAt: time.Now().UTC(),
	}
	s.records[id] = rec
	return rec, nil
}

func (s *%s) Get(id string) (Record, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rec, ok := s.records[id]
	if !ok {
		return Record{}, fmt.Errorf("%%w: %%s", ErrNotFound, id)
	}
	return rec, nil
}

func (s *%s) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.records)
}
`, typeName, typeName, title, typeName, typeName, typeName, typeName, typeName, typeName, typeName)

	tests := fmt.Sprintf(`package generated

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
)

func Test%sUpsertAndVersioning(t *testing.T) {
	ctx := context.Background()
	svc := New%s()

	if _, err := svc.Upsert(ctx, "", nil); !errors.Is(err, ErrEmptyID) {
		t.Fatalf("expected ErrEmptyID, got %%v", err)
	}

	r1, err := svc.Upsert(ctx, "item-1", map[string]string{"state": "draft"})
	if err != nil || r1.Version != 1 {
		t.Fatalf("unexpected v1 upsert: %%+v, err=%%v", r1, err)
	}

	r2, err := svc.Upsert(ctx, "item-1", map[string]string{"state": "ready"})
	if err != nil || r2.Version != 2 {
		t.Fatalf("expected version 2, got %%+v, err=%%v", r2, err)
	}

	got, err := svc.Get("item-1")
	if err != nil || got.Payload["state"] != "ready" {
		t.Fatalf("unexpected Get result: %%+v, err=%%v", got, err)
	}
}

func Test%sConcurrentSafety(t *testing.T) {
	ctx := context.Background()
	svc := New%s()
	var wg sync.WaitGroup

	for i := 0; i < 32; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			id := fmt.Sprintf("key-%%d", n%%8)
			_, _ = svc.Upsert(ctx, id, map[string]string{"worker": fmt.Sprint(n)})
			_, _ = svc.Get(id)
		}(i)
	}
	wg.Wait()

	if svc.Count() != 8 {
		t.Fatalf("expected 8 distinct records, got %%d", svc.Count())
	}
}
`, typeName, typeName, typeName, typeName)

	return "```go path=service.go\n" + code + "```\n\n```go path=service_test.go\n" + tests + "```"
}

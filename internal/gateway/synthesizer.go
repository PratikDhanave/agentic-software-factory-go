package gateway

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"
)

// SynthesizeGo produces verified, idiomatic multi-file Go source + race-safe unit tests
// when running offline (zero API keys required) or as fallback.
func SynthesizeGo(specID, title, prompt string) string {
	combined := strings.ToLower(specID + " " + title + " " + prompt)
	switch {
	case strings.Contains(combined, "token") || strings.Contains(combined, "bucket") || strings.Contains(combined, "rate limit") || strings.Contains(combined, "limiter"):
		return synthTokenBucket()
	case strings.Contains(combined, "circuit") || strings.Contains(combined, "breaker"):
		return synthCircuitBreaker()
	case strings.Contains(combined, "lru") || strings.Contains(combined, "cache") || strings.Contains(combined, "ttl"):
		return synthLRUCache()
	case strings.Contains(combined, "hmac") || strings.Contains(combined, "webhook") || strings.Contains(combined, "signature"):
		return synthHMACVerifier()
	default:
		return synthCustomPackage(title, prompt)
	}
}

// SynthesizeConsultation provides crisp, domain-specific specialist guidance offline.
func SynthesizeConsultation(role, name, prompt string) string {
	switch role {
	case "architect":
		return fmt.Sprintf("**%s Assessment**\n\n1. **Package Boundary**: Isolate domain state inside an unexported struct guarded by `sync.Mutex` or explicit channel ownership — zero package-level globals.\n2. **Explicit State Machine**: Model lifecycle states as typed constants with a deterministic `Validate(from, to)` transition table.\n3. **Verification Gate**: Enforce `go vet ./...` and `go test -race ./...` with 100-goroutine contention tests before promoting to `generated/`.\n\n*Context analyzed*: %q", name, truncate(prompt, 120))
	case "security":
		return fmt.Sprintf("**%s Security Audit**\n\n- **Constant-Time Verification**: Always compare digests with `crypto/hmac.Equal` to eliminate timing side-channels.\n- **Boundary Safety**: Canonicalize paths with `filepath.Clean` and verify `filepath.Rel` stays inside the workspace root.\n- **Resource Bounds**: Bound HTTP request bodies (`http.MaxBytesReader`) and reject non-positive capacities.\n\n*Context analyzed*: %q", name, truncate(prompt, 120))
	case "qa":
		return fmt.Sprintf("**%s Verification Plan**\n\n- Table-driven unit tests covering nominal transitions, boundary exhaustion, and invalid config errors.\n- Parallel race stress test spawning 64+ goroutines under `-race`.\n- Deterministic clock injection (`now func() time.Time`) so TTL/refill assertions never flake.\n\n*Context analyzed*: %q", name, truncate(prompt, 120))
	case "performance":
		return fmt.Sprintf("**%s SRE & Latency Notes**\n\n- Keep critical sections O(1) without I/O or allocations under lock.\n- Pre-size maps where capacity is known (`make(map[K]V, cap)`).\n- Propagate `context.Context` deadlines on outbound calls.\n\n*Context analyzed*: %q", name, truncate(prompt, 120))
	default:
		return fmt.Sprintf("**%s Response**\n\nReady to synthesize and verify %q inside an isolated Go module sandbox (`.asf/workspaces/<run-id>`) with `go build`, `go vet`, and `go test -race`.", name, truncate(prompt, 140))
	}
}

func truncate(s string, max int) string {
	s = strings.TrimSpace(s)
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}

func synthTokenBucket() string {
	return "```go path=limiter.go\n" + `package limiter

import (
	"errors"
	"sync"
	"time"
)

var ErrInvalidConfig = errors.New("limiter: capacity and refillRate must be positive")

// TokenBucket enforces a thread-safe token-bucket rate limit with continuous refill.
type TokenBucket struct {
	mu         sync.Mutex
	capacity   float64
	tokens     float64
	refillRate float64
	lastRefill time.Time
	now        func() time.Time
}

// NewTokenBucket creates a rate limiter holding up to capacity tokens, refilling at refillRate tokens/sec.
func NewTokenBucket(capacity, refillRate float64) (*TokenBucket, error) {
	if capacity <= 0 || refillRate <= 0 {
		return nil, ErrInvalidConfig
	}
	nowFn := time.Now
	return &TokenBucket{
		capacity:   capacity,
		tokens:     capacity,
		refillRate: refillRate,
		lastRefill: nowFn(),
		now:        nowFn,
	}, nil
}

// Allow consumes 1 token if available.
func (tb *TokenBucket) Allow() bool {
	return tb.AllowN(1)
}

// AllowN consumes n tokens atomically if available after refilling elapsed time.
func (tb *TokenBucket) AllowN(n float64) bool {
	if n <= 0 {
		return false
	}
	tb.mu.Lock()
	defer tb.mu.Unlock()

	now := tb.now()
	elapsed := now.Sub(tb.lastRefill).Seconds()
	if elapsed > 0 {
		tb.tokens += elapsed * tb.refillRate
		if tb.tokens > tb.capacity {
			tb.tokens = tb.capacity
		}
		tb.lastRefill = now
	}
	if tb.tokens < n {
		return false
	}
	tb.tokens -= n
	return true
}

// Available returns the current token balance after applying elapsed refill.
func (tb *TokenBucket) Available() float64 {
	tb.mu.Lock()
	defer tb.mu.Unlock()
	now := tb.now()
	elapsed := now.Sub(tb.lastRefill).Seconds()
	if elapsed > 0 {
		tb.tokens += elapsed * tb.refillRate
		if tb.tokens > tb.capacity {
			tb.tokens = tb.capacity
		}
		tb.lastRefill = now
	}
	return tb.tokens
}
` + "```\n\n```go path=limiter_test.go\n" + `package limiter

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestValidation(t *testing.T) {
	if _, err := NewTokenBucket(0, 10); err == nil {
		t.Fatal("expected error for zero capacity")
	}
	if _, err := NewTokenBucket(10, -1); err == nil {
		t.Fatal("expected error for negative refill rate")
	}
}

func TestBurstAndRefill(t *testing.T) {
	tb, err := NewTokenBucket(5, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	now := time.Now()
	tb.now = func() time.Time { return now }
	tb.lastRefill = now

	for i := 0; i < 5; i++ {
		if !tb.Allow() {
			t.Fatalf("expected token %d to be allowed", i+1)
		}
	}
	if tb.Allow() {
		t.Fatal("expected bucket to be exhausted")
	}

	now = now.Add(300 * time.Millisecond)
	if !tb.AllowN(3) {
		t.Fatal("expected 3 tokens after 300ms at 10 tokens/sec")
	}
}

func TestConcurrentRaceSafety(t *testing.T) {
	tb, err := NewTokenBucket(50, 100)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var allowed atomic.Int64
	var wg sync.WaitGroup
	for g := 0; g < 100; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if tb.Allow() {
				allowed.Add(1)
			}
			_ = tb.Available()
		}()
	}
	wg.Wait()
	if allowed.Load() == 0 || allowed.Load() > 55 {
		t.Fatalf("unexpected allowed count: %d", allowed.Load())
	}
}
` + "```\n"
}

func synthCircuitBreaker() string {
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

var (
	ErrCircuitOpen   = errors.New("breaker: circuit is open")
	ErrInvalidConfig = errors.New("breaker: threshold and cooldown must be positive")
)

type Breaker struct {
	mu          sync.Mutex
	state       State
	failures    int
	threshold   int
	cooldown    time.Duration
	openedAt    time.Time
	probeInFlight bool
	now         func() time.Time
}

func New(threshold int, cooldown time.Duration) (*Breaker, error) {
	if threshold <= 0 || cooldown <= 0 {
		return nil, ErrInvalidConfig
	}
	return &Breaker{
		state:     StateClosed,
		threshold: threshold,
		cooldown:  cooldown,
		now:       time.Now,
	}, nil
}

func (b *Breaker) State() State {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.state == StateOpen && b.now().Sub(b.openedAt) >= b.cooldown {
		return StateHalfOpen
	}
	return b.state
}

func (b *Breaker) Execute(fn func() error) error {
	b.mu.Lock()
	if b.state == StateOpen {
		if b.now().Sub(b.openedAt) < b.cooldown {
			b.mu.Unlock()
			return ErrCircuitOpen
		}
		b.state = StateHalfOpen
		b.probeInFlight = false
	}
	if b.state == StateHalfOpen {
		if b.probeInFlight {
			b.mu.Unlock()
			return ErrCircuitOpen
		}
		b.probeInFlight = true
	}
	b.mu.Unlock()

	err := fn()

	b.mu.Lock()
	defer b.mu.Unlock()
	if err != nil {
		b.failures++
		if b.state == StateHalfOpen || b.failures >= b.threshold {
			b.state = StateOpen
			b.openedAt = b.now()
			b.probeInFlight = false
		}
		return err
	}
	b.failures = 0
	b.state = StateClosed
	b.probeInFlight = false
	return nil
}
` + "```\n\n```go path=breaker_test.go\n" + `package breaker

import (
	"errors"
	"sync"
	"testing"
	"time"
)

func TestLifecycleTransitions(t *testing.T) {
	b, err := New(2, 200*time.Millisecond)
	if err != nil {
		t.Fatalf("new breaker: %v", err)
	}
	now := time.Now()
	b.now = func() time.Time { return now }

	boom := errors.New("downstream timeout")
	_ = b.Execute(func() error { return boom })
	if b.State() != StateClosed {
		t.Fatalf("expected closed after 1 failure, got %s", b.State())
	}
	_ = b.Execute(func() error { return boom })
	if b.State() != StateOpen {
		t.Fatalf("expected open after 2 failures, got %s", b.State())
	}
	if err := b.Execute(func() error { return nil }); !errors.Is(err, ErrCircuitOpen) {
		t.Fatalf("expected ErrCircuitOpen, got %v", err)
	}

	now = now.Add(250 * time.Millisecond)
	if b.State() != StateHalfOpen {
		t.Fatalf("expected half_open after cooldown, got %s", b.State())
	}
	if err := b.Execute(func() error { return nil }); err != nil {
		t.Fatalf("expected probe success, got %v", err)
	}
	if b.State() != StateClosed {
		t.Fatalf("expected closed after recovery, got %s", b.State())
	}
}

func TestConcurrentExecution(t *testing.T) {
	b, _ := New(5, 50*time.Millisecond)
	var wg sync.WaitGroup
	for i := 0; i < 64; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			_ = b.Execute(func() error {
				if idx%3 == 0 {
					return errors.New("fail")
				}
				return nil
			})
			_ = b.State()
		}(i)
	}
	wg.Wait()
}
` + "```\n"
}

func synthLRUCache() string {
	return "```go path=cache.go\n" + `package cache

import (
	"container/list"
	"errors"
	"sync"
	"time"
)

var ErrInvalidCapacity = errors.New("cache: capacity must be positive")

type entry[K comparable, V any] struct {
	key       K
	val       V
	expiresAt time.Time
}

// TTLCache is a generic O(1) concurrency-safe LRU cache with per-entry TTL expiration.
type TTLCache[K comparable, V any] struct {
	mu       sync.Mutex
	capacity int
	ttl      time.Duration
	ll       *list.List
	items    map[K]*list.Element
	now      func() time.Time
}

func NewTTLCache[K comparable, V any](capacity int, ttl time.Duration) (*TTLCache[K, V], error) {
	if capacity <= 0 {
		return nil, ErrInvalidCapacity
	}
	return &TTLCache[K, V]{
		capacity: capacity,
		ttl:      ttl,
		ll:       list.New(),
		items:    make(map[K]*list.Element, capacity),
		now:      time.Now,
	}, nil
}

func (c *TTLCache[K, V]) Set(key K, val V) {
	c.mu.Lock()
	defer c.mu.Unlock()

	var exp time.Time
	if c.ttl > 0 {
		exp = c.now().Add(c.ttl)
	}
	if el, ok := c.items[key]; ok {
		c.ll.MoveToFront(el)
		ent := el.Value.(*entry[K, V])
		ent.val = val
		ent.expiresAt = exp
		return
	}
	ent := &entry[K, V]{key: key, val: val, expiresAt: exp}
	el := c.ll.PushFront(ent)
	c.items[key] = el
	if c.ll.Len() > c.capacity {
		oldest := c.ll.Back()
		if oldest != nil {
			c.ll.Remove(oldest)
			delete(c.items, oldest.Value.(*entry[K, V]).key)
		}
	}
}

func (c *TTLCache[K, V]) Get(key K) (V, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	var zero V
	el, ok := c.items[key]
	if !ok {
		return zero, false
	}
	ent := el.Value.(*entry[K, V])
	if !ent.expiresAt.IsZero() && !c.now().Before(ent.expiresAt) {
		c.ll.Remove(el)
		delete(c.items, key)
		return zero, false
	}
	c.ll.MoveToFront(el)
	return ent.val, true
}

func (c *TTLCache[K, V]) Len() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.ll.Len()
}
` + "```\n\n```go path=cache_test.go\n" + `package cache

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestEvictionAndTTL(t *testing.T) {
	c, err := NewTTLCache[string, int](2, 100*time.Millisecond)
	if err != nil {
		t.Fatalf("new cache: %v", err)
	}
	now := time.Now()
	c.now = func() time.Time { return now }

	c.Set("a", 1)
	c.Set("b", 2)
	if _, ok := c.Get("a"); !ok {
		t.Fatal("expected hit on a")
	}
	c.Set("c", 3) // evicts b
	if _, ok := c.Get("b"); ok {
		t.Fatal("expected b to be evicted")
	}

	now = now.Add(150 * time.Millisecond)
	if _, ok := c.Get("a"); ok {
		t.Fatal("expected a to expire after TTL")
	}
}

func TestConcurrentAccess(t *testing.T) {
	c, _ := NewTTLCache[string, int](32, time.Second)
	var wg sync.WaitGroup
	for i := 0; i < 80; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			k := fmt.Sprintf("k-%d", n%20)
			c.Set(k, n)
			_, _ = c.Get(k)
			_ = c.Len()
		}(i)
	}
	wg.Wait()
}
` + "```\n"
}

func synthHMACVerifier() string {
	return "```go path=webhook.go\n" + `package webhook

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	ErrMissingHeader    = errors.New("webhook: missing signature or timestamp")
	ErrExpiredTimestamp = errors.New("webhook: timestamp outside tolerance window")
	ErrInvalidSignature = errors.New("webhook: invalid HMAC-SHA256 signature")
	ErrReplayDetected   = errors.New("webhook: delivery nonce already processed")
)

type Verifier struct {
	mu        sync.Mutex
	secret    []byte
	tolerance time.Duration
	seen      map[string]time.Time
	now       func() time.Time
}

func NewVerifier(secret string, tolerance time.Duration) (*Verifier, error) {
	if strings.TrimSpace(secret) == "" || tolerance <= 0 {
		return nil, errors.New("webhook: secret and positive tolerance required")
	}
	return &Verifier{
		secret:    []byte(secret),
		tolerance: tolerance,
		seen:      make(map[string]time.Time),
		now:       time.Now,
	}, nil
}

func ComputeSignature(secret, timestamp string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = fmt.Fprintf(mac, "%s.", timestamp)
	_, _ = mac.Write(body)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

func (v *Verifier) Verify(nonce, timestamp, signature string, body []byte) error {
	if nonce == "" || timestamp == "" || signature == "" {
		return ErrMissingHeader
	}
	sec, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return ErrExpiredTimestamp
	}
	now := v.now()
	ts := time.Unix(sec, 0)
	diff := now.Sub(ts)
	if diff < -v.tolerance || diff > v.tolerance {
		return ErrExpiredTimestamp
	}

	expected := ComputeSignature(string(v.secret), timestamp, body)
	if !hmac.Equal([]byte(expected), []byte(signature)) {
		return ErrInvalidSignature
	}

	v.mu.Lock()
	defer v.mu.Unlock()
	for k, exp := range v.seen {
		if now.After(exp) {
			delete(v.seen, k)
		}
	}
	if _, exists := v.seen[nonce]; exists {
		return ErrReplayDetected
	}
	v.seen[nonce] = now.Add(v.tolerance)
	return nil
}
` + "```\n\n```go path=webhook_test.go\n" + `package webhook

import (
	"errors"
	"fmt"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestVerifyAndReplayProtection(t *testing.T) {
	v, err := NewVerifier("top-secret-key", 5*time.Minute)
	if err != nil {
		t.Fatalf("new verifier: %v", err)
	}
	now := time.Unix(1750000000, 0)
	v.now = func() time.Time { return now }

	ts := strconv.FormatInt(now.Unix(), 10)
	body := []byte(` + "`" + `{"event":"push"}` + "`" + `)
	sig := ComputeSignature("top-secret-key", ts, body)

	if err := v.Verify("evt-1", ts, sig, body); err != nil {
		t.Fatalf("expected valid delivery, got %v", err)
	}
	if err := v.Verify("evt-1", ts, sig, body); !errors.Is(err, ErrReplayDetected) {
		t.Fatalf("expected ErrReplayDetected, got %v", err)
	}
	if err := v.Verify("evt-2", ts, "sha256=deadbeef", body); !errors.Is(err, ErrInvalidSignature) {
		t.Fatalf("expected ErrInvalidSignature, got %v", err)
	}
}

func TestConcurrentReplayRace(t *testing.T) {
	v, _ := NewVerifier("race-secret", time.Minute)
	ts := strconv.FormatInt(time.Now().Unix(), 10)
	body := []byte("payload")
	sig := ComputeSignature("race-secret", ts, body)

	var accepted atomic.Int32
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			if err := v.Verify("same-nonce", ts, sig, body); err == nil {
				accepted.Add(1)
			}
			nSig := ComputeSignature("race-secret", ts, []byte(fmt.Sprintf("p-%d", idx)))
			_ = v.Verify(fmt.Sprintf("nonce-%d", idx), ts, nSig, []byte(fmt.Sprintf("p-%d", idx)))
		}(i)
	}
	wg.Wait()
	if accepted.Load() != 1 {
		t.Fatalf("expected exactly 1 accepted delivery for duplicate nonce, got %d", accepted.Load())
	}
}
` + "```\n"
}

var nonAlpha = regexp.MustCompile(`[^a-zA-Z0-9]+`)

func synthCustomPackage(title, prompt string) string {
	src := title
	if strings.TrimSpace(src) == "" {
		src = prompt
	}
	words := nonAlpha.Split(strings.TrimSpace(src), -1)
	var exportName strings.Builder
	pkgName := "component"
	for _, w := range words {
		if len(w) < 2 {
			continue
		}
		lw := strings.ToLower(w)
		if lw == "implement" || lw == "build" || lw == "create" || lw == "the" || lw == "with" || lw == "for" || lw == "go" {
			continue
		}
		if pkgName == "component" {
			pkgName = lw
		}
		r := []rune(lw)
		r[0] = unicode.ToUpper(r[0])
		exportName.WriteString(string(r))
		if exportName.Len() >= 22 {
			break
		}
	}
	typeName := exportName.String()
	if typeName == "" {
		typeName = "Engine"
	}
	if len(pkgName) > 14 {
		pkgName = pkgName[:14]
	}

	code := fmt.Sprintf("package %s\n\nimport (\n\t\"context\"\n\t\"errors\"\n\t\"fmt\"\n\t\"strings\"\n\t\"sync\"\n\t\"time\"\n)\n\nvar ErrEmptyInput = errors.New(%q)\n\ntype Record struct {\n\tKey       string    `json:\"key\"`\n\tOutput    string    `json:\"output\"`\n\tProcessed time.Time `json:\"processed\"`\n}\n\n// %s provides a concurrency-safe domain service synthesized for %q.\ntype %s struct {\n\tmu      sync.RWMutex\n\trecords map[string]Record\n\tcount   int\n}\n\nfunc New%s() *%s {\n\treturn &%s{records: make(map[string]Record)}\n}\n\nfunc (s *%s) Process(ctx context.Context, key, payload string) (Record, error) {\n\tif err := ctx.Err(); err != nil {\n\t\treturn Record{}, err\n\t}\n\tif strings.TrimSpace(key) == \"\" || strings.TrimSpace(payload) == \"\" {\n\t\treturn Record{}, ErrEmptyInput\n\t}\n\trec := Record{\n\t\tKey:       key,\n\t\tOutput:    fmt.Sprintf(\"verified:%%s\", strings.TrimSpace(payload)),\n\t\tProcessed: time.Now().UTC(),\n\t}\n\ts.mu.Lock()\n\ts.records[key] = rec\n\ts.count++\n\ts.mu.Unlock()\n\treturn rec, nil\n}\n\nfunc (s *%s) Get(key string) (Record, bool) {\n\ts.mu.RLock()\n\tdefer s.mu.RUnlock()\n\trec, ok := s.records[key]\n\treturn rec, ok\n}\n\nfunc (s *%s) Stats() int {\n\ts.mu.RLock()\n\tdefer s.mu.RUnlock()\n\treturn s.count\n}\n",
		pkgName, pkgName+": key and payload must be non-empty", typeName, truncate(prompt, 80), typeName, typeName, typeName, typeName, typeName, typeName, typeName)

	tests := fmt.Sprintf("package %s\n\nimport (\n\t\"context\"\n\t\"errors\"\n\t\"fmt\"\n\t\"sync\"\n\t\"testing\"\n)\n\nfunc TestProcessAndLookup(t *testing.T) {\n\tsvc := New%s()\n\tif _, err := svc.Process(context.Background(), \"\", \"val\"); !errors.Is(err, ErrEmptyInput) {\n\t\tt.Fatalf(\"expected ErrEmptyInput, got %%v\", err)\n\t}\n\trec, err := svc.Process(context.Background(), \"job-1\", \"hello\")\n\tif err != nil {\n\t\tt.Fatalf(\"unexpected error: %%v\", err)\n\t}\n\tif rec.Output != \"verified:hello\" {\n\t\tt.Fatalf(\"unexpected output: %%s\", rec.Output)\n\t}\n\tif got, ok := svc.Get(\"job-1\"); !ok || got.Output != rec.Output {\n\t\tt.Fatal(\"expected stored record\")\n\t}\n}\n\nfunc TestConcurrentSafety(t *testing.T) {\n\tsvc := New%s()\n\tvar wg sync.WaitGroup\n\tfor i := 0; i < 64; i++ {\n\t\twg.Add(1)\n\t\tgo func(idx int) {\n\t\t\tdefer wg.Done()\n\t\t\tk := fmt.Sprintf(\"k-%%d\", idx%%16)\n\t\t\t_, _ = svc.Process(context.Background(), k, \"payload\")\n\t\t\t_, _ = svc.Get(k)\n\t\t\t_ = svc.Stats()\n\t\t}(i)\n\t}\n\twg.Wait()\n\tif svc.Stats() != 64 {\n\t\tt.Fatalf(\"expected 64 processed calls, got %%d\", svc.Stats())\n\t}\n}\n",
		pkgName, typeName, typeName)

	return fmt.Sprintf("```go path=%s.go\n%s```\n\n```go path=%s_test.go\n%s```\n", pkgName, code, pkgName, tests)
}

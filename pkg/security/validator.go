package security

import (
	"fmt"
	"log"
	"net"
	"regexp"
	"strings"
	"sync"
	"time"
)

// Validator provides input validation and sanitization
type Validator struct{}

// ValidateWorkflowID validates a workflow ID format
func (v *Validator) ValidateWorkflowID(id string) error {
	if id == "" {
		return fmt.Errorf("workflow ID cannot be empty")
	}
	if len(id) > 255 {
		return fmt.Errorf("workflow ID too long (max 255 chars)")
	}
	// UUID or alphanumeric with hyphens
	if !regexp.MustCompile(`^[a-zA-Z0-9_-]+$`).MatchString(id) {
		return fmt.Errorf("invalid workflow ID format")
	}
	return nil
}

// ValidateRepositoryName validates a GitHub repository name
func (v *Validator) ValidateRepositoryName(repo string) error {
	if repo == "" {
		return fmt.Errorf("repository name cannot be empty")
	}
	if len(repo) > 255 {
		return fmt.Errorf("repository name too long")
	}
	parts := strings.Split(repo, "/")
	if len(parts) != 2 {
		return fmt.Errorf("invalid repository format (should be owner/repo)")
	}
	for _, part := range parts {
		if !regexp.MustCompile(`^[a-zA-Z0-9_-]+$`).MatchString(part) {
			return fmt.Errorf("invalid repository name format")
		}
	}
	return nil
}

// ValidateCode validates code input (basic checks)
func (v *Validator) ValidateCode(code string) error {
	if code == "" {
		return fmt.Errorf("code cannot be empty")
	}
	if len(code) > 1000000 { // 1MB limit
		return fmt.Errorf("code too large (max 1MB)")
	}
	return nil
}

// SanitizeInput removes potentially dangerous characters
func (v *Validator) SanitizeInput(input string) string {
	// Remove null bytes
	input = strings.ReplaceAll(input, "\x00", "")
	// Trim whitespace
	input = strings.TrimSpace(input)
	return input
}

// ValidateJSONPath validates a JSON path parameter
func (v *Validator) ValidateJSONPath(path string) error {
	if path == "" {
		return fmt.Errorf("path cannot be empty")
	}
	if len(path) > 255 {
		return fmt.Errorf("path too long")
	}
	if strings.Contains(path, "..") {
		return fmt.Errorf("path traversal detected")
	}
	return nil
}

// ValidateQueryLimit validates pagination limits
func (v *Validator) ValidateQueryLimit(limit, maxLimit int) (int, error) {
	if limit < 1 {
		limit = 10
	}
	if limit > maxLimit {
		limit = maxLimit
	}
	return limit, nil
}

// RateLimiter implements basic rate limiting
type RateLimiter struct {
	limits map[string][]*time.Time
	mu     sync.RWMutex
	max    int
	window time.Duration
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(requestsPerSecond int) *RateLimiter {
	rl := &RateLimiter{
		limits: make(map[string][]*time.Time),
		max:    requestsPerSecond,
		window: time.Second,
	}

	// Cleanup old entries every minute
	go func() {
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()

		for range ticker.C {
			rl.cleanup()
		}
	}()

	return rl
}

// Allow checks if a request from the given identifier is allowed
func (rl *RateLimiter) Allow(identifier string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	windowStart := now.Add(-rl.window)

	// Get or create request history for this identifier
	requests := rl.limits[identifier]

	// Remove old requests outside the window
	var validRequests []*time.Time
	for _, reqTime := range requests {
		if reqTime.After(windowStart) {
			validRequests = append(validRequests, reqTime)
		}
	}

	// Check if limit exceeded
	if len(validRequests) >= rl.max {
		log.Printf("[RateLimit] Limit exceeded for: %s (%d/%d)", identifier, len(validRequests), rl.max)
		rl.limits[identifier] = validRequests
		return false
	}

	// Add new request
	validRequests = append(validRequests, &now)
	rl.limits[identifier] = validRequests

	return true
}

// cleanup removes expired entries
func (rl *RateLimiter) cleanup() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	windowStart := now.Add(-rl.window * 2) // Keep 2 windows of history

	for id, requests := range rl.limits {
		var validRequests []*time.Time
		for _, reqTime := range requests {
			if reqTime.After(windowStart) {
				validRequests = append(validRequests, reqTime)
			}
		}

		if len(validRequests) == 0 {
			delete(rl.limits, id)
		} else {
			rl.limits[id] = validRequests
		}
	}
}

// IPWhitelist manages IP whitelist
type IPWhitelist struct {
	ips map[string]bool
	mu  sync.RWMutex
}

// NewIPWhitelist creates a new IP whitelist
func NewIPWhitelist() *IPWhitelist {
	return &IPWhitelist{
		ips: make(map[string]bool),
	}
}

// Add adds an IP to the whitelist
func (w *IPWhitelist) Add(ip string) error {
	if !isValidIP(ip) {
		return fmt.Errorf("invalid IP address: %s", ip)
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	w.ips[ip] = true
	log.Printf("[IPWhitelist] Added: %s", ip)
	return nil
}

// IsAllowed checks if an IP is whitelisted
func (w *IPWhitelist) IsAllowed(ip string) bool {
	w.mu.RLock()
	defer w.mu.RUnlock()

	return w.ips[ip]
}

// Remove removes an IP from the whitelist
func (w *IPWhitelist) Remove(ip string) {
	w.mu.Lock()
	defer w.mu.Unlock()

	delete(w.ips, ip)
	log.Printf("[IPWhitelist] Removed: %s", ip)
}

// isValidIP checks if a string is a valid IP address
func isValidIP(ip string) bool {
	return net.ParseIP(ip) != nil
}

// CORS configuration
type CORS struct {
	AllowedOrigins   []string
	AllowedMethods   []string
	AllowedHeaders   []string
	ExposedHeaders   []string
	AllowCredentials bool
	MaxAge           int
}

// DefaultCORS returns default CORS configuration
func DefaultCORS() *CORS {
	return &CORS{
		AllowedOrigins: []string{"http://localhost:3000", "http://localhost:8080"},
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"Content-Type", "Authorization"},
		ExposedHeaders: []string{"Content-Length"},
		MaxAge:         3600,
	}
}

// IsOriginAllowed checks if origin is allowed
func (c *CORS) IsOriginAllowed(origin string) bool {
	for _, allowed := range c.AllowedOrigins {
		if allowed == "*" || allowed == origin {
			return true
		}
	}
	return false
}

// AuditLog logs security-relevant events
type AuditLog struct {
	events []*AuditEvent
	mu     sync.RWMutex
	max    int
}

// AuditEvent represents a security event
type AuditEvent struct {
	Timestamp time.Time
	EventType string
	Actor     string
	Resource  string
	Action    string
	Result    string
	Details   string
}

// NewAuditLog creates a new audit log
func NewAuditLog(maxEvents int) *AuditLog {
	return &AuditLog{
		events: make([]*AuditEvent, 0, maxEvents),
		max:    maxEvents,
	}
}

// Log records an audit event
func (al *AuditLog) Log(eventType, actor, resource, action, result, details string) {
	al.mu.Lock()
	defer al.mu.Unlock()

	event := &AuditEvent{
		Timestamp: time.Now(),
		EventType: eventType,
		Actor:     actor,
		Resource:  resource,
		Action:    action,
		Result:    result,
		Details:   details,
	}

	al.events = append(al.events, event)

	// Keep only max events
	if len(al.events) > al.max {
		al.events = al.events[len(al.events)-al.max:]
	}

	log.Printf("[Audit] %s: %s %s %s (%s) - %s", eventType, actor, action, resource, result, details)
}

// GetEvents returns recent audit events
func (al *AuditLog) GetEvents(limit int) []*AuditEvent {
	al.mu.RLock()
	defer al.mu.RUnlock()

	if limit > len(al.events) {
		limit = len(al.events)
	}

	start := len(al.events) - limit
	if start < 0 {
		start = 0
	}

	return al.events[start:]
}

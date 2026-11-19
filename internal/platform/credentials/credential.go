package credentials

import (
	"context"
	"fmt"
	"time"

	"github.com/PratikDhanave/agentic-software-factory-go/internal/platform/identity"
	"github.com/google/uuid"
)

// Credential represents a capability grant for an agent.
type Credential struct {
	ID        string            `json:"id"`
	AgentID   string            `json:"agent_id"`
	Subject   string            `json:"subject"` // e.g., "code:analyze", "tool:invoke"
	Actions   []string          `json:"actions"` // e.g., ["read", "write", "execute"]
	IssuedAt  time.Time         `json:"issued_at"`
	ExpiresAt time.Time         `json:"expires_at"`
	Signature string            `json:"signature"`
	Metadata  map[string]string `json:"metadata,omitempty"`
	Revoked   bool              `json:"revoked,omitempty"`
	RevokedAt *time.Time        `json:"revoked_at,omitempty"`
}

// CredentialRequest is used to request a new credential.
type CredentialRequest struct {
	AgentID  string
	Subject  string
	Actions  []string
	Duration time.Duration
	Metadata map[string]string
}

// Broker manages credential lifecycle.
type Broker struct {
	identities   *identity.System
	credentials  map[string]*Credential
	signingKey   map[string]string // agent -> private key mapping
}

// NewBroker creates a new credential broker.
func NewBroker(identities *identity.System) *Broker {
	return &Broker{
		identities:  identities,
		credentials: make(map[string]*Credential),
		signingKey:  make(map[string]string),
	}
}

// SetSigningKey registers a private key for signing credentials.
func (b *Broker) SetSigningKey(agentID string, privateKeyPem string) {
	b.signingKey[agentID] = privateKeyPem
}

// Issue creates and signs a new credential.
func (b *Broker) Issue(ctx context.Context, req *CredentialRequest) (*Credential, error) {
	if req.AgentID == "" {
		return nil, fmt.Errorf("agent ID required")
	}
	if req.Subject == "" {
		return nil, fmt.Errorf("subject required")
	}
	if len(req.Actions) == 0 {
		return nil, fmt.Errorf("actions required")
	}
	if req.Duration == 0 {
		req.Duration = 24 * time.Hour // default to 24 hours
	}

	// Verify agent exists
	_, err := b.identities.Get(ctx, req.AgentID)
	if err != nil {
		return nil, fmt.Errorf("agent not found: %w", err)
	}

	now := time.Now().UTC()
	credentialID := uuid.New().String()

	cred := &Credential{
		ID:        credentialID,
		AgentID:   req.AgentID,
		Subject:   req.Subject,
		Actions:   req.Actions,
		IssuedAt:  now,
		ExpiresAt: now.Add(req.Duration),
		Metadata:  req.Metadata,
	}

	// Sign the credential
	credentialData := fmt.Sprintf("%s:%s:%s:%d",
		cred.ID, cred.AgentID, cred.Subject, cred.ExpiresAt.Unix())

	signingKey, err := b.identities.GetSigningKey(ctx, req.AgentID)
	if err != nil {
		return nil, fmt.Errorf("get signing key: %w", err)
	}

	sig, err := identity.Sign(signingKey.PrivateKey, []byte(credentialData))
	if err != nil {
		return nil, fmt.Errorf("sign credential: %w", err)
	}

	cred.Signature = identity.EncodeSignature(sig)

	b.credentials[credentialID] = cred
	return cred, nil
}

// Verify checks if a credential is valid (signature, expiry, not revoked).
func (b *Broker) Verify(ctx context.Context, cred *Credential) (bool, string, error) {
	if cred == nil {
		return false, "credential is nil", nil
	}

	// Check if revoked
	if cred.Revoked {
		return false, "credential revoked", nil
	}

	// Check expiry
	if time.Now().UTC().After(cred.ExpiresAt) {
		return false, "credential expired", nil
	}

	// Verify agent exists
	_, err := b.identities.Get(ctx, cred.AgentID)
	if err != nil {
		return false, fmt.Sprintf("agent not found: %v", err), nil
	}

	// Verify signature
	credentialData := fmt.Sprintf("%s:%s:%s:%d",
		cred.ID, cred.AgentID, cred.Subject, cred.ExpiresAt.Unix())

	sig, err := identity.DecodeSignature(cred.Signature)
	if err != nil {
		return false, fmt.Sprintf("decode signature: %v", err), nil
	}

	valid, err := b.identities.Verify(ctx, cred.AgentID, []byte(credentialData), sig)
	if err != nil {
		return false, fmt.Sprintf("verify signature: %v", err), nil
	}

	if !valid {
		return false, "signature invalid", nil
	}

	return true, "", nil
}

// Get retrieves a credential by ID.
func (b *Broker) Get(ctx context.Context, credentialID string) (*Credential, error) {
	cred, ok := b.credentials[credentialID]
	if !ok {
		return nil, fmt.Errorf("credential not found: %s", credentialID)
	}
	return cred, nil
}

// Revoke immediately invalidates a credential.
func (b *Broker) Revoke(ctx context.Context, credentialID string) error {
	cred, err := b.Get(ctx, credentialID)
	if err != nil {
		return fmt.Errorf("get credential: %w", err)
	}

	now := time.Now().UTC()
	cred.Revoked = true
	cred.RevokedAt = &now

	return nil
}

// Refresh extends the expiry of a credential.
func (b *Broker) Refresh(ctx context.Context, credentialID string, duration time.Duration) (*Credential, error) {
	cred, err := b.Get(ctx, credentialID)
	if err != nil {
		return nil, fmt.Errorf("get credential: %w", err)
	}

	if cred.Revoked {
		return nil, fmt.Errorf("cannot refresh revoked credential")
	}

	// Create new credential with extended expiry
	now := time.Now().UTC()
	newCred := &Credential{
		ID:        uuid.New().String(),
		AgentID:   cred.AgentID,
		Subject:   cred.Subject,
		Actions:   cred.Actions,
		IssuedAt:  now,
		ExpiresAt: now.Add(duration),
		Metadata:  cred.Metadata,
	}

	// Sign the new credential
	credentialData := fmt.Sprintf("%s:%s:%s:%d",
		newCred.ID, newCred.AgentID, newCred.Subject, newCred.ExpiresAt.Unix())

	signingKey, err := b.identities.GetSigningKey(ctx, cred.AgentID)
	if err != nil {
		return nil, fmt.Errorf("get signing key: %w", err)
	}

	sig, err := identity.Sign(signingKey.PrivateKey, []byte(credentialData))
	if err != nil {
		return nil, fmt.Errorf("sign credential: %w", err)
	}

	newCred.Signature = identity.EncodeSignature(sig)
	b.credentials[newCred.ID] = newCred

	// Revoke the old credential
	_ = b.Revoke(ctx, credentialID)

	return newCred, nil
}

// ListForAgent returns all valid (non-revoked, non-expired) credentials for an agent.
func (b *Broker) ListForAgent(ctx context.Context, agentID string) ([]*Credential, error) {
	var credentials []*Credential
	now := time.Now().UTC()

	for _, cred := range b.credentials {
		if cred.AgentID == agentID && !cred.Revoked && now.Before(cred.ExpiresAt) {
			credentials = append(credentials, cred)
		}
	}

	return credentials, nil
}

// HasCapability checks if an agent has a specific capability.
func (b *Broker) HasCapability(ctx context.Context, agentID string, subject string, action string) (bool, error) {
	credentials, err := b.ListForAgent(ctx, agentID)
	if err != nil {
		return false, fmt.Errorf("list credentials: %w", err)
	}

	for _, cred := range credentials {
		// Check subject match (exact or wildcard)
		if cred.Subject != subject && cred.Subject != "*" {
			continue
		}

		// Check action match
		for _, credAction := range cred.Actions {
			if credAction == action || credAction == "*" {
				return true, nil
			}
		}
	}

	return false, nil
}

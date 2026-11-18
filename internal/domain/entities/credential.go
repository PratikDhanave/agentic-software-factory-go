package entities

import (
	"time"

	"github.com/google/uuid"
)

type CredentialType string

const (
	CredentialTypeAPIKey      CredentialType = "api_key"
	CredentialTypeOAuth       CredentialType = "oauth"
	CredentialTypeBasicAuth   CredentialType = "basic_auth"
	CredentialTypePrivateKey  CredentialType = "private_key"
	CredentialTypeToken       CredentialType = "token"
	CredentialTypeAwsKey      CredentialType = "aws_key"
)

type Credential struct {
	ID          string
	Owner       string
	OwnerType   PrincipalType
	Type        CredentialType
	Name        string
	Secret      string
	Scopes      []string
	Metadata    map[string]interface{}
	Active      bool
	ExpiresAt   *time.Time
	LastUsedAt  *time.Time
	LastRotatedAt *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type CredentialRequest struct {
	ID            string
	CredentialID  string
	PrincipalID   string
	PrincipalType PrincipalType
	Purpose       string
	Duration      time.Duration
	Scopes        []string
	RequestedAt   time.Time
	ExpiresAt     time.Time
	CorrelationID string
}

type InjectedCredential struct {
	Value       string
	Type        CredentialType
	ExpiresAt   time.Time
	Scopes      []string
	RequestID   string
}

type CredentialRotationPolicy struct {
	CredentialID      string
	RotationInterval  time.Duration
	AutoRotateEnabled bool
	LastRotatedAt     *time.Time
	NextRotationAt    time.Time
}

func NewCredential(owner string, ownerType PrincipalType, credType CredentialType, name string) *Credential {
	return &Credential{
		ID:        uuid.New().String(),
		Owner:     owner,
		OwnerType: ownerType,
		Type:      credType,
		Name:      name,
		Secret:    "",
		Scopes:    []string{},
		Metadata:  make(map[string]interface{}),
		Active:    true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

func (c *Credential) SetSecret(secret string) {
	c.Secret = secret
	c.UpdatedAt = time.Now()
}

func (c *Credential) AddScope(scope string) *Credential {
	for _, s := range c.Scopes {
		if s == scope {
			return c
		}
	}
	c.Scopes = append(c.Scopes, scope)
	c.UpdatedAt = time.Now()
	return c
}

func (c *Credential) SetExpiry(expiry time.Time) *Credential {
	c.ExpiresAt = &expiry
	c.UpdatedAt = time.Now()
	return c
}

func (c *Credential) SetMetadata(key string, value interface{}) *Credential {
	c.Metadata[key] = value
	c.UpdatedAt = time.Now()
	return c
}

func (c *Credential) IsExpired() bool {
	if c.ExpiresAt == nil {
		return false
	}
	return time.Now().After(*c.ExpiresAt)
}

func (c *Credential) UpdateLastUsed() {
	now := time.Now()
	c.LastUsedAt = &now
}

func (c *Credential) CanAccess(scope string) bool {
	if len(c.Scopes) == 0 {
		return true
	}

	for _, s := range c.Scopes {
		if s == "*" || s == scope {
			return true
		}
	}

	return false
}

func NewCredentialRequest(credentialID, principalID string, principalType PrincipalType, purpose string, duration time.Duration) *CredentialRequest {
	return &CredentialRequest{
		ID:            uuid.New().String(),
		CredentialID:  credentialID,
		PrincipalID:   principalID,
		PrincipalType: principalType,
		Purpose:       purpose,
		Duration:      duration,
		Scopes:        []string{},
		RequestedAt:   time.Now(),
		ExpiresAt:     time.Now().Add(duration),
		CorrelationID: uuid.New().String(),
	}
}

func (cr *CredentialRequest) AddScope(scope string) *CredentialRequest {
	cr.Scopes = append(cr.Scopes, scope)
	return cr
}

func (cr *CredentialRequest) IsExpired() bool {
	return time.Now().After(cr.ExpiresAt)
}

func NewInjectedCredential(value string, credType CredentialType, expiresAt time.Time, requestID string) *InjectedCredential {
	return &InjectedCredential{
		Value:     value,
		Type:      credType,
		ExpiresAt: expiresAt,
		Scopes:    []string{},
		RequestID: requestID,
	}
}

func (ic *InjectedCredential) AddScope(scope string) *InjectedCredential {
	ic.Scopes = append(ic.Scopes, scope)
	return ic
}

func (ic *InjectedCredential) IsExpired() bool {
	return time.Now().After(ic.ExpiresAt)
}

func NewCredentialRotationPolicy(credentialID string, interval time.Duration) *CredentialRotationPolicy {
	return &CredentialRotationPolicy{
		CredentialID:      credentialID,
		RotationInterval:  interval,
		AutoRotateEnabled: false,
		NextRotationAt:    time.Now().Add(interval),
	}
}

func (crp *CredentialRotationPolicy) ShouldRotate() bool {
	if !crp.AutoRotateEnabled {
		return false
	}
	return time.Now().After(crp.NextRotationAt)
}

func (crp *CredentialRotationPolicy) MarkRotated() {
	now := time.Now()
	crp.LastRotatedAt = &now
	crp.NextRotationAt = now.Add(crp.RotationInterval)
}

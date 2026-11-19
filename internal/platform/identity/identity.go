package identity

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// AgentIdentity represents a registered agent in the system.
type AgentIdentity struct {
	ID        string    `json:"id"`
	Type      string    `json:"type"`
	PublicKey string    `json:"public_key"`
	CreatedAt time.Time `json:"created_at"`
	CreatedBy string    `json:"created_by"`
	Metadata  map[string]string `json:"metadata,omitempty"`
}

// SigningKey contains private/public key pair for an agent.
type SigningKey struct {
	AgentID    string `json:"agent_id"`
	PublicKey  string `json:"public_key"`
	PrivateKey string `json:"private_key"` // PEM-encoded, store securely
}

// RegistrationRequest is used to register a new agent.
type RegistrationRequest struct {
	Type      string
	CreatedBy string
	Metadata  map[string]string
}

// System manages agent identities and signing keys.
type System struct {
	identities map[string]*AgentIdentity
	keys       map[string]*SigningKey
}

// New creates a new identity system.
func New() *System {
	return &System{
		identities: make(map[string]*AgentIdentity),
		keys:       make(map[string]*SigningKey),
	}
}

// Register registers a new agent and generates signing keys.
func (s *System) Register(ctx context.Context, req *RegistrationRequest) (*AgentIdentity, *SigningKey, error) {
	if req.Type == "" {
		return nil, nil, fmt.Errorf("agent type required")
	}

	agentID := uuid.New().String()

	// Generate RSA key pair
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, fmt.Errorf("generate key: %w", err)
	}

	publicKeyBytes, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	if err != nil {
		return nil, nil, fmt.Errorf("marshal public key: %w", err)
	}

	publicKeyPem := pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: publicKeyBytes,
	})

	privateKeyBytes := x509.MarshalPKCS1PrivateKey(privateKey)
	privateKeyPem := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: privateKeyBytes,
	})

	now := time.Now().UTC()

	identity := &AgentIdentity{
		ID:        agentID,
		Type:      req.Type,
		PublicKey: string(publicKeyPem),
		CreatedAt: now,
		CreatedBy: req.CreatedBy,
		Metadata:  req.Metadata,
	}

	signingKey := &SigningKey{
		AgentID:    agentID,
		PublicKey:  string(publicKeyPem),
		PrivateKey: string(privateKeyPem),
	}

	s.identities[agentID] = identity
	s.keys[agentID] = signingKey

	return identity, signingKey, nil
}

// Get retrieves an agent identity by ID.
func (s *System) Get(ctx context.Context, agentID string) (*AgentIdentity, error) {
	identity, ok := s.identities[agentID]
	if !ok {
		return nil, fmt.Errorf("agent not found: %s", agentID)
	}
	return identity, nil
}

// GetSigningKey retrieves the signing key for an agent.
func (s *System) GetSigningKey(ctx context.Context, agentID string) (*SigningKey, error) {
	key, ok := s.keys[agentID]
	if !ok {
		return nil, fmt.Errorf("signing key not found: %s", agentID)
	}
	return key, nil
}

// Verify verifies a signature using the agent's public key.
func (s *System) Verify(ctx context.Context, agentID string, data []byte, signature []byte) (bool, error) {
	identity, err := s.Get(ctx, agentID)
	if err != nil {
		return false, fmt.Errorf("get identity: %w", err)
	}

	// Decode PEM public key
	block, _ := pem.Decode([]byte(identity.PublicKey))
	if block == nil {
		return false, fmt.Errorf("failed to parse public key")
	}

	pubKey, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return false, fmt.Errorf("parse public key: %w", err)
	}

	rsaPublicKey, ok := pubKey.(*rsa.PublicKey)
	if !ok {
		return false, fmt.Errorf("not an RSA public key")
	}

	// Verify signature
	hash := sha256.Sum256(data)
	err = rsa.VerifyPKCS1v15(rsaPublicKey, 0, hash[:], signature)
	if err != nil {
		return false, nil
	}

	return true, nil
}

// Sign signs data with the agent's private key.
func Sign(privateKeyPem string, data []byte) ([]byte, error) {
	block, _ := pem.Decode([]byte(privateKeyPem))
	if block == nil {
		return nil, fmt.Errorf("failed to parse private key")
	}

	privKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse private key: %w", err)
	}

	hash := sha256.Sum256(data)
	signature, err := rsa.SignPKCS1v15(rand.Reader, privKey, 0, hash[:])
	if err != nil {
		return nil, fmt.Errorf("sign data: %w", err)
	}

	return signature, nil
}

// EncodeSignature encodes a signature to base64 for transport.
func EncodeSignature(signature []byte) string {
	return base64.StdEncoding.EncodeToString(signature)
}

// DecodeSignature decodes a base64-encoded signature.
func DecodeSignature(encoded string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(encoded)
}

// List returns all registered agent identities.
func (s *System) List(ctx context.Context) ([]*AgentIdentity, error) {
	identities := make([]*AgentIdentity, 0, len(s.identities))
	for _, identity := range s.identities {
		identities = append(identities, identity)
	}
	return identities, nil
}

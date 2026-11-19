package identity

import (
	"context"
	"testing"
	"time"
)

func TestRegisterAgent(t *testing.T) {
	sys := New()
	ctx := context.Background()

	identity, sigKey, err := sys.Register(ctx, &RegistrationRequest{
		Type:      "code-reviewer",
		CreatedBy: "system",
		Metadata: map[string]string{
			"version": "1.0",
		},
	})

	if err != nil {
		t.Fatalf("register failed: %v", err)
	}

	if identity.ID == "" {
		t.Error("agent ID is empty")
	}

	if identity.Type != "code-reviewer" {
		t.Errorf("got type %s, want code-reviewer", identity.Type)
	}

	if identity.CreatedBy != "system" {
		t.Errorf("got createdBy %s, want system", identity.CreatedBy)
	}

	if sigKey.AgentID != identity.ID {
		t.Errorf("signing key agent ID mismatch")
	}

	if sigKey.PublicKey == "" {
		t.Error("public key is empty")
	}

	if sigKey.PrivateKey == "" {
		t.Error("private key is empty")
	}
}

func TestGetIdentity(t *testing.T) {
	sys := New()
	ctx := context.Background()

	registered, _, _ := sys.Register(ctx, &RegistrationRequest{
		Type:      "bug-finder",
		CreatedBy: "user",
	})

	retrieved, err := sys.Get(ctx, registered.ID)
	if err != nil {
		t.Fatalf("get failed: %v", err)
	}

	if retrieved.ID != registered.ID {
		t.Errorf("ID mismatch: got %s, want %s", retrieved.ID, registered.ID)
	}
}

func TestGetNonexistentIdentity(t *testing.T) {
	sys := New()
	ctx := context.Background()

	_, err := sys.Get(ctx, "nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent agent")
	}
}

func TestSignAndVerify(t *testing.T) {
	sys := New()
	ctx := context.Background()

	identity, sigKey, _ := sys.Register(ctx, &RegistrationRequest{
		Type:      "test",
		CreatedBy: "test",
	})

	data := []byte("test data")
	signature, err := Sign(sigKey.PrivateKey, data)
	if err != nil {
		t.Fatalf("sign failed: %v", err)
	}

	valid, err := sys.Verify(ctx, identity.ID, data, signature)
	if err != nil {
		t.Fatalf("verify failed: %v", err)
	}

	if !valid {
		t.Error("signature verification failed")
	}
}

func TestVerifyInvalidSignature(t *testing.T) {
	sys := New()
	ctx := context.Background()

	identity, _, _ := sys.Register(ctx, &RegistrationRequest{
		Type:      "test",
		CreatedBy: "test",
	})

	data := []byte("test data")
	invalidSignature := []byte("invalid signature")

	valid, err := sys.Verify(ctx, identity.ID, data, invalidSignature)
	if err != nil {
		t.Fatalf("verify failed: %v", err)
	}

	if valid {
		t.Error("expected verification to fail with invalid signature")
	}
}

func TestListIdentities(t *testing.T) {
	sys := New()
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		_, _, _ = sys.Register(ctx, &RegistrationRequest{
			Type:      "test",
			CreatedBy: "test",
		})
	}

	identities, err := sys.List(ctx)
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}

	if len(identities) != 3 {
		t.Errorf("got %d identities, want 3", len(identities))
	}
}

func TestEncodeDecodeSignature(t *testing.T) {
	sys := New()
	ctx := context.Background()

	_, sigKey, _ := sys.Register(ctx, &RegistrationRequest{
		Type:      "test",
		CreatedBy: "test",
	})

	data := []byte("test data")
	sig, _ := Sign(sigKey.PrivateKey, data)

	encoded := EncodeSignature(sig)
	decoded, err := DecodeSignature(encoded)
	if err != nil {
		t.Fatalf("decode failed: %v", err)
	}

	if string(decoded) != string(sig) {
		t.Error("encoded/decoded signature mismatch")
	}
}

func TestAgentCreatedAtTimestamp(t *testing.T) {
	sys := New()
	ctx := context.Background()

	before := time.Now().UTC()
	identity, _, _ := sys.Register(ctx, &RegistrationRequest{
		Type:      "test",
		CreatedBy: "test",
	})
	after := time.Now().UTC()

	if identity.CreatedAt.Before(before) || identity.CreatedAt.After(after) {
		t.Error("created_at timestamp is out of range")
	}
}

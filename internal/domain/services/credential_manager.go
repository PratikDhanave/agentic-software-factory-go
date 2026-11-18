package services

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"time"

	"github.com/PratikDhanave/agentic-software-factory-go/internal/domain/entities"
	"github.com/PratikDhanave/agentic-software-factory-go/internal/domain/repositories"
)

type CredentialManager struct {
	credRepo repositories.CredentialRepository
	auditLog repositories.AuditRepository
	encryptionKey []byte
	logger   Logger
}

func NewCredentialManager(
	credRepo repositories.CredentialRepository,
	auditLog repositories.AuditRepository,
	encryptionKey []byte,
	logger Logger,
) *CredentialManager {
	return &CredentialManager{
		credRepo:      credRepo,
		auditLog:      auditLog,
		encryptionKey: encryptionKey,
		logger:        logger,
	}
}

func (cm *CredentialManager) StoreCredential(ctx *entities.SecurityContext, cred *entities.Credential) error {
	if !ctx.CanPerform(entities.ResourceCredential, entities.ActionCreate) {
		cm.auditAccess(ctx, entities.ActionCreate, entities.ResourceCredential, cred.ID, false, "unauthorized")
		return fmt.Errorf("unauthorized to store credentials")
	}

	encrypted, err := cm.encryptSecret(cred.Secret)
	if err != nil {
		return err
	}

	cred.Secret = encrypted

	err = cm.credRepo.SaveCredential(cred)
	if err != nil {
		cm.auditAccess(ctx, entities.ActionCreate, entities.ResourceCredential, cred.ID, false, err.Error())
		return err
	}

	cm.auditAccess(ctx, entities.ActionCreate, entities.ResourceCredential, cred.ID, true, "")
	cm.logger.Info("credential stored",
		"credential_id", cred.ID,
		"owner", cred.Owner,
		"type", cred.Type,
	)

	return nil
}

func (cm *CredentialManager) RequestCredential(ctx *entities.SecurityContext, req *entities.CredentialRequest) (*entities.InjectedCredential, error) {
	if !ctx.CanPerform(entities.ResourceCredential, entities.ActionRead) {
		cm.auditAccess(ctx, entities.ActionRead, entities.ResourceCredential, req.CredentialID, false, "unauthorized")
		return nil, fmt.Errorf("unauthorized to access credentials")
	}

	cred, err := cm.credRepo.GetCredential(req.CredentialID)
	if err != nil {
		cm.auditAccess(ctx, entities.ActionRead, entities.ResourceCredential, req.CredentialID, false, "not found")
		return nil, err
	}

	if !cred.Active {
		cm.auditAccess(ctx, entities.ActionRead, entities.ResourceCredential, cred.ID, false, "credential inactive")
		return nil, fmt.Errorf("credential is inactive")
	}

	if cred.IsExpired() {
		cm.auditAccess(ctx, entities.ActionRead, entities.ResourceCredential, cred.ID, false, "credential expired")
		return nil, fmt.Errorf("credential has expired")
	}

	for _, reqScope := range req.Scopes {
		if !cred.CanAccess(reqScope) {
			cm.auditAccess(ctx, entities.ActionRead, entities.ResourceCredential, cred.ID, false, fmt.Sprintf("scope denied: %s", reqScope))
			return nil, fmt.Errorf("credential scope denied: %s", reqScope)
		}
	}

	decrypted, err := cm.decryptSecret(cred.Secret)
	if err != nil {
		cm.auditAccess(ctx, entities.ActionRead, entities.ResourceCredential, cred.ID, false, "decryption failed")
		return nil, err
	}

	cred.UpdateLastUsed()
	cm.credRepo.UpdateCredential(cred)

	injected := entities.NewInjectedCredential(decrypted, cred.Type, req.ExpiresAt, req.ID)
	for _, scope := range req.Scopes {
		injected.AddScope(scope)
	}

	cm.auditAccess(ctx, entities.ActionRead, entities.ResourceCredential, cred.ID, true, fmt.Sprintf("purpose: %s", req.Purpose))
	cm.logger.Info("credential injected",
		"credential_id", cred.ID,
		"request_id", req.ID,
		"principal", req.PrincipalID,
		"purpose", req.Purpose,
	)

	return injected, nil
}

func (cm *CredentialManager) RotateCredential(ctx *entities.SecurityContext, credentialID string, newSecret string) error {
	if !ctx.CanPerform(entities.ResourceCredential, entities.ActionUpdate) {
		cm.auditAccess(ctx, entities.ActionUpdate, entities.ResourceCredential, credentialID, false, "unauthorized")
		return fmt.Errorf("unauthorized to rotate credentials")
	}

	cred, err := cm.credRepo.GetCredential(credentialID)
	if err != nil {
		return err
	}

	encrypted, err := cm.encryptSecret(newSecret)
	if err != nil {
		return err
	}

	oldSecret := cred.Secret
	cred.Secret = encrypted
	now := time.Now()
	cred.LastRotatedAt = &now

	err = cm.credRepo.UpdateCredential(cred)
	if err != nil {
		cm.auditAccess(ctx, entities.ActionUpdate, entities.ResourceCredential, credentialID, false, err.Error())
		return err
	}

	cm.auditAccess(ctx, entities.ActionUpdate, entities.ResourceCredential, credentialID, true, "credential rotated")
	cm.logger.Info("credential rotated",
		"credential_id", credentialID,
		"old_secret", oldSecret[:min(10, len(oldSecret))]+"...",
		"rotated_by", ctx.PrincipalID,
	)

	return nil
}

func (cm *CredentialManager) RevokeCredential(ctx *entities.SecurityContext, credentialID string) error {
	if !ctx.CanPerform(entities.ResourceCredential, entities.ActionDelete) {
		cm.auditAccess(ctx, entities.ActionDelete, entities.ResourceCredential, credentialID, false, "unauthorized")
		return fmt.Errorf("unauthorized to revoke credentials")
	}

	cred, err := cm.credRepo.GetCredential(credentialID)
	if err != nil {
		return err
	}

	cred.Active = false
	cred.UpdatedAt = time.Now()

	err = cm.credRepo.UpdateCredential(cred)
	if err != nil {
		cm.auditAccess(ctx, entities.ActionDelete, entities.ResourceCredential, credentialID, false, err.Error())
		return err
	}

	cm.auditAccess(ctx, entities.ActionDelete, entities.ResourceCredential, credentialID, true, "credential revoked")
	cm.logger.Info("credential revoked",
		"credential_id", credentialID,
		"revoked_by", ctx.PrincipalID,
	)

	return nil
}

func (cm *CredentialManager) encryptSecret(secret string) (string, error) {
	if len(cm.encryptionKey) == 0 {
		return "", fmt.Errorf("encryption key not configured")
	}

	block, err := aes.NewCipher(cm.encryptionKey)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	_, err = io.ReadFull(rand.Reader, nonce)
	if err != nil {
		return "", err
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(secret), nil)
	return hex.EncodeToString(ciphertext), nil
}

func (cm *CredentialManager) decryptSecret(encrypted string) (string, error) {
	if len(cm.encryptionKey) == 0 {
		return "", fmt.Errorf("encryption key not configured")
	}

	ciphertext, err := hex.DecodeString(encrypted)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(cm.encryptionKey)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]

	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}

func (cm *CredentialManager) auditAccess(ctx *entities.SecurityContext, action entities.Action, resource entities.Resource, resourceID string, success bool, details string) {
	entry := entities.NewAuditEntry(ctx, action, resource, resourceID)

	if success {
		entry.SetStatus(entities.AuditSuccess)
	} else {
		entry.SetStatus(entities.AuditDenied)
		entry.SetError(details)
	}

	entry.AddChange("resource_id", resourceID)

	cm.auditLog.AppendEntry(entry)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

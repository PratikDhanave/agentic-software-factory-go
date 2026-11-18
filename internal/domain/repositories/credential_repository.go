package repositories

import (
	"github.com/PratikDhanave/agentic-software-factory-go/internal/domain/entities"
)

type CredentialRepository interface {
	SaveCredential(credential *entities.Credential) error
	GetCredential(id string) (*entities.Credential, error)
	UpdateCredential(credential *entities.Credential) error
	GetCredentialsByOwner(ownerID string) ([]*entities.Credential, error)
	GetCredentialsByType(credType entities.CredentialType) ([]*entities.Credential, error)
	ListActiveCredentials() ([]*entities.Credential, error)
	DeleteCredential(id string) error
}

package repositories

import (
	"github.com/PratikDhanave/agentic-software-factory-go/internal/domain/entities"
)

type PrincipalRepository interface {
	SavePrincipal(principal *entities.Principal) error
	GetPrincipal(id string) (*entities.Principal, error)
	GetPrincipalByName(name string) (*entities.Principal, error)
	UpdatePrincipal(principal *entities.Principal) error
	ListPrincipals(principalType entities.PrincipalType) ([]*entities.Principal, error)
	DeletePrincipal(id string) error
}

package repositories

import (
	"time"

	"github.com/PratikDhanave/agentic-software-factory-go/internal/domain/entities"
)

type AuditRepository interface {
	AppendEntry(entry *entities.AuditEntry) error
	GetEntry(id string) (*entities.AuditEntry, error)
	GetEntriesByPrincipal(principalID string) ([]*entities.AuditEntry, error)
	GetEntriesByResource(resource entities.Resource, resourceID string) ([]*entities.AuditEntry, error)
	GetEntriesByDateRange(start, end time.Time) ([]*entities.AuditEntry, error)
	GetEntriesByStatus(status entities.AuditStatus) ([]*entities.AuditEntry, error)
	VerifyIntegrity() bool
}

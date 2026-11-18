package repositories

import (
	"github.com/PratikDhanave/agentic-software-factory-go/internal/domain/entities"
)

type CoordinationSessionRepository interface {
	SaveSession(session *entities.CoordinationSession) error
	GetSession(sessionID string) (*entities.CoordinationSession, error)
	UpdateSessionStatus(sessionID string, status entities.SessionStatus) error
	UpdateDecision(sessionID string, agentRole string, decision *entities.Decision) error
	ApproveSession(sessionID string, approver string) error
	ListPendingApprovals() ([]*entities.CoordinationSession, error)
	DeleteSession(sessionID string) error
}

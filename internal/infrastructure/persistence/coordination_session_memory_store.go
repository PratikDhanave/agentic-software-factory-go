package persistence

import (
	"fmt"
	"sync"

	"github.com/PratikDhanave/agentic-software-factory-go/internal/domain/entities"
	"github.com/PratikDhanave/agentic-software-factory-go/internal/domain/repositories"
)

type CoordinationSessionMemoryStore struct {
	sessions map[string]*entities.CoordinationSession
	mu       sync.RWMutex
}

func NewCoordinationSessionMemoryStore() repositories.CoordinationSessionRepository {
	return &CoordinationSessionMemoryStore{
		sessions: make(map[string]*entities.CoordinationSession),
	}
}

func (store *CoordinationSessionMemoryStore) SaveSession(session *entities.CoordinationSession) error {
	store.mu.Lock()
	defer store.mu.Unlock()

	if session.ID == "" {
		return fmt.Errorf("session ID cannot be empty")
	}

	store.sessions[session.ID] = session
	return nil
}

func (store *CoordinationSessionMemoryStore) GetSession(sessionID string) (*entities.CoordinationSession, error) {
	store.mu.RLock()
	defer store.mu.RUnlock()

	session, exists := store.sessions[sessionID]
	if !exists {
		return nil, fmt.Errorf("session not found: %s", sessionID)
	}

	return session, nil
}

func (store *CoordinationSessionMemoryStore) UpdateSessionStatus(sessionID string, status entities.SessionStatus) error {
	store.mu.Lock()
	defer store.mu.Unlock()

	session, exists := store.sessions[sessionID]
	if !exists {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	session.Status = status
	return nil
}

func (store *CoordinationSessionMemoryStore) UpdateDecision(sessionID string, agentRole string, decision *entities.Decision) error {
	store.mu.Lock()
	defer store.mu.Unlock()

	session, exists := store.sessions[sessionID]
	if !exists {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	session.Decisions[agentRole] = decision
	return nil
}

func (store *CoordinationSessionMemoryStore) ApproveSession(sessionID string, approver string) error {
	store.mu.Lock()
	defer store.mu.Unlock()

	session, exists := store.sessions[sessionID]
	if !exists {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	if session.Status != entities.SessionWaitingApproval {
		return fmt.Errorf("session not in waiting_approval status: %s", session.Status)
	}

	session.Status = entities.SessionExecuting
	return nil
}

func (store *CoordinationSessionMemoryStore) ListPendingApprovals() ([]*entities.CoordinationSession, error) {
	store.mu.RLock()
	defer store.mu.RUnlock()

	pending := []*entities.CoordinationSession{}
	for _, session := range store.sessions {
		if session.Status == entities.SessionWaitingApproval {
			pending = append(pending, session)
		}
	}

	return pending, nil
}

func (store *CoordinationSessionMemoryStore) DeleteSession(sessionID string) error {
	store.mu.Lock()
	defer store.mu.Unlock()

	delete(store.sessions, sessionID)
	return nil
}

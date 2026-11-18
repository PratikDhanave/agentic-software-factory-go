package entities

import (
	"time"

	"github.com/google/uuid"
)

type PrincipalType string

const (
	PrincipalAgent   PrincipalType = "agent"
	PrincipalUser    PrincipalType = "user"
	PrincipalService PrincipalType = "service"
)

func (pt PrincipalType) IsValid() bool {
	return pt == PrincipalAgent || pt == PrincipalUser || pt == PrincipalService
}

type Action string

const (
	ActionCreate  Action = "create"
	ActionRead    Action = "read"
	ActionUpdate  Action = "update"
	ActionDelete  Action = "delete"
	ActionExecute Action = "execute"
	ActionOverride Action = "override"
)

type Resource string

const (
	ResourceWorkflow   Resource = "workflow"
	ResourceExecution  Resource = "execution"
	ResourceCredential Resource = "credential"
	ResourceAudit      Resource = "audit"
	ResourcePolicy     Resource = "policy"
	ResourceAgent      Resource = "agent"
	ResourceUser       Resource = "user"
)

type Permission struct {
	ID         string
	Resource   Resource
	Action     Action
	Scope      string
	Conditions []string
	CreatedAt  time.Time
}

type Principal struct {
	ID          string
	Type        PrincipalType
	Name        string
	DisplayName string
	Email       string
	Credentials map[string]interface{}
	Permissions []*Permission
	Active      bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
	LastSeenAt  *time.Time
}

type SecurityContext struct {
	CorrelationID string
	PrincipalID   string
	PrincipalType PrincipalType
	Permissions   []*Permission
	Timestamp     time.Time
	IPAddress     string
	UserAgent     string
}

func NewPrincipal(principalType PrincipalType, name string) *Principal {
	return &Principal{
		ID:          uuid.New().String(),
		Type:        principalType,
		Name:        name,
		DisplayName: name,
		Credentials: make(map[string]interface{}),
		Permissions: []*Permission{},
		Active:      true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
}

func NewPermission(resource Resource, action Action, scope string) *Permission {
	return &Permission{
		ID:         uuid.New().String(),
		Resource:   resource,
		Action:     action,
		Scope:      scope,
		Conditions: []string{},
		CreatedAt:  time.Now(),
	}
}

func NewSecurityContext(principalID string, principalType PrincipalType) *SecurityContext {
	return &SecurityContext{
		CorrelationID: uuid.New().String(),
		PrincipalID:   principalID,
		PrincipalType: principalType,
		Permissions:   []*Permission{},
		Timestamp:     time.Now(),
	}
}

func (sc *SecurityContext) WithIPAddress(ip string) *SecurityContext {
	sc.IPAddress = ip
	return sc
}

func (sc *SecurityContext) WithUserAgent(ua string) *SecurityContext {
	sc.UserAgent = ua
	return sc
}

func (sc *SecurityContext) AddPermissions(perms ...*Permission) *SecurityContext {
	sc.Permissions = append(sc.Permissions, perms...)
	return sc
}

func (p *Principal) HasPermission(resource Resource, action Action) bool {
	for _, perm := range p.Permissions {
		if perm.Resource == resource && perm.Action == action {
			return true
		}
	}
	return false
}

func (p *Principal) HasPermissionForScope(resource Resource, action Action, scope string) bool {
	for _, perm := range p.Permissions {
		if perm.Resource == resource && perm.Action == action {
			if perm.Scope == "*" || perm.Scope == scope {
				return true
			}
		}
	}
	return false
}

func (p *Principal) AddPermission(perm *Permission) {
	p.Permissions = append(p.Permissions, perm)
	p.UpdatedAt = time.Now()
}

func (p *Principal) SetCredential(key string, value interface{}) {
	p.Credentials[key] = value
	p.UpdatedAt = time.Now()
}

func (p *Principal) GetCredential(key string) (interface{}, bool) {
	val, exists := p.Credentials[key]
	return val, exists
}

func (p *Principal) Deactivate() {
	p.Active = false
	p.UpdatedAt = time.Now()
}

func (p *Principal) UpdateLastSeen() {
	now := time.Now()
	p.LastSeenAt = &now
}

func (sc *SecurityContext) CanPerform(resource Resource, action Action) bool {
	for _, perm := range sc.Permissions {
		if perm.Resource == resource && perm.Action == action {
			return true
		}
	}
	return false
}

func (sc *SecurityContext) CanPerformOnScope(resource Resource, action Action, scope string) bool {
	for _, perm := range sc.Permissions {
		if perm.Resource == resource && perm.Action == action {
			if perm.Scope == "*" || perm.Scope == scope {
				return true
			}
		}
	}
	return false
}

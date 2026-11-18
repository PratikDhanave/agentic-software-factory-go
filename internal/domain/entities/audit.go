package entities

import (
	"crypto/sha256"
	"encoding/hex"
	"time"

	"github.com/google/uuid"
)

type AuditStatus string

const (
	AuditSuccess    AuditStatus = "success"
	AuditFailure    AuditStatus = "failure"
	AuditDenied     AuditStatus = "denied"
	AuditTimeout    AuditStatus = "timeout"
	AuditPartial    AuditStatus = "partial"
)

type AuditEntry struct {
	ID            string
	CorrelationID string
	Timestamp     time.Time
	Principal     string
	PrincipalType PrincipalType
	Action        Action
	Resource      Resource
	ResourceID    string
	Status        AuditStatus
	IPAddress     string
	UserAgent     string
	Changes       map[string]interface{}
	Error         string
	Duration      time.Duration
	Hash          string
	PreviousHash  string
}

type AuditLog struct {
	Entries    []*AuditEntry
	LastHash   string
	EntryCount int
}

func NewAuditEntry(ctx *SecurityContext, action Action, resource Resource, resourceID string) *AuditEntry {
	return &AuditEntry{
		ID:            uuid.New().String(),
		CorrelationID: ctx.CorrelationID,
		Timestamp:     time.Now(),
		Principal:     ctx.PrincipalID,
		PrincipalType: ctx.PrincipalType,
		Action:        action,
		Resource:      resource,
		ResourceID:    resourceID,
		Status:        AuditSuccess,
		IPAddress:     ctx.IPAddress,
		UserAgent:     ctx.UserAgent,
		Changes:       make(map[string]interface{}),
	}
}

func (ae *AuditEntry) SetStatus(status AuditStatus) *AuditEntry {
	ae.Status = status
	return ae
}

func (ae *AuditEntry) SetError(err string) *AuditEntry {
	ae.Error = err
	ae.Status = AuditFailure
	return ae
}

func (ae *AuditEntry) SetDenied(reason string) *AuditEntry {
	ae.Error = reason
	ae.Status = AuditDenied
	return ae
}

func (ae *AuditEntry) AddChange(key string, value interface{}) *AuditEntry {
	ae.Changes[key] = value
	return ae
}

func (ae *AuditEntry) ComputeHash(previousHash string) string {
	ae.PreviousHash = previousHash

	hashInput := ae.ID +
		ae.CorrelationID +
		ae.Timestamp.String() +
		ae.Principal +
		string(ae.Action) +
		string(ae.Resource) +
		ae.ResourceID +
		string(ae.Status) +
		previousHash

	hash := sha256.Sum256([]byte(hashInput))
	ae.Hash = hex.EncodeToString(hash[:])

	return ae.Hash
}

func (ae *AuditEntry) VerifyHash(previousHash string) bool {
	computedHash := sha256.Sum256([]byte(
		ae.ID +
			ae.CorrelationID +
			ae.Timestamp.String() +
			ae.Principal +
			string(ae.Action) +
			string(ae.Resource) +
			ae.ResourceID +
			string(ae.Status) +
			previousHash,
	))

	expectedHash := hex.EncodeToString(computedHash[:])
	return ae.Hash == expectedHash
}

func NewAuditLog() *AuditLog {
	return &AuditLog{
		Entries:    []*AuditEntry{},
		LastHash:   "",
		EntryCount: 0,
	}
}

func (al *AuditLog) AppendEntry(entry *AuditEntry) error {
	entry.ComputeHash(al.LastHash)

	al.Entries = append(al.Entries, entry)
	al.LastHash = entry.Hash
	al.EntryCount++

	return nil
}

func (al *AuditLog) VerifyIntegrity() bool {
	currentHash := ""

	for _, entry := range al.Entries {
		if !entry.VerifyHash(currentHash) {
			return false
		}
		currentHash = entry.Hash
	}

	return al.LastHash == currentHash
}

func (al *AuditLog) GetEntriesByPrincipal(principalID string) []*AuditEntry {
	entries := []*AuditEntry{}
	for _, entry := range al.Entries {
		if entry.Principal == principalID {
			entries = append(entries, entry)
		}
	}
	return entries
}

func (al *AuditLog) GetEntriesByResource(resource Resource, resourceID string) []*AuditEntry {
	entries := []*AuditEntry{}
	for _, entry := range al.Entries {
		if entry.Resource == resource && entry.ResourceID == resourceID {
			entries = append(entries, entry)
		}
	}
	return entries
}

func (al *AuditLog) GetEntriesByAction(action Action) []*AuditEntry {
	entries := []*AuditEntry{}
	for _, entry := range al.Entries {
		if entry.Action == action {
			entries = append(entries, entry)
		}
	}
	return entries
}

func (al *AuditLog) GetEntriesByStatus(status AuditStatus) []*AuditEntry {
	entries := []*AuditEntry{}
	for _, entry := range al.Entries {
		if entry.Status == status {
			entries = append(entries, entry)
		}
	}
	return entries
}

func (al *AuditLog) GetEntriesByDateRange(start, end time.Time) []*AuditEntry {
	entries := []*AuditEntry{}
	for _, entry := range al.Entries {
		if entry.Timestamp.After(start) && entry.Timestamp.Before(end) {
			entries = append(entries, entry)
		}
	}
	return entries
}

type ComplianceReport struct {
	ID              string
	PeriodStart     time.Time
	PeriodEnd       time.Time
	TotalActions    int
	SuccessfulCount int
	DeniedCount     int
	FailureCount    int
	UniqueUsers     map[string]int
	ActionCounts    map[string]int
	Violations      []*AuditEntry
	RiskScore       float64
	GeneratedAt     time.Time
}

func NewComplianceReport(start, end time.Time, auditLog *AuditLog) *ComplianceReport {
	entries := auditLog.GetEntriesByDateRange(start, end)

	report := &ComplianceReport{
		ID:          uuid.New().String(),
		PeriodStart: start,
		PeriodEnd:   end,
		TotalActions: len(entries),
		UniqueUsers: make(map[string]int),
		ActionCounts: make(map[string]int),
		Violations:  []*AuditEntry{},
		GeneratedAt: time.Now(),
	}

	for _, entry := range entries {
		report.UniqueUsers[entry.Principal]++
		report.ActionCounts[string(entry.Action)]++

		switch entry.Status {
		case AuditSuccess:
			report.SuccessfulCount++
		case AuditDenied:
			report.DeniedCount++
			report.Violations = append(report.Violations, entry)
		case AuditFailure:
			report.FailureCount++
		}
	}

	report.CalculateRiskScore()

	return report
}

func (cr *ComplianceReport) CalculateRiskScore() {
	if cr.TotalActions == 0 {
		cr.RiskScore = 0
		return
	}

	deniedRatio := float64(cr.DeniedCount) / float64(cr.TotalActions)
	failureRatio := float64(cr.FailureCount) / float64(cr.TotalActions)

	cr.RiskScore = (deniedRatio * 0.6) + (failureRatio * 0.4)

	if cr.RiskScore > 1.0 {
		cr.RiskScore = 1.0
	}
}

func (cr *ComplianceReport) IsCompliant(threshold float64) bool {
	return cr.RiskScore <= threshold
}

package model

import (
	"errors"
	"regexp"
	"strings"
	"time"
)

type ImportStatus string

const (
	StatusQueued     ImportStatus = "QUEUED"
	StatusProcessing ImportStatus = "PROCESSING"
	StatusSucceeded  ImportStatus = "SUCCEEDED"
	StatusFailed     ImportStatus = "FAILED"
	StatusDeadLetter ImportStatus = "DEAD_LETTER"
)

type Import struct {
	ID               string       `json:"id"`
	TenantID         string       `json:"tenantId"`
	Source           string       `json:"source"`
	RecordCount      int          `json:"recordCount"`
	Status           ImportStatus `json:"status"`
	Attempts         int          `json:"attempts"`
	LastError        string       `json:"lastError,omitempty"`
	CreatedAt        time.Time    `json:"createdAt"`
	UpdatedAt        time.Time    `json:"updatedAt"`
	ProcessedAt      *time.Time   `json:"processedAt,omitempty"`
	IdempotentReplay bool         `json:"-"`
}

type CreateImportRequest struct {
	Source      string `json:"source"`
	RecordCount int    `json:"recordCount"`
}

type ValidationError struct {
	Fields map[string]string `json:"fields"`
}

func (e ValidationError) Error() string { return "invalid request" }

var identifierPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{1,31}$`)

func NormalizeTenant(value string) (string, error) {
	normalized := strings.ToLower(strings.TrimSpace(value))
	if !identifierPattern.MatchString(normalized) {
		return "", errors.New("tenant must have 2-32 lowercase letters, numbers, hyphens or underscores")
	}
	return normalized, nil
}

func (request *CreateImportRequest) Validate() error {
	request.Source = strings.ToLower(strings.TrimSpace(request.Source))
	problems := map[string]string{}

	if !identifierPattern.MatchString(request.Source) {
		problems["source"] = "must have 2-32 lowercase letters, numbers, hyphens or underscores"
	}
	if request.RecordCount < 1 || request.RecordCount > 10_000 {
		problems["recordCount"] = "must be between 1 and 10000"
	}
	if len(problems) > 0 {
		return ValidationError{Fields: problems}
	}
	return nil
}

type ImportRequestedEvent struct {
	ImportID      string            `json:"importId"`
	TenantID      string            `json:"tenantId"`
	Source        string            `json:"source"`
	RecordCount   int               `json:"recordCount"`
	CorrelationID string            `json:"correlationId"`
	TraceContext  map[string]string `json:"traceContext,omitempty"`
}

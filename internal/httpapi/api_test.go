package httpapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/asafeferreira/devops-observability-lab/internal/model"
	"github.com/asafeferreira/devops-observability-lab/internal/observability"
)

type fakeRepository struct {
	created *model.Import
}

func (repository *fakeRepository) CreateImport(_ context.Context, tenant, _ string, _ model.CreateImportRequest, _ string, _ map[string]string) (*model.Import, error) {
	copy := *repository.created
	copy.TenantID = tenant
	return &copy, nil
}

func (repository *fakeRepository) GetImport(context.Context, string, string) (*model.Import, error) {
	return repository.created, nil
}

func (repository *fakeRepository) ListImports(context.Context, string, int) ([]model.Import, error) {
	return []model.Import{*repository.created}, nil
}

func (repository *fakeRepository) Ping(context.Context) error { return nil }

func TestCreateImport(t *testing.T) {
	now := time.Now().UTC()
	repository := &fakeRepository{created: &model.Import{
		ID: "0198-import", Source: "erp", RecordCount: 25, Status: model.StatusQueued,
		CreatedAt: now, UpdatedAt: now,
	}}
	handler := New(repository, nil, observability.NewMetrics("test-api"))
	request := httptest.NewRequest(http.MethodPost, "/api/v1/imports", strings.NewReader(`{"source":"erp","recordCount":25}`))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-Tenant-ID", "Client-A")
	request.Header.Set("Idempotency-Key", "portfolio-test-001")
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", response.Code, response.Body.String())
	}
	if !strings.Contains(response.Body.String(), `"tenantId":"client-a"`) {
		t.Fatalf("tenant was not normalized: %s", response.Body.String())
	}
	if response.Header().Get("X-Correlation-ID") == "" {
		t.Fatal("missing correlation response header")
	}
}

func TestCreateImportRejectsInvalidTenant(t *testing.T) {
	handler := New(&fakeRepository{}, nil, observability.NewMetrics("test-api"))
	request := httptest.NewRequest(http.MethodPost, "/api/v1/imports", strings.NewReader(`{"source":"erp","recordCount":25}`))
	request.Header.Set("X-Tenant-ID", "invalid tenant")
	request.Header.Set("Idempotency-Key", "portfolio-test-001")
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", response.Code)
	}
}

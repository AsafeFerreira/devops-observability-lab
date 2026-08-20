package simulator

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/asafeferreira/devops-observability-lab/internal/observability"
)

func TestForcedFailureRequiresLabMode(t *testing.T) {
	requestBody := `{"importId":"import-1","source":"force-error","recordCount":10}`

	for _, test := range []struct {
		name       string
		labMode    bool
		wantStatus int
	}{
		{name: "disabled", labMode: false, wantStatus: http.StatusOK},
		{name: "enabled", labMode: true, wantStatus: http.StatusInternalServerError},
	} {
		t.Run(test.name, func(t *testing.T) {
			handler := New(test.labMode, observability.NewMetrics("simulator-test-"+test.name))
			request := httptest.NewRequest(http.MethodPost, "/api/v1/process", strings.NewReader(requestBody))
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != test.wantStatus {
				t.Fatalf("expected %d, got %d", test.wantStatus, response.Code)
			}
		})
	}
}

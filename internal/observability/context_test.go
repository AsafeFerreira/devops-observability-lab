package observability

import "testing"

func TestSanitizeCorrelationID(t *testing.T) {
	if got := SanitizeCorrelationID("safe-id_123"); got != "safe-id_123" {
		t.Fatalf("safe id changed: %q", got)
	}
	if got := SanitizeCorrelationID("unsafe id with spaces"); got == "unsafe id with spaces" || got == "" {
		t.Fatalf("unsafe id was not replaced: %q", got)
	}
}

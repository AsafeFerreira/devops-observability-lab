package config

import (
	"testing"
	"time"
)

func TestLoadUsesDefaults(t *testing.T) {
	t.Setenv("PORT", "")
	t.Setenv("LAB_MODE", "")
	t.Setenv("OUTBOX_INTERVAL", "")

	cfg := Load("test-service", "9999")

	if cfg.ServiceName != "test-service" || cfg.Port != "9999" {
		t.Fatalf("unexpected service config: %#v", cfg)
	}
	if cfg.LabMode {
		t.Fatal("lab mode must be disabled by default")
	}
	if cfg.OutboxInterval != time.Second {
		t.Fatalf("unexpected interval: %s", cfg.OutboxInterval)
	}
}

func TestLoadReadsEnvironment(t *testing.T) {
	t.Setenv("PORT", "7070")
	t.Setenv("LAB_MODE", "true")
	t.Setenv("OUTBOX_INTERVAL", "250ms")

	cfg := Load("test-service", "9999")

	if cfg.Port != "7070" || !cfg.LabMode || cfg.OutboxInterval != 250*time.Millisecond {
		t.Fatalf("environment was not applied: %#v", cfg)
	}
}

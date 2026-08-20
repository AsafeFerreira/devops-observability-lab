package model

import "testing"

func TestNormalizeTenant(t *testing.T) {
	tenant, err := NormalizeTenant(" Client-A ")
	if err != nil || tenant != "client-a" {
		t.Fatalf("unexpected result: tenant=%q err=%v", tenant, err)
	}

	if _, err := NormalizeTenant("contains spaces"); err == nil {
		t.Fatal("expected invalid tenant to be rejected")
	}
}

func TestCreateImportRequestValidate(t *testing.T) {
	request := CreateImportRequest{Source: " ERP ", RecordCount: 50}
	if err := request.Validate(); err != nil {
		t.Fatalf("valid request rejected: %v", err)
	}
	if request.Source != "erp" {
		t.Fatalf("source was not normalized: %q", request.Source)
	}

	invalid := CreateImportRequest{Source: "", RecordCount: 0}
	if err := invalid.Validate(); err == nil {
		t.Fatal("invalid request accepted")
	}
}

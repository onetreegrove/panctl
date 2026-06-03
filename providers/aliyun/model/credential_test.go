package model

import "testing"

func TestCredentialDefaultsAndRedaction(t *testing.T) {
	cred := Credential{RefreshToken: "abcdefghijklmnopqrstuvwxyz"}.WithDefaults()
	if cred.DriveType != DriveTypeResource {
		t.Fatalf("expected default drive type resource, got %q", cred.DriveType)
	}
	if got := cred.RedactedRefreshToken(); got != "abcd***wxyz" {
		t.Fatalf("unexpected redaction: %s", got)
	}
}

func TestCredentialNormalizesInvalidDriveType(t *testing.T) {
	cred := Credential{RefreshToken: "token", DriveType: "invalid"}.WithDefaults()
	if cred.DriveType != DriveTypeResource {
		t.Fatalf("invalid drive type should normalize to resource, got %q", cred.DriveType)
	}
}

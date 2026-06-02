package model

import "testing"

func TestCredentialRedactsRefreshToken(t *testing.T) {
	cred := Credential{
		RefreshToken: "abcdefghijklmnopqrstuvwxyz",
		ClientID:     "client-id",
		ClientSecret: "client-secret",
	}

	if got := cred.RedactedRefreshToken(); got != "abcd***wxyz" {
		t.Fatalf("redacted refresh token = %q", got)
	}
}

func TestCredentialDefaultsClientKeys(t *testing.T) {
	cred := Credential{RefreshToken: "token"}

	got := cred.WithDefaults()

	if got.ClientID == "" || got.ClientSecret == "" {
		t.Fatalf("expected default client keys, got %+v", got)
	}
	if got.RefreshToken != "token" {
		t.Fatalf("refresh token changed: %+v", got)
	}
}

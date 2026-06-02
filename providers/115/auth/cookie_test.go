package auth

import "testing"

func TestParseCookieExtractsRequiredFields(t *testing.T) {
	cred, err := ParseCookie("UID=u; CID=c; SEID=s; KID=k; OTHER=x")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if cred.UID != "u" || cred.CID != "c" || cred.SEID != "s" || cred.KID != "k" {
		t.Fatalf("credential = %+v", cred)
	}
}

func TestParseCookieRejectsMissingFields(t *testing.T) {
	_, err := ParseCookie("UID=u;CID=c")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestCredentialRedaction(t *testing.T) {
	cred, err := ParseCookie("UID=abcdef;CID=c;SEID=s;KID=k")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if got := cred.RedactedUID(); got != "abc***def" {
		t.Fatalf("redacted uid = %q", got)
	}
}

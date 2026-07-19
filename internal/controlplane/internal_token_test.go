package controlplane

import "testing"

func TestCheckInternalToken(t *testing.T) {
	t.Parallel()
	if err := CheckInternalToken("secret", ""); err == nil {
		t.Fatal("empty header should fail")
	}
	if err := CheckInternalToken("secret", "wrong"); err == nil {
		t.Fatal("wrong token should fail")
	}
	if err := CheckInternalToken("secret", "secret"); err != nil {
		t.Fatalf("matching token: %v", err)
	}
	if err := CheckInternalToken("", "anything"); err == nil {
		t.Fatal("empty configured token should reject all HTTP internal calls")
	}
}

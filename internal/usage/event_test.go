package usage

import "testing"

func TestEventMarkBYOK(t *testing.T) {
	e := Event{CredentialID: "cred_1"}
	e.MarkBYOK()
	if !e.UsedBYOK {
		t.Fatal("expected used_byok when credential_id set")
	}
	e2 := Event{}
	e2.MarkBYOK()
	if e2.UsedBYOK {
		t.Fatal("expected used_byok false without credential")
	}
}

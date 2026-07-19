package controlplane

import (
	"testing"
	"time"
)

func TestPasswordAndJWT(t *testing.T) {
	hash, err := HashPassword("admin")
	if err != nil {
		t.Fatal(err)
	}
	if !CheckPassword(hash, "admin") {
		t.Fatal("password should match")
	}
	if CheckPassword(hash, "wrong") {
		t.Fatal("password should not match")
	}

	tok, err := IssueToken("secret", time.Hour, "u1", "a@b.c", "admin")
	if err != nil {
		t.Fatal(err)
	}
	claims, err := ParseToken("secret", tok)
	if err != nil {
		t.Fatal(err)
	}
	if claims.UserID != "u1" || claims.Email != "a@b.c" {
		t.Fatalf("unexpected claims: %+v", claims)
	}
}

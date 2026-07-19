package auth_test

import (
	"testing"
	"time"

	"github.com/curefatih/afi/internal/adapters/auth"
	"github.com/golang-jwt/jwt/v5"
)

func TestHashAndCheckPassword(t *testing.T) {
	t.Parallel()
	hash, err := auth.HashPassword("s3cret")
	if err != nil {
		t.Fatal(err)
	}
	if hash == "" || hash == "s3cret" {
		t.Fatalf("unexpected hash %q", hash)
	}
	if !auth.CheckPassword(hash, "s3cret") {
		t.Fatal("expected password match")
	}
	if auth.CheckPassword(hash, "wrong") {
		t.Fatal("expected password mismatch")
	}
}

func TestIssueAndParseToken(t *testing.T) {
	t.Parallel()
	const secret = "test-jwt-secret"
	tok, err := auth.IssueToken(secret, time.Hour, "user_1", "a@b.c", "admin")
	if err != nil {
		t.Fatal(err)
	}
	claims, err := auth.ParseToken(secret, tok)
	if err != nil {
		t.Fatal(err)
	}
	if claims.UserID != "user_1" || claims.Email != "a@b.c" || claims.Role != "admin" {
		t.Fatalf("claims=%+v", claims)
	}
	if claims.Subject != "user_1" {
		t.Fatalf("subject=%q", claims.Subject)
	}
}

func TestParseTokenRejectsBadSecret(t *testing.T) {
	t.Parallel()
	tok, err := auth.IssueToken("good", time.Hour, "u", "e@x.y", "admin")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := auth.ParseToken("bad", tok); err == nil {
		t.Fatal("expected error for wrong secret")
	}
}

func TestParseTokenRejectsWrongAlg(t *testing.T) {
	t.Parallel()
	claims := auth.Claims{
		UserID: "u",
		Email:  "e@x.y",
		Role:   "admin",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
			Subject:   "u",
		},
	}
	tok, err := jwt.NewWithClaims(jwt.SigningMethodNone, claims).SignedString(jwt.UnsafeAllowNoneSignatureType)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := auth.ParseToken("secret", tok); err == nil {
		t.Fatal("expected error for none alg")
	}
}

func TestParseTokenRejectsExpired(t *testing.T) {
	t.Parallel()
	tok, err := auth.IssueToken("secret", -time.Hour, "u", "e@x.y", "admin")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := auth.ParseToken("secret", tok); err == nil {
		t.Fatal("expected expired token error")
	}
}

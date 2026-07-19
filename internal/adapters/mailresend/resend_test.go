package mailresend_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/curefatih/afi/internal/adapters/mailresend"
	"github.com/curefatih/afi/internal/mail"
)

func TestSenderRequiresAPIKey(t *testing.T) {
	t.Parallel()
	s := mailresend.Sender{Cfg: mailresend.Config{From: "a@b.c"}}
	err := s.Send(context.Background(), mail.Message{To: "u@x.y", Subject: "hi"})
	if err == nil || !strings.Contains(err.Error(), "api key") {
		t.Fatalf("err=%v", err)
	}
}

func TestSenderPostsJSON(t *testing.T) {
	t.Parallel()
	var gotAuth string
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &gotBody)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id":"email_1"}`))
	}))
	defer srv.Close()

	s := mailresend.Sender{
		Cfg: mailresend.Config{
			APIKey:  "re_test_key",
			From:    "AFI <noreply@afi.local>",
			BaseURL: srv.URL,
		},
		Client: srv.Client(),
	}
	err := s.Send(context.Background(), mail.Message{
		To: "user@example.com", Subject: "Hello", TextBody: "text", HTMLBody: "<p>html</p>",
	})
	if err != nil {
		t.Fatal(err)
	}
	if gotAuth != "Bearer re_test_key" {
		t.Fatalf("auth=%q", gotAuth)
	}
	if gotBody["from"] != "AFI <noreply@afi.local>" || gotBody["subject"] != "Hello" {
		t.Fatalf("body=%v", gotBody)
	}
	to, _ := gotBody["to"].([]any)
	if len(to) != 1 || to[0] != "user@example.com" {
		t.Fatalf("to=%v", gotBody["to"])
	}
}

func TestSenderSurfacesHTTPErrors(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"message":"invalid"}`, http.StatusUnauthorized)
	}))
	defer srv.Close()

	s := mailresend.Sender{
		Cfg:    mailresend.Config{APIKey: "bad", From: "a@b.c", BaseURL: srv.URL},
		Client: srv.Client(),
	}
	err := s.Send(context.Background(), mail.Message{To: "u@x.y", Subject: "x"})
	if err == nil || !strings.Contains(err.Error(), "401") {
		t.Fatalf("err=%v", err)
	}
}

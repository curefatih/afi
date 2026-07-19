package mail_test

import (
	"context"
	"strings"
	"testing"

	"github.com/curefatih/afi/internal/mail"
)

func TestValidateProvider(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		wantErr bool
	}{
		{"smtp", false},
		{"RESEND", false},
		{"ses", false},
		{"log", false},
		{"", false},
		{"  smtp  ", false},
		{"sendgrid", true},
	}
	for _, tc := range cases {
		err := mail.ValidateProvider(tc.name)
		if tc.wantErr && err == nil {
			t.Fatalf("ValidateProvider(%q): want error", tc.name)
		}
		if !tc.wantErr && err != nil {
			t.Fatalf("ValidateProvider(%q): %v", tc.name, err)
		}
	}
}

func TestProviderName(t *testing.T) {
	t.Parallel()
	if got := mail.ProviderName("  SMTP "); got != "smtp" {
		t.Fatalf("got %q", got)
	}
}

func TestFormatAddress(t *testing.T) {
	t.Parallel()
	cases := []struct {
		in, display, email string
	}{
		{"", "AFI", "noreply@afi.local"},
		{"ops@afi.dev", "ops@afi.dev", "ops@afi.dev"},
		{"AFI <noreply@afi.dev>", "AFI", "noreply@afi.dev"},
		{" <solo@afi.dev>", "solo@afi.dev", "solo@afi.dev"},
	}
	for _, tc := range cases {
		display, email := mail.FormatAddress(tc.in)
		if display != tc.display || email != tc.email {
			t.Fatalf("FormatAddress(%q)=(%q,%q) want (%q,%q)", tc.in, display, email, tc.display, tc.email)
		}
	}
}

func TestBuildMIME(t *testing.T) {
	t.Parallel()
	raw := string(mail.BuildMIME("AFI <a@b.c>", "u@x.y", "Hello", "plain", "<b>html</b>"))
	for _, want := range []string{
		"From: AFI <a@b.c>",
		"To: u@x.y",
		"Subject: Hello",
		"Content-Type: multipart/alternative",
		"plain",
		"<b>html</b>",
		"--afi-mail-boundary--",
	} {
		if !strings.Contains(raw, want) {
			t.Fatalf("MIME missing %q\n%s", want, raw)
		}
	}
}

func TestInviteTemplatesEscapeHTML(t *testing.T) {
	t.Parallel()
	msg := mail.InviteNewUser(`Org <script>`, `https://app/invite?x="1"`)
	if strings.Contains(msg.HTMLBody, "<script>") {
		t.Fatal("org name not escaped")
	}
	if !strings.Contains(msg.HTMLBody, "Org &lt;script&gt;") {
		t.Fatalf("expected escaped org, got %s", msg.HTMLBody)
	}
	if !strings.Contains(msg.HTMLBody, "https://app/invite?x=&#34;1&#34;") &&
		!strings.Contains(msg.HTMLBody, `https://app/invite?x=&quot;1&quot;`) {
		t.Fatalf("expected escaped URL, got %s", msg.HTMLBody)
	}
	if !strings.Contains(msg.Subject, "Org <script>") {
		t.Fatalf("subject should keep raw org for text clients: %s", msg.Subject)
	}

	existing := mail.InviteExistingUser("Acme", "https://app/login")
	if existing.Subject == "" || !strings.Contains(existing.TextBody, "https://app/login") {
		t.Fatalf("bad existing-user invite: %+v", existing)
	}
}

func TestTestMessage(t *testing.T) {
	t.Parallel()
	msg := mail.TestMessage()
	if msg.Subject != "AFI mail test" || msg.TextBody == "" || msg.HTMLBody == "" {
		t.Fatalf("unexpected test message: %+v", msg)
	}
}

func TestMemorySenderRecords(t *testing.T) {
	t.Parallel()
	var s mail.MemorySender
	msg := mail.Message{To: "a@b.c", Subject: "hi", TextBody: "body"}
	if err := s.Send(context.Background(), msg); err != nil {
		t.Fatal(err)
	}
	if len(s.Sent) != 1 || s.Sent[0].To != "a@b.c" {
		t.Fatalf("sent=%+v", s.Sent)
	}
}

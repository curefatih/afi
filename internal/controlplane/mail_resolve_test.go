package controlplane

import (
	"log/slog"
	"testing"

	"github.com/curefatih/afi/internal/adapters/mailresend"
	"github.com/curefatih/afi/internal/adapters/mailsmtp"
	"github.com/curefatih/afi/internal/kernel"
	"github.com/curefatih/afi/internal/mail"
)

func TestResolveMailSenderLogFallback(t *testing.T) {
	t.Parallel()
	cfg := &kernel.Config{}
	sender, provider, err := resolveMailSender(cfg, "", slog.Default())
	if err != nil {
		t.Fatal(err)
	}
	if provider != mail.ProviderLog {
		t.Fatalf("provider=%q", provider)
	}
	if _, ok := sender.(mail.LogSender); !ok {
		t.Fatalf("sender type %T", sender)
	}
}

func TestResolveMailSenderExplicitLog(t *testing.T) {
	t.Parallel()
	cfg := &kernel.Config{}
	cfg.Mail.SMTP.Enabled = true
	sender, provider, err := resolveMailSender(cfg, mail.ProviderLog, slog.Default())
	if err != nil {
		t.Fatal(err)
	}
	if provider != mail.ProviderLog {
		t.Fatalf("provider=%q", provider)
	}
	if _, ok := sender.(mail.LogSender); !ok {
		t.Fatalf("sender type %T", sender)
	}
}

func TestResolveMailSenderDisabledPreferred(t *testing.T) {
	t.Parallel()
	cfg := &kernel.Config{}
	cfg.Mail.SMTP.Enabled = true
	_, _, err := resolveMailSender(cfg, mail.ProviderResend, slog.Default())
	if err == nil {
		t.Fatal("expected disabled provider error")
	}
}

func TestResolveMailSenderSMTP(t *testing.T) {
	t.Parallel()
	cfg := &kernel.Config{}
	cfg.Mail.From = "AFI <noreply@afi.local>"
	cfg.Mail.SMTP.Enabled = true
	cfg.Mail.SMTP.Host = "smtp.example"
	cfg.Mail.SMTP.Port = 587
	sender, provider, err := resolveMailSender(cfg, "", slog.Default())
	if err != nil {
		t.Fatal(err)
	}
	if provider != mail.ProviderSMTP {
		t.Fatalf("provider=%q", provider)
	}
	s, ok := sender.(mailsmtp.Sender)
	if !ok {
		t.Fatalf("sender type %T", sender)
	}
	if s.Cfg.Host != "smtp.example" || s.Cfg.Port != 587 {
		t.Fatalf("smtp cfg=%+v", s.Cfg)
	}
}

func TestResolveMailSenderResendAndOrgOverride(t *testing.T) {
	t.Parallel()
	cfg := &kernel.Config{}
	cfg.Mail.DefaultProvider = mail.ProviderSMTP
	cfg.Mail.SMTP.Enabled = true
	cfg.Mail.Resend.Enabled = true
	cfg.Mail.Resend.APIKey = "re_test"
	sender, provider, err := resolveMailSender(cfg, mail.ProviderResend, slog.Default())
	if err != nil {
		t.Fatal(err)
	}
	if provider != mail.ProviderResend {
		t.Fatalf("provider=%q", provider)
	}
	s, ok := sender.(mailresend.Sender)
	if !ok {
		t.Fatalf("sender type %T", sender)
	}
	if s.Cfg.APIKey != "re_test" {
		t.Fatalf("api key=%q", s.Cfg.APIKey)
	}
}

func TestResolveMailSenderSESNotImplemented(t *testing.T) {
	t.Parallel()
	cfg := &kernel.Config{}
	cfg.Mail.SES.Enabled = true
	_, _, err := resolveMailSender(cfg, mail.ProviderSES, slog.Default())
	if err == nil {
		t.Fatal("expected ses not implemented")
	}
}

func TestEnabledMailProviders(t *testing.T) {
	t.Parallel()
	cfg := &kernel.Config{}
	cfg.Mail.SMTP.Enabled = true
	cfg.Mail.Resend.Enabled = true
	got := enabledMailProviders(cfg)
	if len(got) != 2 || got[0] != mail.ProviderSMTP || got[1] != mail.ProviderResend {
		t.Fatalf("got=%v", got)
	}
}

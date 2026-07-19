package controlplane

import (
	"fmt"
	"log/slog"

	"github.com/curefatih/afi/internal/adapters/mailresend"
	"github.com/curefatih/afi/internal/adapters/mailsmtp"
	"github.com/curefatih/afi/internal/kernel"
	"github.com/curefatih/afi/internal/mail"
)

func enabledMailProviders(cfg *kernel.Config) []string {
	var out []string
	if cfg.Mail.SMTP.Enabled {
		out = append(out, mail.ProviderSMTP)
	}
	if cfg.Mail.Resend.Enabled {
		out = append(out, mail.ProviderResend)
	}
	if cfg.Mail.SES.Enabled {
		out = append(out, mail.ProviderSES)
	}
	return out
}

func resolveMailSender(cfg *kernel.Config, orgProvider string, log *slog.Logger) (mail.Sender, string, error) {
	preferred := mail.ProviderName(orgProvider)
	if preferred == "" {
		preferred = mail.ProviderName(cfg.Mail.DefaultProvider)
	}
	enabled := map[string]bool{}
	for _, p := range enabledMailProviders(cfg) {
		enabled[p] = true
	}
	if preferred == mail.ProviderLog || (preferred == "" && len(enabled) == 0) {
		return mail.LogSender{Log: log}, mail.ProviderLog, nil
	}
	if preferred != "" && !enabled[preferred] {
		return nil, "", fmt.Errorf("mail provider %q is not enabled", preferred)
	}
	if preferred == "" {
		for _, p := range []string{mail.ProviderSMTP, mail.ProviderResend, mail.ProviderSES} {
			if enabled[p] {
				preferred = p
				break
			}
		}
	}
	if preferred == "" {
		return mail.LogSender{Log: log}, mail.ProviderLog, nil
	}
	switch preferred {
	case mail.ProviderSMTP:
		return mailsmtp.Sender{Cfg: mailsmtp.Config{
			Host: cfg.Mail.SMTP.Host, Port: cfg.Mail.SMTP.Port,
			Username: cfg.Mail.SMTP.Username, Password: cfg.Mail.SMTP.Password,
			TLS: cfg.Mail.SMTP.TLS, From: cfg.Mail.From,
		}}, mail.ProviderSMTP, nil
	case mail.ProviderResend:
		return mailresend.Sender{Cfg: mailresend.Config{
			APIKey: cfg.Mail.Resend.APIKey, From: cfg.Mail.From,
		}}, mail.ProviderResend, nil
	case mail.ProviderSES:
		return nil, "", fmt.Errorf("mail provider ses is not implemented yet; enable smtp or resend")
	default:
		return nil, "", fmt.Errorf("unknown mail provider %q", preferred)
	}
}

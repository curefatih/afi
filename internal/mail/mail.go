package mail

import (
	"context"
	"fmt"
	"strings"
)

const (
	ProviderSMTP   = "smtp"
	ProviderResend = "resend"
	ProviderSES    = "ses"
	ProviderLog    = "log"
)

// Message is an outbound email.
type Message struct {
	To       string
	Subject  string
	TextBody string
	HTMLBody string
}

// Sender delivers email messages.
type Sender interface {
	Send(ctx context.Context, msg Message) error
}

// ProviderName normalizes a mail provider id.
func ProviderName(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

// ValidateProvider reports whether name is a known provider id.
func ValidateProvider(name string) error {
	switch ProviderName(name) {
	case ProviderSMTP, ProviderResend, ProviderSES, ProviderLog, "":
		return nil
	default:
		return fmt.Errorf("unknown mail provider %q", name)
	}
}

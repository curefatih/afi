package mail

import (
	"bytes"
	"fmt"
	"html"
	"strings"
)

// InviteNewUser builds the invite email for an unknown address.
func InviteNewUser(orgName, acceptURL string) Message {
	org := strings.TrimSpace(orgName)
	if org == "" {
		org = "an organization"
	}
	subject := fmt.Sprintf("You're invited to join %s on AFI", org)
	text := fmt.Sprintf(
		"You've been invited to join %s on AFI.\n\nAccept your invite:\n%s\n\nThis link expires soon. If you did not expect this email, you can ignore it.\n",
		org, acceptURL,
	)
	safeOrg := html.EscapeString(org)
	safeURL := html.EscapeString(acceptURL)
	htmlBody := fmt.Sprintf(
		`<p>You've been invited to join <strong>%s</strong> on AFI.</p><p><a href="%s">Accept your invite</a></p><p>This link expires soon. If you did not expect this email, you can ignore it.</p>`,
		safeOrg, safeURL,
	)
	return Message{Subject: subject, TextBody: text, HTMLBody: htmlBody}
}

// InviteExistingUser builds the notification for an existing platform user.
func InviteExistingUser(orgName, loginURL string) Message {
	org := strings.TrimSpace(orgName)
	if org == "" {
		org = "an organization"
	}
	subject := fmt.Sprintf("You've been added to %s on AFI", org)
	text := fmt.Sprintf(
		"You've been added to %s on AFI.\n\nSign in:\n%s\n",
		org, loginURL,
	)
	safeOrg := html.EscapeString(org)
	safeURL := html.EscapeString(loginURL)
	htmlBody := fmt.Sprintf(
		`<p>You've been added to <strong>%s</strong> on AFI.</p><p><a href="%s">Sign in</a></p>`,
		safeOrg, safeURL,
	)
	return Message{Subject: subject, TextBody: text, HTMLBody: htmlBody}
}

// TestMessage builds a simple connectivity test email.
func TestMessage() Message {
	return Message{
		Subject:  "AFI mail test",
		TextBody: "This is a test email from AFI. Mail delivery is working.\n",
		HTMLBody: "<p>This is a test email from AFI. Mail delivery is working.</p>",
	}
}

// PasswordReset builds the password recovery email.
func PasswordReset(resetURL string) Message {
	subject := "Reset your AFI password"
	text := fmt.Sprintf(
		"Reset your AFI password using this link:\n%s\n\nThis link expires in one hour. If you did not request a reset, you can ignore this email.\n",
		resetURL,
	)
	safeURL := html.EscapeString(resetURL)
	htmlBody := fmt.Sprintf(
		`<p>Reset your AFI password:</p><p><a href="%s">Choose a new password</a></p><p>This link expires in one hour. If you did not request a reset, you can ignore this email.</p>`,
		safeURL,
	)
	return Message{Subject: subject, TextBody: text, HTMLBody: htmlBody}
}

// FormatAddress parses "Name <email>" or bare email for SMTP MAIL FROM / headers.
func FormatAddress(from string) (display, email string) {
	from = strings.TrimSpace(from)
	if from == "" {
		return "AFI", "noreply@afi.local"
	}
	if i := strings.Index(from, "<"); i >= 0 {
		j := strings.Index(from, ">")
		if j > i {
			display = strings.TrimSpace(from[:i])
			email = strings.TrimSpace(from[i+1 : j])
			if display == "" {
				display = email
			}
			return display, email
		}
	}
	return from, from
}

// BuildMIME builds a simple multipart/alternative body.
func BuildMIME(from, to, subject, textBody, htmlBody string) []byte {
	var buf bytes.Buffer
	boundary := "afi-mail-boundary"
	_, _ = fmt.Fprintf(&buf, "From: %s\r\n", from)
	_, _ = fmt.Fprintf(&buf, "To: %s\r\n", to)
	_, _ = fmt.Fprintf(&buf, "Subject: %s\r\n", subject)
	_, _ = fmt.Fprintf(&buf, "MIME-Version: 1.0\r\n")
	_, _ = fmt.Fprintf(&buf, "Content-Type: multipart/alternative; boundary=%s\r\n\r\n", boundary)
	_, _ = fmt.Fprintf(&buf, "--%s\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n%s\r\n", boundary, textBody)
	_, _ = fmt.Fprintf(&buf, "--%s\r\nContent-Type: text/html; charset=UTF-8\r\n\r\n%s\r\n", boundary, htmlBody)
	_, _ = fmt.Fprintf(&buf, "--%s--\r\n", boundary)
	return buf.Bytes()
}

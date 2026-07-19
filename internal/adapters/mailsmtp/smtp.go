package mailsmtp

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/smtp"
	"strconv"

	"github.com/curefatih/afi/internal/mail"
)

// Config holds SMTP transport settings.
type Config struct {
	Host     string
	Port     int
	Username string
	Password string
	TLS      bool
	From     string
}

// Sender delivers mail over SMTP.
type Sender struct {
	Cfg Config
}

func (s Sender) Send(ctx context.Context, msg mail.Message) error {
	_ = ctx
	fromDisplay, fromEmail := mail.FormatAddress(s.Cfg.From)
	fromHeader := fromEmail
	if fromDisplay != fromEmail {
		fromHeader = fmt.Sprintf("%s <%s>", fromDisplay, fromEmail)
	}
	raw := mail.BuildMIME(fromHeader, msg.To, msg.Subject, msg.TextBody, msg.HTMLBody)
	addr := net.JoinHostPort(s.Cfg.Host, strconv.Itoa(s.Cfg.Port))

	var auth smtp.Auth
	if s.Cfg.Username != "" {
		auth = smtp.PlainAuth("", s.Cfg.Username, s.Cfg.Password, s.Cfg.Host)
	}

	if s.Cfg.TLS {
		return sendTLS(addr, s.Cfg.Host, auth, fromEmail, msg.To, raw)
	}
	return smtp.SendMail(addr, auth, fromEmail, []string{msg.To}, raw)
}

func sendTLS(addr, host string, auth smtp.Auth, from, to string, raw []byte) error {
	conn, err := tls.Dial("tcp", addr, &tls.Config{ServerName: host})
	if err != nil {
		return err
	}
	defer conn.Close()
	c, err := smtp.NewClient(conn, host)
	if err != nil {
		return err
	}
	defer c.Close()
	if auth != nil {
		if err := c.Auth(auth); err != nil {
			return err
		}
	}
	if err := c.Mail(from); err != nil {
		return err
	}
	if err := c.Rcpt(to); err != nil {
		return err
	}
	w, err := c.Data()
	if err != nil {
		return err
	}
	if _, err := w.Write(raw); err != nil {
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}
	return c.Quit()
}

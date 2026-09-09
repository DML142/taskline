package invite

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/mail"
	"net/smtp"
	"strconv"

	"taskline/apps/api/internal/platform/config"
)

type Message struct {
	To            string
	WorkspaceName string
	AcceptanceURL string
}

type Mailer interface {
	Send(context.Context, Message) error
}

type SMTPMailer struct {
	config config.SMTPConfig
}

func NewSMTPMailer(cfg config.SMTPConfig) *SMTPMailer {
	return &SMTPMailer{config: cfg}
}

func (m *SMTPMailer) Send(ctx context.Context, message Message) error {
	dialer := net.Dialer{}
	conn, err := dialer.DialContext(ctx, "tcp", net.JoinHostPort(m.config.Host, strconv.Itoa(m.config.Port)))
	if err != nil {
		return fmt.Errorf("dial SMTP: %w", err)
	}
	defer conn.Close()
	client, err := smtp.NewClient(conn, m.config.Host)
	if err != nil {
		return fmt.Errorf("create SMTP client: %w", err)
	}
	defer client.Quit()
	if m.config.TLSMode == "starttls" {
		if ok, _ := client.Extension("STARTTLS"); !ok {
			return fmt.Errorf("SMTP server does not support STARTTLS")
		}
		if err := client.StartTLS(&tls.Config{ServerName: m.config.Host, MinVersion: tls.VersionTLS12}); err != nil {
			return fmt.Errorf("start SMTP TLS: %w", err)
		}
	}
	if m.config.Username != "" || m.config.Password != "" {
		if err := client.Auth(smtp.PlainAuth("", m.config.Username, m.config.Password, m.config.Host)); err != nil {
			return fmt.Errorf("authenticate SMTP: %w", err)
		}
	}
	from, err := mail.ParseAddress(m.config.From)
	if err != nil {
		return fmt.Errorf("parse SMTP sender: %w", err)
	}
	if err := client.Mail(from.Address); err != nil {
		return fmt.Errorf("set SMTP sender: %w", err)
	}
	if err := client.Rcpt(message.To); err != nil {
		return fmt.Errorf("set SMTP recipient: %w", err)
	}
	writer, err := client.Data()
	if err != nil {
		return fmt.Errorf("start SMTP message: %w", err)
	}
	body := "To: " + message.To + "\r\nFrom: " + m.config.From + "\r\nSubject: Invitation to " + message.WorkspaceName + "\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\nAccept this workspace invitation: " + message.AcceptanceURL + "\r\n"
	if _, err := writer.Write([]byte(body)); err != nil {
		return fmt.Errorf("write SMTP message: %w", err)
	}
	if err := writer.Close(); err != nil {
		return fmt.Errorf("finish SMTP message: %w", err)
	}
	return nil
}

type disabledMailer struct{}

func (disabledMailer) Send(context.Context, Message) error {
	return fmt.Errorf("invite mailer is not configured")
}

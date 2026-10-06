package mailer

import (
	"bytes"
	"embed"
	"fmt"
	"html/template"
	"net/smtp"
	"strings"
	"time"
)

//go:embed templates/*
var templateFS embed.FS

type Client interface {
	Send(recipient, templateFile string, data any) error
}

type Config struct {
	Host     string
	Port     int
	Username string
	Password string
	Sender   string
}

type SMTPMailer struct {
	config Config
}

func NewSMTPMailer(cfg Config) *SMTPMailer {
	return &SMTPMailer{config: cfg}
}

func (m *SMTPMailer) Send(recipient, templateFile string, data any) error {
	tmpl, err := template.New("email").ParseFS(templateFS, "templates/"+templateFile)
	if err != nil {
		return fmt.Errorf("parsing template: %w", err)
	}

	var subject, body bytes.Buffer
	if err := tmpl.ExecuteTemplate(&subject, "subject", data); err != nil {
		return fmt.Errorf("rendering subject: %w", err)
	}
	if err := tmpl.ExecuteTemplate(&body, "body", data); err != nil {
		return fmt.Errorf("rendering body: %w", err)
	}

	var msg bytes.Buffer
	fmt.Fprintf(&msg, "From: %s\r\n", m.config.Sender)
	fmt.Fprintf(&msg, "To: %s\r\n", recipient)
	fmt.Fprintf(&msg, "Subject: %s\r\n", strings.TrimSpace(subject.String()))
	fmt.Fprintf(&msg, "MIME-Version: 1.0\r\n")
	fmt.Fprintf(&msg, "Content-Type: text/html; charset=\"UTF-8\"\r\n")
	fmt.Fprintf(&msg, "Date: %s\r\n\r\n", time.Now().Format(time.RFC1123Z))
	msg.Write(body.Bytes())

	addr := fmt.Sprintf("%s:%d", m.config.Host, m.config.Port)
	var auth smtp.Auth
	if m.config.Username != "" {
		auth = smtp.PlainAuth("", m.config.Username, m.config.Password, m.config.Host)
	}

	return smtp.SendMail(addr, auth, m.config.Sender, []string{recipient}, msg.Bytes())
}

type MockMailer struct{}

func NewMockMailer() *MockMailer {
	return &MockMailer{}
}

func (m *MockMailer) Send(recipient, templateFile string, data any) error {
	return nil
}
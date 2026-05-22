package email

import (
	"fmt"
	"net"
	"net/smtp"
	"regexp"
	"strings"
)

type EmailSender struct {
	auth     smtp.Auth
	addr     string
	from     string
	password string
}

var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

func NewEmailSender(host, port, from, password string) (*EmailSender, error) {
	auth := smtp.PlainAuth("", from, password, host)
	addr := fmt.Sprintf("%s:%s", host, port)
	if err := smtp.SendMail(addr, auth, from, []string{from}, []byte("Subject: Test Email\n\nThis is a test email sent during the startup process.")); err != nil {
		return nil, fmt.Errorf("failed to send test email: %w", err)
	}
	return &EmailSender{
		auth:     auth,
		addr:     addr,
		from:     from,
		password: password,
	}, nil
}

func (es *EmailSender) SendVerificationEmail(to string, code int32) error {
	return es.SendEmail(to, "Movie Reserve - Verify Your Email", fmt.Sprintf("Your verification code is: %d", code))
}

func (es *EmailSender) SendEmail(to, subject, body string) error {
	if !IsValidEmail(to) {
		return fmt.Errorf("invalid email address: %s", to)
	}
	msg := fmt.Sprintf("From: %s\nTo: %s\nSubject: %s\n\n%s", es.from, to, subject, body)
	return smtp.SendMail(es.addr, es.auth, es.from, []string{to}, []byte(msg))
}

func IsValidEmail(email string) bool {
	if len(email) < 3 || len(email) > 254 {
		return false
	}
	if !emailRegex.MatchString(email) {
		return false
	}
	parts := strings.Split(email, "@")
	host := parts[1]
	_, err := net.LookupMX(host)
	if err != nil {
		return false
	}
	return true
}

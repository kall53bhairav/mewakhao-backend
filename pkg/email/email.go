package email

import (
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net/smtp"
	"strconv"
)

type Sender struct {
	host string
	port int
	user string
	pass string
	from string
}

func NewSender(host string, port int, user, pass, from string) *Sender {
	return &Sender{host: host, port: port, user: user, pass: pass, from: from}
}

func (s *Sender) isConfigured() bool {
	return s.host != "" && s.user != "" && s.pass != ""
}

// Send sends an HTML email. In development (SMTP not configured) it logs to stdout.
func (s *Sender) Send(to, subject, htmlBody string) error {
	if !s.isConfigured() {
		log.Printf("[EMAIL DEV] To=%s | Subject=%s | Body=%s", to, subject, htmlBody)
		return nil
	}

	addr := s.host + ":" + strconv.Itoa(s.port)
	auth := smtp.PlainAuth("", s.user, s.pass, s.host)
	msg := buildMessage(s.from, to, subject, htmlBody)

	if s.port == 465 {
		return sendTLS(addr, s.host, auth, s.from, to, msg)
	}
	return sendSTARTTLS(addr, s.host, auth, s.from, to, msg)
}

func buildMessage(from, to, subject, htmlBody string) string {
	return fmt.Sprintf(
		"From: MewaKhao <%s>\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/html; charset=\"UTF-8\"\r\n\r\n%s",
		from, to, subject, htmlBody,
	)
}

func sendSTARTTLS(addr, host string, auth smtp.Auth, from, to, msg string) error {
	client, err := smtp.Dial(addr)
	if err != nil {
		return fmt.Errorf("smtp dial: %w", err)
	}
	defer client.Close()

	if err = client.StartTLS(&tls.Config{ServerName: host}); err != nil {
		return fmt.Errorf("starttls: %w", err)
	}
	return sendViaClient(client, auth, from, to, msg)
}

func sendTLS(addr, host string, auth smtp.Auth, from, to, msg string) error {
	conn, err := tls.Dial("tcp", addr, &tls.Config{ServerName: host})
	if err != nil {
		return fmt.Errorf("tls dial: %w", err)
	}
	client, err := smtp.NewClient(conn, host)
	if err != nil {
		return fmt.Errorf("smtp client: %w", err)
	}
	defer client.Close()
	return sendViaClient(client, auth, from, to, msg)
}

func sendViaClient(client *smtp.Client, auth smtp.Auth, from, to, msg string) error {
	if err := client.Auth(auth); err != nil {
		return fmt.Errorf("smtp auth: %w", err)
	}
	if err := client.Mail(from); err != nil {
		return err
	}
	if err := client.Rcpt(to); err != nil {
		return err
	}
	w, err := client.Data()
	if err != nil {
		return err
	}
	if _, err = io.WriteString(w, msg); err != nil {
		return err
	}
	return w.Close()
}

// OTPEmailBody returns the HTML body for an OTP email.
func OTPEmailBody(otp string) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html>
<body style="font-family:sans-serif;max-width:480px;margin:40px auto;padding:24px;border:1px solid #e5e7eb;border-radius:8px">
  <h2 style="color:#16a34a;margin-bottom:8px">MewaKhao</h2>
  <p style="color:#374151;margin-bottom:24px">Use the code below to sign in. It expires in <strong>10 minutes</strong>.</p>
  <div style="background:#f3f4f6;border-radius:8px;padding:24px;text-align:center">
    <span style="font-size:36px;font-weight:bold;letter-spacing:8px;color:#111827">%s</span>
  </div>
  <p style="color:#6b7280;font-size:13px;margin-top:24px">If you didn&apos;t request this, you can safely ignore this email.</p>
</body>
</html>`, otp)
}

package provider

import (
	"context"
	"fmt"
	"log"
	"net/smtp"
	"os"
)

type SMTPProvider struct {
	host     string
	port     string
	user     string
	password string
	from     string
}

func NewSMTPProvider() NotificationProvider {
	return &SMTPProvider{
		host:     getEnvOrDefault("SMTP_HOST", "smtp.mailjet.com"),
		port:     getEnvOrDefault("SMTP_PORT", "587"),
		user:     getEnvOrDefault("SMTP_USER", ""),
		password: getEnvOrDefault("SMTP_PASSWORD", ""),
		from:     getEnvOrDefault("EMAIL_FROM", "noreply@example.com"),
	}
}

func (p *SMTPProvider) Send(ctx context.Context, n Notification) error {
	dollars := fmt.Sprintf("$%.2f", float64(n.Amount)/100.0)
	subject := fmt.Sprintf("Your Order #%s – Payment %s", n.OrderID, n.Status)
	body := fmt.Sprintf(
		"Hello,\n\nYour order #%s has been updated.\n\nAmount: %s\nStatus:  %s\n\nThank you for your business.",
		n.OrderID, dollars, n.Status,
	)
	msg := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\n\r\n%s",
		p.from, n.To, subject, body)

	addr := fmt.Sprintf("%s:%s", p.host, p.port)
	auth := smtp.PlainAuth("", p.user, p.password, p.host)

	if ctx.Err() != nil {
		return ctx.Err()
	}

	if err := smtp.SendMail(addr, auth, p.from, []string{n.To}, []byte(msg)); err != nil {
		return fmt.Errorf("smtp send: %w", err)
	}

	log.Printf("[SMTPProvider] ✉  Email sent  to=%s  order=%s  amount=%s", n.To, n.OrderID, dollars)
	return nil
}

func getEnvOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

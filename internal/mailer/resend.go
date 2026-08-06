// Package mailer отправляет транзакционные письма через Resend HTTP API.
// https://resend.com/docs/api-reference/emails/send-email
package mailer

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
)

// ResendMailer шлёт письма через Resend. Реализует интерфейс auth.Mailer (Send + Enabled),
// но сам этого интерфейса не импортирует — чтобы не тянуть зависимость service→mailer→service.
type ResendMailer struct {
	apiKey string
	from   string
	client *http.Client
}

// NewResendMailer создаёt мейлер напрямую. from — адрес отправителя целиком,
// напр. "Hanbin <noreply@hanbin-drama.com>".
func NewResendMailer(apiKey, from string) *ResendMailer {
	return &ResendMailer{
		apiKey: apiKey,
		from:   from,
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

// NewFromEnv создаёт ResendMailer, читая RESEND_API_KEY и (опционально) RESEND_FROM из окружения.
// Если RESEND_API_KEY не задан — Enabled() будет возвращать false, вызывающий код должен
// сам решить, что делать в этом случае (см. auth.Service.ForgotPassword).
func NewFromEnv() *ResendMailer {
	apiKey := os.Getenv("RESEND_API_KEY")
	from := os.Getenv("RESEND_FROM")
	if from == "" {
		from = "Hanbin <noreply@hanbin-drama.com>"
	}
	return NewResendMailer(apiKey, from)
}

// Enabled — настроен ли реально мейлер (есть ли API-ключ).
func (m *ResendMailer) Enabled() bool { return m.apiKey != "" }

type resendPayload struct {
	From    string   `json:"from"`
	To      []string `json:"to"`
	Subject string   `json:"subject"`
	HTML    string   `json:"html"`
}

// Send отправляет одно письмо. to — адрес получателя, htmlBody — HTML-содержимое письма.
func (m *ResendMailer) Send(to, subject, htmlBody string) error {
	if m.apiKey == "" {
		return fmt.Errorf("mailer: RESEND_API_KEY not set")
	}

	payload := resendPayload{
		From:    m.from,
		To:      []string{to},
		Subject: subject,
		HTML:    htmlBody,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("mailer: marshal payload: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, "https://api.resend.com/emails", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("mailer: build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+m.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := m.client.Do(req)
	if err != nil {
		return fmt.Errorf("mailer: send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("mailer: resend responded with status %d", resp.StatusCode)
	}
	return nil
}

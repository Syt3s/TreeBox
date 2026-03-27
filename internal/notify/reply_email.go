package notify

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"mime"
	"mime/quotedprintable"
	"net"
	"net/mail"
	"net/smtp"
	"strconv"
	"strings"
	"time"

	"github.com/syt3s/TreeBox/internal/branding"
	"github.com/syt3s/TreeBox/internal/config"
)

var ErrMailNotConfigured = errors.New("mail is not configured")

type ReplyEmailInput struct {
	To              string
	PageName        string
	PageDomain      string
	QuestionContent string
	AnswerContent   string
	QuestionURL     string
	IsPrivate       bool
}

type ReplyEmailSender interface {
	SendQuestionAnswered(ctx context.Context, input ReplyEmailInput) error
}

type smtpReplyEmailSender struct{}

var defaultReplyEmailSender ReplyEmailSender = smtpReplyEmailSender{}

func SetReplyEmailSender(sender ReplyEmailSender) func() {
	previous := defaultReplyEmailSender
	if sender == nil {
		sender = smtpReplyEmailSender{}
	}
	defaultReplyEmailSender = sender
	return func() {
		defaultReplyEmailSender = previous
	}
}

func SendQuestionAnswered(ctx context.Context, input ReplyEmailInput) error {
	return defaultReplyEmailSender.SendQuestionAnswered(ctx, input)
}

func (smtpReplyEmailSender) SendQuestionAnswered(ctx context.Context, input ReplyEmailInput) error {
	if strings.TrimSpace(input.To) == "" {
		return nil
	}
	if !mailConfigured() {
		return ErrMailNotConfigured
	}

	message, err := buildQuestionAnsweredMessage(input)
	if err != nil {
		return err
	}

	return sendMail(ctx, strings.TrimSpace(config.Mail.Account), []string{strings.TrimSpace(input.To)}, message)
}

func mailConfigured() bool {
	return strings.TrimSpace(config.Mail.Account) != "" &&
		strings.TrimSpace(config.Mail.SMTP) != "" &&
		config.Mail.Port > 0
}

func buildQuestionAnsweredMessage(input ReplyEmailInput) ([]byte, error) {
	pageName := strings.TrimSpace(input.PageName)
	if pageName == "" {
		pageName = strings.TrimSpace(input.PageDomain)
	}
	if pageName == "" {
		pageName = branding.ProductName
	}

	subject := fmt.Sprintf("%s answered your question on %s", pageName, branding.ProductName)

	bodyLines := []string{
		"Hello,",
		"",
		fmt.Sprintf("You received a new answer on %s.", branding.ProductName),
		fmt.Sprintf("Question box: %s", pageName),
	}

	if pageDomain := strings.TrimSpace(input.PageDomain); pageDomain != "" {
		bodyLines = append(bodyLines, fmt.Sprintf("Question box id: %s", pageDomain))
	}

	bodyLines = append(
		bodyLines,
		"",
		"Your question:",
		strings.TrimSpace(input.QuestionContent),
		"",
		"New answer:",
		strings.TrimSpace(input.AnswerContent),
	)

	if !input.IsPrivate && strings.TrimSpace(input.QuestionURL) != "" {
		bodyLines = append(bodyLines, "", fmt.Sprintf("Public page: %s", strings.TrimSpace(input.QuestionURL)))
	}

	bodyLines = append(bodyLines, "", "If you did not ask this question, you can ignore this email.")
	body := strings.Join(bodyLines, "\r\n")

	var message bytes.Buffer
	from := (&mail.Address{
		Name:    branding.ProductName,
		Address: strings.TrimSpace(config.Mail.Account),
	}).String()
	to := (&mail.Address{Address: strings.TrimSpace(input.To)}).String()

	headers := []string{
		fmt.Sprintf("From: %s", from),
		fmt.Sprintf("To: %s", to),
		fmt.Sprintf("Subject: %s", mime.QEncoding.Encode("utf-8", subject)),
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=UTF-8",
		"Content-Transfer-Encoding: quoted-printable",
		"",
	}

	for _, header := range headers {
		if _, err := message.WriteString(header + "\r\n"); err != nil {
			return nil, err
		}
	}

	bodyWriter := quotedprintable.NewWriter(&message)
	if _, err := bodyWriter.Write([]byte(body)); err != nil {
		return nil, err
	}
	if err := bodyWriter.Close(); err != nil {
		return nil, err
	}

	return message.Bytes(), nil
}

func sendMail(ctx context.Context, from string, to []string, message []byte) error {
	host := strings.TrimSpace(config.Mail.SMTP)
	addr := net.JoinHostPort(host, strconv.Itoa(config.Mail.Port))

	client, err := newSMTPClient(ctx, host, addr, config.Mail.Port)
	if err != nil {
		return fmt.Errorf("connect smtp server: %w", err)
	}
	defer func() { _ = client.Close() }()

	if config.Mail.Port != 465 {
		if ok, _ := client.Extension("STARTTLS"); ok {
			if err := client.StartTLS(&tls.Config{
				ServerName: host,
				MinVersion: tls.VersionTLS12,
			}); err != nil {
				return fmt.Errorf("start tls: %w", err)
			}
		}
	}

	if strings.TrimSpace(config.Mail.Password) != "" {
		auth := smtp.PlainAuth("", strings.TrimSpace(config.Mail.Account), strings.TrimSpace(config.Mail.Password), host)
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("authenticate smtp user: %w", err)
		}
	}

	if err := client.Mail(from); err != nil {
		return fmt.Errorf("set smtp sender: %w", err)
	}
	for _, recipient := range to {
		if err := client.Rcpt(recipient); err != nil {
			return fmt.Errorf("set smtp recipient: %w", err)
		}
	}

	writer, err := client.Data()
	if err != nil {
		return fmt.Errorf("open smtp data writer: %w", err)
	}

	if _, err := writer.Write(message); err != nil {
		_ = writer.Close()
		return fmt.Errorf("write smtp message: %w", err)
	}
	if err := writer.Close(); err != nil {
		return fmt.Errorf("close smtp data writer: %w", err)
	}

	if err := client.Quit(); err != nil {
		return fmt.Errorf("quit smtp session: %w", err)
	}

	return nil
}

func newSMTPClient(ctx context.Context, host, addr string, port int) (*smtp.Client, error) {
	dialer := &net.Dialer{
		Timeout: 10 * time.Second,
	}

	if port == 465 {
		conn, err := tls.DialWithDialer(dialer, "tcp", addr, &tls.Config{
			ServerName: host,
			MinVersion: tls.VersionTLS12,
		})
		if err != nil {
			return nil, err
		}
		_ = conn.SetDeadline(time.Now().Add(15 * time.Second))
		return smtp.NewClient(conn, host)
	}

	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return nil, err
	}
	_ = conn.SetDeadline(time.Now().Add(15 * time.Second))
	return smtp.NewClient(conn, host)
}

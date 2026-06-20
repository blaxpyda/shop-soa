package channels

import (
	"context"
	"fmt"
	"net/smtp"
	"os"
)

type emailDispatcher struct {
	host     string
	port     string
	username string
	password string
	from     string
}

func NewEmailDispatcher() Dispatcher {
	return &emailDispatcher{
		host:     os.Getenv("EMAIL_HOST"),
		port:     os.Getenv("EMAIL_PORT"),
		username: os.Getenv("EMAIL_USER"),
		password: os.Getenv("EMAIL_PASS"),
		from:     os.Getenv("EMAIL_FROM"),
	}
}

func (e *emailDispatcher) Send(_ context.Context, to, subject, body string) error {
	msg := fmt.Sprintf(
		"From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/html; charset=UTF-8\r\n\r\n%s",
		e.from, to, subject, body,
	)
	auth := smtp.PlainAuth("", e.username, e.password, e.host)
	return smtp.SendMail(fmt.Sprintf("%s:%s", e.host, e.port), auth, e.from, []string{to}, []byte(msg))
}

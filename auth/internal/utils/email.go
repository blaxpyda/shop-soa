package utils

import (
	"fmt"
	"net/smtp"
	"os"
)

type EmailService interface {
	SendPasswordResetEmail(email string, token string) error
	SendVerificationCode(email string, code string) error
}

type emailService struct {
	host     string
	port     int
	username string
	password string
}

func NewEmailService(host string, port int, username, password string) EmailService {
	return &emailService{
		host:     host,
		port:     port,
		username: username,
		password: password,
	}
}

func (e *emailService) SendPasswordResetEmail(to string, token string) error {
	resetURL := fmt.Sprintf("%s/reset-password?token=%s", os.Getenv("APP_URL"), token)

	subject := "Password Reset Request"
	body := fmt.Sprintf(`
		<html>
		<body>
			<h2>Password Reset</h2>
			<p>Click the link below to reset your password:</p>
			<a href="%s">Reset Password</a>
			<p>This link expires in 1 hour.</p>
			<p>If you didn't request a password reset, ignore this email.</p>
		</body>
		</html>
	`, resetURL)

	return e.sendEmail(to, subject, body)
}

func (e *emailService) SendVerificationCode(to string, code string) error {
	subject := "Verify Your Account - StockStar"
	body := fmt.Sprintf(`
		<html>
		<body style="font-family: Arial, sans-serif; max-width: 600px; margin: 0 auto;">
			<div style="background: #4CAF50; color: white; padding: 20px; text-align: center; border-radius: 8px 8px 0 0;">
				<h2 style="margin: 0;">StockStar</h2>
			</div>
			<div style="padding: 30px; background: #f9f9f9; border-radius: 0 0 8px 8px;">
				<h3>Verify Your Account</h3>
				<p>Enter the following code to verify your account:</p>
				<div style="background: #fff; border: 2px dashed #4CAF50; padding: 20px; text-align: center; margin: 20px 0; border-radius: 8px;">
					<span style="font-size: 32px; font-weight: bold; letter-spacing: 8px; color: #333;">%s</span>
				</div>
				<p style="color: #666;">This code expires in <strong>10 minutes</strong>.</p>
				<p style="color: #999; font-size: 12px;">If you didn't create an account, you can safely ignore this email.</p>
			</div>
		</body>
		</html>
	`, code)

	return e.sendEmail(to, subject, body)
}

func (e *emailService) sendEmail(to string, subject string, body string) error {
	addr := fmt.Sprintf("%s:%d", e.host, e.port)
	msg := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/html; charset=UTF-8\r\n\r\n%s",
		e.username, to, subject, body)
	auth := smtp.PlainAuth("", e.username, e.password, e.host)
	return smtp.SendMail(addr, auth, e.username, []string{to}, []byte(msg))
}

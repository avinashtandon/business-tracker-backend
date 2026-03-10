package service

import (
	"fmt"
	"net/smtp"
	"os"
	"time"
)

// SendResetEmail sends an email to the user with their password reset link.
func SendResetEmail(toEmail, resetURL string) error {
	smtpHost := os.Getenv("SMTP_HOST")
	smtpPort := os.Getenv("SMTP_PORT")
	smtpUser := os.Getenv("SMTP_USERNAME")
	smtpPass := os.Getenv("SMTP_PASSWORD")
	fromEmail := os.Getenv("SMTP_FROM_EMAIL")

	if smtpHost == "" || smtpPort == "" {
		fmt.Printf("MOCK EMAIL SENT TO %s:\nReset Link: %s\n\n", toEmail, resetURL)
		return nil
	}

	auth := smtp.PlainAuth("", smtpUser, smtpPass, smtpHost)

	subject := "Business Tracker - Password Reset Request"

	body := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<body style="font-family: Arial, sans-serif; line-height:1.6;">
<p>Hello,</p>

<p>We received a request to reset your password.</p>

<p>
<a href="%s" 
style="background:#007bff;color:white;padding:12px 20px;text-decoration:none;border-radius:6px;display:inline-block;">
Reset Password
</a>
</p>

<p>Or copy and paste this link into your browser:</p>

<p>%s</p>

<p style="font-size:12px;color:#666;">This link expires in 15 minutes.</p>
</body>
</html>
`, resetURL, resetURL)

	message := fmt.Sprintf("From: \"Business Tracker\" <%s>\r\n"+
		"To: %s\r\n"+
		"Subject: %s\r\n"+
		"Date: %s\r\n"+
		"Message-ID: <%d@atbhdx.com>\r\n"+
		"MIME-Version: 1.0\r\n"+
		"Content-Type: text/html; charset=UTF-8\r\n"+
		"\r\n"+
		"%s", fromEmail, toEmail, subject, time.Now().Format(time.RFC1123Z), time.Now().UnixNano(), body)

	addr := smtpHost + ":" + smtpPort

	err := smtp.SendMail(addr, auth, fromEmail, []string{toEmail}, []byte(message))
	if err != nil {
		return err
	}

	return nil
}

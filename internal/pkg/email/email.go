package email

import (
	"fmt"
	"log/slog"

	gomail "gopkg.in/mail.v2"
)

// Sender — email илгээх interface. Test-д mock хийхэд ашигладаг.
type Sender interface {
	SendPasswordReset(to, resetLink string) error
}

// SMTPSender нь gopkg.in/mail.v2 ашиглан SMTP-р илгээнэ.
type SMTPSender struct {
	host     string
	port     int
	username string
	password string
	from     string
}

func NewSMTPSender(host string, port int, username, password, from string) *SMTPSender {
	return &SMTPSender{
		host:     host,
		port:     port,
		username: username,
		password: password,
		from:     from,
	}
}

func (s *SMTPSender) SendPasswordReset(to, resetLink string) error {
	m := gomail.NewMessage()
	m.SetHeader("From", s.from)
	m.SetHeader("To", to)
	m.SetHeader("Subject", "Нууц үг сэргээх хүсэлт")
	m.SetBody("text/plain", buildPlainText(resetLink))
	m.AddAlternative("text/html", buildHTML(resetLink))

	d := gomail.NewDialer(s.host, s.port, s.username, s.password)

	if err := d.DialAndSend(m); err != nil {
		return fmt.Errorf("send email: %w", err)
	}
	slog.Info("password reset email sent", "to", to)
	return nil
}

// NoopSender нь email тохиргоо байхгүй үед ашиглагдана.
// Token-г зөвхөн log-д хэвлэнэ (development орчин).
type NoopSender struct{}

func (NoopSender) SendPasswordReset(to, _ string) error {
	slog.Warn("EMAIL NOT CONFIGURED - password reset link was not sent",
		"to", to,
	)
	return nil
}

func buildPlainText(link string) string {
	return fmt.Sprintf(`Сайн байна уу,

Таны бүртгэлтэй имэйл хаягаар нууц үг сэргээх хүсэлт ирлээ.

Дараах линкийг дарж нууц үгээ шинэчлэнэ үү:
%s

Линк 1 цагийн дотор хүчинтэй байна.

Хэрэв та энэ хүсэлт гаргаагүй бол энэ имэйлийг үл тоомсорлоно уу.

Хүндэтгэлтэй,
Систем`, link)
}

func buildHTML(link string) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="mn">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>Нууц үг сэргээх</title>
  <style>
    body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif;
           background: #f5f5f5; margin: 0; padding: 20px; }
    .container { max-width: 520px; margin: 40px auto; background: #fff;
                 border-radius: 8px; overflow: hidden;
                 box-shadow: 0 2px 8px rgba(0,0,0,.1); }
    .header { background: #1a1a2e; padding: 32px; text-align: center; }
    .header h1 { color: #fff; margin: 0; font-size: 22px; font-weight: 600; }
    .body { padding: 40px 32px; }
    .body p { color: #444; line-height: 1.7; margin: 0 0 20px; }
    .btn { display: inline-block; background: #4f46e5; color: #fff !important;
           text-decoration: none; padding: 14px 32px; border-radius: 6px;
           font-weight: 600; font-size: 15px; margin: 8px 0 24px; }
    .note { font-size: 13px; color: #888; }
    .footer { background: #f9f9f9; padding: 20px 32px;
              text-align: center; font-size: 12px; color: #aaa; }
  </style>
</head>
<body>
  <div class="container">
    <div class="header">
      <h1>🔐 Нууц үг сэргээх</h1>
    </div>
    <div class="body">
      <p>Сайн байна уу,</p>
      <p>Таны бүртгэлтэй имэйл хаягаар <strong>нууц үг сэргээх хүсэлт</strong> ирлээ.</p>
      <p>Дараах товчлуурыг дарж нууц үгээ шинэчлэнэ үү:</p>
      <a href="%s" class="btn">Нууц үг шинэчлэх</a>
      <p class="note">
        ⏱ Линк <strong>1 цагийн</strong> дотор хүчинтэй байна.<br>
        Хэрэв та энэ хүсэлт гаргаагүй бол энэ имэйлийг үл тоомсорлоно уу.
      </p>
    </div>
    <div class="footer">
      Линк ажиллахгүй бол дараах хаягийг хуулж хөтөч дээр нээнэ үү:<br>
      <small>%s</small>
    </div>
  </div>
</body>
</html>`, link, link)
}

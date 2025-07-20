package email

import (
	"context"
	"fmt"

	"gopkg.in/gomail.v2"

	"delayed-notifier/internal/config"
	"delayed-notifier/internal/entity"
)

type Mailer struct {
	dialer *gomail.Dialer
	from   string
}

func NewMailer(cfg config.MailConfig) *Mailer {
	return &Mailer{
		dialer: gomail.NewDialer(cfg.Host, cfg.Port, cfg.User, cfg.Password),
		from:   cfg.User,
	}
}

func (s *Mailer) Send(ctx context.Context, notify entity.Notify) error {
	to := notify.Email
	if to == "" {
		return fmt.Errorf("email not found in notify")
	}

	m := gomail.NewMessage()
	m.SetHeader("From", s.from)
	m.SetHeader("To", to)
	m.SetHeader("Subject", "Уведомление")

	body := fmt.Sprintf(`
		<html>
		  <body style="font-family: Arial, sans-serif; line-height: 1.6;">
			<h2 style="color: #2c3e50;">Уведомление</h2>
			<p><strong>Сообщение:</strong> %s</p>
			<p><strong>Дата отправки:</strong> %s</p>
			<hr/>
			<p style="font-size: 12px; color: #999;">ID уведомления: %s</p>
		  </body>
		</html>`,
		notify.Message,
		notify.SendAt.Format("02.01.2006 15:04:05"),
		notify.ID,
	)

	m.SetBody("text/html", body)

	return s.dialer.DialAndSend(m)
}

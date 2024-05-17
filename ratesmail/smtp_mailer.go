package ratesmail

import (
	"context"
	"log"

	mail "github.com/wneessen/go-mail"
)

type SmtpMailer struct {
	client *mail.Client
	from   string
}

func NewSmtpMailer(config EmailConfig) (mailer SmtpMailer, err error) {
	mailer.from = config.From
	mailer.client, err = mail.NewClient(
		config.SMTP.Host,
		mail.WithPort(config.SMTP.Port),
		mail.WithSMTPAuth(mail.SMTPAuthPlain),
		mail.WithUsername(config.Username),
		mail.WithPassword(config.Password),
	)
	if err != nil {
		return
	}

	if config.SMTP.SSL {
		err = mail.WithSSLPort(false)(mailer.client)
		if err != nil {
			return
		}
	}

	return
}

func (m SmtpMailer) Send(ctx context.Context, to []string, subject string, body string) error {
	messages := make([]*mail.Msg, 0, len(to))

	for _, recipient := range to {
		msg := mail.NewMsg()
		err := msg.From(m.from)
		if err != nil {
			return err
		}
		msg.Subject(subject)
		msg.SetBodyString(mail.TypeTextPlain, body)
		err = msg.To(recipient)
		if err != nil {
            log.Printf("Invalid email address: %s", recipient)
			continue
		}

		messages = append(messages, msg)
	}

	return m.client.DialAndSendWithContext(ctx, messages...)
}

func (m SmtpMailer) Close() error {
    return m.client.Close()
}



package ratesmail

import (
	"context"
	"database/sql"
	"fmt"
	"log"
)

type Mailer interface {
	Send(ctx context.Context, to []string, subject string, body string) error
	Close() error
}

type PublishMailingListOpts struct {
	*sql.DB
	RateFetcher
	Mailer
	ExchangeRateConfig
}

func PublishMailingList(opts PublishMailingListOpts) error {
	db := opts.DB

	rate, err := opts.RateFetcher(
		context.Background(),
		opts.ExchangeRateConfig.From,
		opts.ExchangeRateConfig.To,
	)

	rows, err := db.Query("SELECT email FROM subscribers")
	if err != nil {
		return err
	}
	defer rows.Close()

	emails := make([]string, 0)
	for rows.Next() {
		var email string
		err = rows.Scan(&email)
		if err != nil {
			return err
		}
		emails = append(emails, email)
	}

	body := fmt.Sprintf("The toda's exchange rate is %f", rate)
	subject := fmt.Sprintf("Your daily %s to %s exchange rate newsletter", opts.ExchangeRateConfig.From, opts.ExchangeRateConfig.To)
	err = opts.Mailer.Send(context.Background(), emails, subject, body)
	if err != nil {
		return err
	}

	log.Printf("Sent %d emails", len(emails))

	return nil
}

package main

import "github.com/malien/usd-rates-emailer/ratesmail"

func must(err error) {
	if err != nil {
		panic(err)
	}
}

func main() {
	config, err := ratesmail.ParseConfig()
	must(err)

	mailer, err := ratesmail.NewSmtpMailer(config.Email)
	must(err)
	defer mailer.Close()

	db, err := ratesmail.OpenDB(config.DB)
	must(err)
	defer db.Close()

	err = ratesmail.PublishMailingList(ratesmail.PublishMailingListOpts{
		DB:                 db,
		RateFetcher:        ratesmail.FetchExchangeRateAPIOpenRates,
		Mailer:             mailer,
		ExchangeRateConfig: config.ExchangeRates,
	})
	must(err)
}

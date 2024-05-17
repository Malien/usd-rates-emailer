package main

import (
	"github.com/malien/usd-rates-emailer/ratesmail"
	_ "github.com/mattn/go-sqlite3"
)

func must(err error) {
	if err != nil {
		panic(err)
	}
}

func main() {
	config, err := ratesmail.ParseConfig()
	must(err)

	db, err := ratesmail.OpenDB(config.DB)
	must(err)
	defer db.Close()

	err = ratesmail.RunServer(ratesmail.RunServerOptions{
		DB:                 db,
		RateFetcher:        ratesmail.FetchExchangeRateAPIOpenRates,
		Config:             config.Server,
		ExchangeRateConfig: config.ExchangeRates,
	})
	must(err)
}

package main

import (
	"log"

	"github.com/malien/usd-rates-emailer/common"
	"github.com/malien/usd-rates-emailer/subserver"

	_ "github.com/mattn/go-sqlite3"
)

func must(err error) {
	if err != nil {
		panic(err)
	}
}

func main() {
    config, err := common.ParseConfig()
    log.Printf("Loaded config: %+v", config)

    db, err := common.OpenDB(config.DB)
	must(err)

	err = subserver.Bootstrap(subserver.BootstrapOptions{
		DB:          db,
		RateFetcher: subserver.FetchExchangeRateAPIOpenRates,
        Config:      config.Server,
	})
	must(err)
}

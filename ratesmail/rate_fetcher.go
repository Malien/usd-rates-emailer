package ratesmail

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

type RateFetcher func(context.Context, string, string) (float64, error)

type exchangeRateAPIResponse struct {
	Rates map[string]float64 `json:"rates"`
}

func FetchExchangeRateAPIOpenRates(ctx context.Context, from string, to string) (float64, error) {
	requestId := requestId(ctx)

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://open.er-api.com/v6/latest/"+from, nil)
	if err != nil {
		return 0, err
	}
	request.Header.Set("Accept", "application/json")
    if requestId != "" {
        log.Printf("[request=%s] Fetching rate via %s", requestId, request.URL.String())
    } else {
        log.Printf("Fetching rate via %s", request.URL.String())
    }
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return 0, err
	}
	defer response.Body.Close()

	var apiResponse exchangeRateAPIResponse
	err = json.NewDecoder(response.Body).Decode(&apiResponse)
	if err != nil {
		return 0, err
	}

	rate, ok := apiResponse.Rates[to]
	if !ok {
		return 0, fmt.Errorf("Conversion rate from %s to %s not found", from, to)
	}

    if requestId != "" {
        log.Printf("[request=%s] Rate fetched from %s to %s: %f", requestId, from, to, rate)
    } else {
        log.Printf("Rate fetched from %s to %s: %f", from, to, rate)
    }
	return rate, nil
}


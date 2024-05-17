package usdratesemailer_test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net"
	"net/http"
	"net/url"
	"testing"

	"github.com/malien/usd-rates-emailer/ratesmail"
)

func must(t *testing.T, err error) {
	if err != nil {
		t.Fatal(err)
	}
}

type testState struct {
	db       *sql.DB
	listener net.Listener
}

func (ts *testState) close() {
	ts.db.Close()
	ts.listener.Close()
}

func prepare(t *testing.T, rateFetcher ratesmail.RateFetcher) testState {
	dbConfig := ratesmail.DBConfig{
		Filename: ":memory:",
		WALMode:  false,
	}
	db, err := ratesmail.OpenDB(dbConfig)
	must(t, err)

	handler := ratesmail.Bootstrap(ratesmail.BootstrapOptions{
		DB:          db,
		RateFetcher: rateFetcher,
		ExchangeRateConfig: ratesmail.ExchangeRateConfig{
			From: "USD",
			To:   "EUR",
		},
	})

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	must(t, err)

	go http.Serve(listener, handler)

	return testState{
		db:       db,
		listener: listener,
	}
}

func (s *testState) url(path string) string {
	return "http://" + s.listener.Addr().String() + path
}

func dummyRates(ctx context.Context, from string, to string) (float64, error) {
	return 6.9, nil
}

func TestReturnsRatesFromAPIResponse(t *testing.T) {
	state := prepare(t, dummyRates)
	defer state.close()

	response, err := http.Get(state.url("/rate"))
	must(t, err)

	if response.StatusCode != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", response.StatusCode)
	}

	var num float64
	err = json.NewDecoder(response.Body).Decode(&num)
	must(t, err)

	if num != 6.9 {
		t.Fatalf("Expected 6.9, got %f", num)
	}
}

func addSubscriber(t *testing.T, state testState, email string) *http.Response {
	body := bytes.NewBuffer([]byte("email=" + url.PathEscape(email)))
	request, err := http.NewRequest(http.MethodPost, state.url("/subscribe"), body)
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	must(t, err)

	response, err := http.DefaultClient.Do(request)
	must(t, err)

	return response
}

func TestUniqueEmailsOnly(t *testing.T) {
	state := prepare(t, dummyRates)
	defer state.close()

	response := addSubscriber(t, state, "foo@mail.com")
	if response.StatusCode != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", response.StatusCode)
	}

	response = addSubscriber(t, state, "foo@mail.com")
	if response.StatusCode != http.StatusConflict {
		t.Fatalf("Expected status 409, got %d", response.StatusCode)
	}

	response = addSubscriber(t, state, "bar@mail.com")
	if response.StatusCode != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", response.StatusCode)
	}
}

type dummyMessage struct {
    to      string
    subject string
    body    string
}

type dummyMailer struct {
    messages []dummyMessage
}

func (m *dummyMailer) Send(ctx context.Context, to []string, subject string, body string) error {
    for _, recipient := range to {
        m.messages = append(m.messages, dummyMessage{recipient, subject, body})
    }
    return nil
}

func (m *dummyMailer) Close() error {
    return nil
}

func TestSendsEmails(t *testing.T) {
    state := prepare(t, dummyRates)
    defer state.close()

    addSubscriber(t, state, "foo@mail.com")
    addSubscriber(t, state, "bar@mail.com")
    addSubscriber(t, state, "baz@surelymail.gov.uk")

    mailer := &dummyMailer{}
    ratesmail.PublishMailingList(ratesmail.PublishMailingListOpts{
        DB: state.db,
        RateFetcher: dummyRates,
        Mailer: mailer,
        ExchangeRateConfig: ratesmail.ExchangeRateConfig{
            From: "USD",
            To: "EUR",
        },
    })

    expected := [3]dummyMessage{
        {"foo@mail.com", "Your daily USD to EUR exchange rate newsletter", "The today's exchange rate is 6.90"},
        {"bar@mail.com", "Your daily USD to EUR exchange rate newsletter", "The today's exchange rate is 6.90"},
        {"baz@surelymail.gov.uk", "Your daily USD to EUR exchange rate newsletter", "The today's exchange rate is 6.90"},
    }

    if len(mailer.messages) != 3 {
        t.Fatalf("Expected 3 messages, got %d", len(mailer.messages))
    }

    for i, msg := range mailer.messages {
        if msg != expected[i] {
            t.Fatalf("Expected %v, got %v", expected[i], msg)
        }
    }
}

package ratesmail

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/mail"
	"time"

	nanoid "github.com/matoous/go-nanoid/v2"
	_ "github.com/mattn/go-sqlite3"
)

type BootstrapOptions struct {
	*sql.DB
	RateFetcher
	Config ServerConfig
	ExchangeRateConfig
}

func Bootstrap(opts BootstrapOptions) error {
	db := opts.DB

	mux := http.NewServeMux()
	handleTracedFunc(mux, "/health", func(w http.ResponseWriter, r *http.Request) error {
		w.WriteHeader(http.StatusOK)
		return nil
	})
	handleTracedFunc(mux, "/subscribe", func(w http.ResponseWriter, r *http.Request) error {
		return insertSubscriber(db, w, r)
	})
	handleTracedFunc(mux, "/rate", func(w http.ResponseWriter, r *http.Request) error {
		return fetchRates(opts.RateFetcher, opts.ExchangeRateConfig, w, r)
	})

	sock, err := net.Listen("tcp", fmt.Sprintf("%s:%d", opts.Config.BindAddress, opts.Config.Port))
	if err != nil {
		return err
	}
	log.Printf("Listening on %s", sock.Addr())
	return http.Serve(sock, mux)
}

type SimpleResponseBody struct {
	Ok      bool   `json:"ok"`
	Message string `json:"message"`
}

func okResponse(w http.ResponseWriter, message string) error {
	return json.NewEncoder(w).Encode(SimpleResponseBody{Ok: true, Message: message})
}
func errorResponse(w http.ResponseWriter, message string) error {
	return json.NewEncoder(w).Encode(SimpleResponseBody{Ok: false, Message: message})
}

func insertSubscriber(db *sql.DB, w http.ResponseWriter, r *http.Request) error {
	requestId := requestId(r.Context())
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return fmt.Errorf("Method not allowed")
	}

	email := r.FormValue("email")
	if email == "" {
		w.WriteHeader(http.StatusBadRequest)
		return fmt.Errorf("Missing email parameter")
	}

	// Arbitary limit of 512 characters to prevent abuse
	if len(email) > 512 {
		w.WriteHeader(http.StatusBadRequest)
		return fmt.Errorf("Email too long")
	}

	_, err := mail.ParseAddress(email)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return fmt.Errorf("Invalid email address")
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}

	var exists bool
	err = tx.QueryRow("select exists(select 1 from subscribers where email = ?)", email).Scan(&exists)
	if err != nil {
		return fmt.Errorf("Error checking subscriber existence: %w", err)
	}

	if exists {
		w.WriteHeader(http.StatusConflict)
		log.Printf("[request=%s] Subscriber %s already exists", requestId, email)
		return fmt.Errorf("Subscriber already exists")
	}

	_, err = tx.Exec("insert into subscribers (email) values (?)", email)
	if err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("Error inserting subscriber: %w", err)
	}
	err = tx.Commit()
	if err != nil {
		return err
	}

	log.Printf("[request=%s] Subscriber %s inserted", requestId, email)
	w.WriteHeader(http.StatusOK)
	must(okResponse(w, "Subscriber added"))
	return nil
}

func fetchRates(fetchRates RateFetcher, conf ExchangeRateConfig, w http.ResponseWriter, r *http.Request) error {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return fmt.Errorf("Method not allowed")
	}

	rate, err := fetchRates(r.Context(), conf.From, conf.To)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return fmt.Errorf("Error fetching rate: %w", err)
	}

	w.WriteHeader(http.StatusOK)
	return json.NewEncoder(w).Encode(rate)
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}

type requestIdKey struct{}

var requestIdContextKey = requestIdKey{}

func requestId(ctx context.Context) string {
    value, _ := ctx.Value(requestIdContextKey).(string)
    return value
}

type statusTracedResponseWriter struct {
	http.ResponseWriter
	statusCode  int
	headersSent bool
}

func (w *statusTracedResponseWriter) WriteHeader(statusCode int) {
	w.statusCode = statusCode
	w.headersSent = true
	w.ResponseWriter.WriteHeader(statusCode)
}

func handleTracedFunc(mux *http.ServeMux, pattern string, handler func(http.ResponseWriter, *http.Request) error) {
	mux.HandleFunc(pattern, func(w http.ResponseWriter, r *http.Request) {
		requestId := nanoid.Must(10)
		startTime := time.Now()
		ctx := context.WithValue(r.Context(), requestIdContextKey, requestId)

		log.Printf("[request=%s] Request received: %s %s", requestId, r.Method, r.URL.Path)
		req := r.WithContext(ctx)
		wtr := &statusTracedResponseWriter{w, http.StatusOK, false}
		err := handler(wtr, req)
		if err != nil {
			log.Printf("[request=%s] Error handling request: %s", requestId, err)
			if !wtr.headersSent {
				wtr.WriteHeader(http.StatusInternalServerError)
				contentType := wtr.Header().Get("Content-Type")
				if contentType == "" {
					wtr.Header().Set("Content-Type", "application/json")
				}
			}
			err = errorResponse(wtr, err.Error())
			if err != nil {
				log.Printf("[request=%s] Error writing error response: %s", requestId, err)
			}
		}
		log.Printf("[request=%s] Request handled: %d %s. Took %s", requestId, wtr.statusCode, http.StatusText(wtr.statusCode), time.Since(startTime))
	})
}

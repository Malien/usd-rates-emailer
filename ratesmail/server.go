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
	ExchangeRateConfig
}

func Bootstrap(opts BootstrapOptions) http.Handler {
	db := opts.DB

	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handleTracedFunc(mux, "/subscribe", func(w http.ResponseWriter, r *http.Request) {
		insertSubscriber(db, w, r)
	})
	handleTracedFunc(mux, "/rate", func(w http.ResponseWriter, r *http.Request) {
		fetchRates(opts.RateFetcher, opts.ExchangeRateConfig, w, r)
	})

	return mux
}

type RunServerOptions struct {
	*sql.DB
	RateFetcher
	ExchangeRateConfig
	Config ServerConfig
}

func RunServer(opts RunServerOptions) error {
	handler := Bootstrap(BootstrapOptions{
		DB:                 opts.DB,
		RateFetcher:        opts.RateFetcher,
		ExchangeRateConfig: opts.ExchangeRateConfig,
	})

	sock, err := net.Listen("tcp", fmt.Sprintf("%s:%d", opts.Config.BindAddress, opts.Config.Port))
	if err != nil {
		return err
	}
	log.Printf("Listening on %s", sock.Addr())
	return http.Serve(sock, handler)
}

type SimpleResponseBody struct {
	Ok      bool   `json:"ok"`
	Message string `json:"message"`
}

func writeOk(w http.ResponseWriter, message string) {
	err := json.NewEncoder(w).Encode(SimpleResponseBody{Ok: true, Message: message})
	if err != nil {
		log.Printf("Error writing response: %s", err)
	}
}
func writeError(w http.ResponseWriter, message string) {
	err := json.NewEncoder(w).Encode(SimpleResponseBody{Ok: false, Message: message})
	if err != nil {
		log.Printf("Error writing response: %s", err)
	}
}

func insertSubscriber(db *sql.DB, w http.ResponseWriter, r *http.Request) {
	requestId := requestId(r.Context())
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		writeError(w, "Method not allowed")
		return
	}

	email := r.FormValue("email")
	if email == "" {
		w.WriteHeader(http.StatusBadRequest)
		log.Printf("[request=%s] Email not provided", requestId)
		writeError(w, "Email is required")
		return
	}

	// Arbitary limit of 512 characters to prevent abuse
	if len(email) > 512 {
		w.WriteHeader(http.StatusBadRequest)
		log.Printf("[request=%s] Email is too long %d", requestId, len(email))
		writeError(w, "Email too long")
		return
	}

	_, err := mail.ParseAddress(email)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		log.Printf("[request=%s] Invalid email address: %s", requestId, email)
		writeError(w, "Invalid email address")
		return
	}

	tx, err := db.Begin()
    defer tx.Rollback()
	if err != nil {
		log.Printf("[request=%s] Error starting transaction: %s", requestId, err)
		w.WriteHeader(http.StatusInternalServerError)
		writeError(w, "Internal server error")
		return
	}

	var exists bool
	err = tx.QueryRow("select exists(select 1 from subscribers where email = ?)", email).Scan(&exists)
	if err != nil {
		log.Printf("[request=%s] Error checking subscriber existence: %s", requestId, err)
		w.WriteHeader(http.StatusInternalServerError)
		writeError(w, "Internal server error")
		return
	}

	if exists {
		w.WriteHeader(http.StatusConflict)
		log.Printf("[request=%s] Subscriber %s already exists", requestId, email)
		writeError(w, "Subscriber already exists")
		return
	}

    _, err = tx.Exec("insert into subscribers (email) values (?)", email)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Printf("[request=%s] Error inserting subscriber: %s", requestId, err)
		writeError(w, "Internal server error")
	}
	err = tx.Commit()
	if err != nil {
		log.Printf("[request=%s] Error committing transaction: %s", requestId, err)
		w.WriteHeader(http.StatusInternalServerError)
		writeError(w, "Internal server error")
		return
	}

	log.Printf("[request=%s] Subscriber %s inserted", requestId, email)
	w.WriteHeader(http.StatusOK)
	writeOk(w, "Subscriber added")
}

func fetchRates(
	fetchRates RateFetcher,
	conf ExchangeRateConfig,
	w http.ResponseWriter,
	r *http.Request,
) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		writeError(w, "Method not allowed")
		return
	}

	rate, err := fetchRates(r.Context(), conf.From, conf.To)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Printf("Error fetching rate: %s", err)
		writeError(w, "Internal server error")
	}

	w.WriteHeader(http.StatusOK)
	err = json.NewEncoder(w).Encode(rate)
	if err != nil {
		log.Printf("Error writing response: %s", err)
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

func handleTracedFunc(mux *http.ServeMux, pattern string, handler func(http.ResponseWriter, *http.Request)) {
	mux.HandleFunc(pattern, func(w http.ResponseWriter, r *http.Request) {
		requestId := nanoid.Must(10)
		startTime := time.Now()
		ctx := context.WithValue(r.Context(), requestIdContextKey, requestId)

		log.Printf("[request=%s] Request received: %s %s", requestId, r.Method, r.URL.Path)
		req := r.WithContext(ctx)
		wtr := &statusTracedResponseWriter{w, http.StatusOK, false}
		handler(wtr, req)
		log.Printf("[request=%s] Request handled: %d %s. Took %s", requestId, wtr.statusCode, http.StatusText(wtr.statusCode), time.Since(startTime))
	})
}

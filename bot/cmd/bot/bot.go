package main

import (
	"context"
	"log"
	"net/http"
	"os/signal"
	"syscall"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	tele "gopkg.in/telebot.v4"
)

func main() {
	cfg, err := loadConfig()
	if err != nil {
		log.Fatalf("invalid configuration: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	migConn, err := pgx.Connect(ctx, cfg.DBURL)
	if err != nil {
		log.Fatalf("connect for migrations failed: %v", err)
	}
	if err := runMigrations(ctx, migConn); err != nil {
		log.Fatalf("run migrations failed: %v", err)
	}
	if err := migConn.Close(ctx); err != nil {
		log.Printf("close migration connection failed: err=%v", err)
	}

	pool, err := pgxpool.New(ctx, cfg.DBURL)
	if err != nil {
		log.Fatalf("create db pool failed: %v", err)
	}
	defer pool.Close()

	b, err := tele.NewBot(tele.Settings{
		Token:  cfg.BotToken,
		Poller: &tele.LongPoller{Timeout: longPollerTimeout},
	})
	if err != nil {
		log.Fatalf("create bot failed: %v", err)
	}

	registerHandlers(b, pool)

	go b.Start()

	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("/remind", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if err := runReminderPass(r.Context(), b, pool); err != nil {
			log.Printf("reminder pass failed: err=%v", err)
			http.Error(w, "Reminder pass failed", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	})

	srv := &http.Server{
		Addr:    cfg.ListenAddr,
		Handler: mux,
	}

	go func() {
		log.Printf("http server listening: addr=%s", cfg.ListenAddr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("http server error: err=%v", err)
		}
	}()

	log.Println("bot is running")

	<-ctx.Done()
	log.Println("shutting down")

	b.Stop()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), httpShutdownTimeout)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("http server shutdown failed: err=%v", err)
	}
}

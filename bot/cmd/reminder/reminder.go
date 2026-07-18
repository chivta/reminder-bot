package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"
)

// requestTimeout bounds the HTTP call to the bot's /remind endpoint. The bot
// handles the reminder pass synchronously and paces outgoing messages, so a
// large backlog can take minutes; a short timeout here would abort the request
// and make the CronJob retry, duplicating reminders.
const requestTimeout = 5 * time.Minute

func main() {
	remindURL := os.Getenv("REMIND_URL")
	if remindURL == "" {
		log.Fatal("REMIND_URL is required")
	}

	ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, remindURL, nil)
	if err != nil {
		log.Fatalf("build request failed: err=%v", err)
	}

	client := &http.Client{Timeout: requestTimeout}
	resp, err := client.Do(req)
	if err != nil {
		log.Fatalf("remind request failed: url=%s err=%v", remindURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		log.Fatalf("remind request returned non-2xx status: url=%s status=%d", remindURL, resp.StatusCode)
	}

	log.Printf("reminder triggered successfully: url=%s status=%d", remindURL, resp.StatusCode)
}

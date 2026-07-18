package main

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	tele "gopkg.in/telebot.v4"
)

// pendingItem is a single unacknowledged saved message.
type pendingItem struct {
	ID        int64
	UserID    int64
	ChatID    int64
	MessageID int
}

// fetchPending loads all unacknowledged messages ordered by user and save time.
func fetchPending(ctx context.Context, pool *pgxpool.Pool) ([]pendingItem, error) {
	rows, err := pool.Query(ctx,
		`SELECT id, user_id, chat_id, message_id
		 FROM messages
		 WHERE acknowledged_at IS NULL
		 ORDER BY user_id, saved_at ASC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []pendingItem
	for rows.Next() {
		var it pendingItem
		if err := rows.Scan(&it.ID, &it.UserID, &it.ChatID, &it.MessageID); err != nil {
			return nil, err
		}
		items = append(items, it)
	}
	return items, rows.Err()
}

// groupByUser buckets pending items by user, preserving saved_at order.
func groupByUser(items []pendingItem) map[int64][]pendingItem {
	grouped := make(map[int64][]pendingItem)
	for _, it := range items {
		grouped[it.UserID] = append(grouped[it.UserID], it)
	}
	return grouped
}

// runReminderPass sends every user their pending items and re-attaches a
// fresh ✅ Done keyboard to each. Errors for one user never abort others.
func runReminderPass(ctx context.Context, b *tele.Bot, pool *pgxpool.Pool) error {
	items, err := fetchPending(ctx, pool)
	if err != nil {
		return fmt.Errorf("fetch pending: %w", err)
	}

	btnAck := tele.Btn{Unique: ackButtonUnique}

	for userID, userItems := range groupByUser(items) {
		recipient := &tele.User{ID: userID}

		header := fmt.Sprintf("🔔 You have %d unread saved item(s):", len(userItems))
		if _, err := b.Send(recipient, header); err != nil {
			log.Printf("remind header failed: user_id=%d err=%v", userID, err)
			continue
		}
		time.Sleep(remindSendDelay)

		for _, it := range userItems {
			markup := &tele.ReplyMarkup{}
			btn := markup.Data(ackButtonText, btnAck.Unique, strconv.FormatInt(it.ID, 10))
			markup.Inline(markup.Row(btn))

			stored := &tele.StoredMessage{
				MessageID: strconv.Itoa(it.MessageID),
				ChatID:    it.ChatID,
			}
			if _, err := b.Copy(recipient, stored, markup); err != nil {
				log.Printf("remind copy failed: id=%d user_id=%d chat_id=%d message_id=%d err=%v",
					it.ID, userID, it.ChatID, it.MessageID, err)
				continue
			}
			time.Sleep(remindSendDelay)
		}
	}

	return nil
}

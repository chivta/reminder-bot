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

// fetchDue returns, for each user, their single oldest unacknowledged message
// that was saved before cutoff. Messages newer than cutoff are left for a
// later pass.
func fetchDue(ctx context.Context, pool *pgxpool.Pool, cutoff time.Time) ([]pendingItem, error) {
	rows, err := pool.Query(ctx,
		`SELECT DISTINCT ON (user_id) id, user_id, chat_id, message_id
		 FROM messages
		 WHERE acknowledged_at IS NULL AND saved_at <= $1
		 ORDER BY user_id, saved_at ASC`,
		cutoff,
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

// runReminderPass sends each user their oldest unacknowledged message that is
// at least minRemindAge old, with a fresh ✅ Done keyboard attached. One
// message per user per pass; errors for one user never abort others.
func runReminderPass(ctx context.Context, b *tele.Bot, pool *pgxpool.Pool) error {
	items, err := fetchDue(ctx, pool, time.Now().Add(-minRemindAge))
	if err != nil {
		return fmt.Errorf("fetch due messages: %w", err)
	}

	btnAck := tele.Btn{Unique: ackButtonUnique}

	for _, it := range items {
		markup := &tele.ReplyMarkup{}
		btn := markup.Data(ackButtonText, btnAck.Unique, strconv.FormatInt(it.ID, 10))
		markup.Inline(markup.Row(btn))

		stored := &tele.StoredMessage{
			MessageID: strconv.Itoa(it.MessageID),
			ChatID:    it.ChatID,
		}
		if _, err := b.Copy(&tele.User{ID: it.UserID}, stored, markup); err != nil {
			log.Printf("remind copy failed: id=%d user_id=%d chat_id=%d message_id=%d err=%v",
				it.ID, it.UserID, it.ChatID, it.MessageID, err)
			continue
		}
		time.Sleep(remindSendDelay)
	}

	return nil
}

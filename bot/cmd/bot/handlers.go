package main

import (
	"context"
	"log"
	"strconv"

	"github.com/jackc/pgx/v5/pgxpool"
	tele "gopkg.in/telebot.v4"
)

const (
	introMessage = "Send me anything you want to read later — a link, an article, a message, " +
		"a photo, whatever. I'll save it and remind you every morning and evening until " +
		"you mark it done with the ✅ Done button."
	savedMessage  = "Saved ⏰ I'll remind you until you mark it done."
	genericError  = "Something went wrong. Please try again."
	ackDoneText   = "Done ✅"
	ackButtonText = "✅ Done"
)

// registerUser upserts the sender into the users table so saving and
// reminding works even if they never sent /start.
func registerUser(ctx context.Context, pool *pgxpool.Pool, userID int64) error {
	qctx, cancel := context.WithTimeout(ctx, dbQueryTimeout)
	defer cancel()

	_, err := pool.Exec(qctx, `INSERT INTO users (id) VALUES ($1) ON CONFLICT DO NOTHING`, userID)
	return err
}

// registerHandlers wires all telebot handlers onto b.
func registerHandlers(b *tele.Bot, pool *pgxpool.Pool) {
	btnAck := tele.Btn{Unique: ackButtonUnique}

	ackMarkup := func(id int64) *tele.ReplyMarkup {
		markup := &tele.ReplyMarkup{}
		btn := markup.Data(ackButtonText, btnAck.Unique, strconv.FormatInt(id, 10))
		markup.Inline(markup.Row(btn))
		return markup
	}

	b.Handle("/start", func(c tele.Context) error {
		if err := registerUser(context.Background(), pool, c.Sender().ID); err != nil {
			log.Printf("register user failed: user_id=%d err=%v", c.Sender().ID, err)
			return c.Send(genericError)
		}
		return c.Send(introMessage)
	})

	b.Handle("/pending", func(c tele.Context) error {
		ctx, cancel := context.WithTimeout(context.Background(), dbQueryTimeout)
		defer cancel()

		var count int
		err := pool.QueryRow(ctx,
			`SELECT COUNT(*) FROM messages WHERE user_id = $1 AND acknowledged_at IS NULL`,
			c.Sender().ID,
		).Scan(&count)
		if err != nil {
			log.Printf("count pending failed: user_id=%d err=%v", c.Sender().ID, err)
			return c.Send(genericError)
		}
		return c.Send("You have " + strconv.Itoa(count) + " pending item(s).")
	})

	saveMessage := func(c tele.Context) error {
		sender := c.Sender().ID
		if err := registerUser(context.Background(), pool, sender); err != nil {
			log.Printf("register user failed: user_id=%d err=%v", sender, err)
			return c.Send(genericError)
		}

		ctx, cancel := context.WithTimeout(context.Background(), dbQueryTimeout)
		defer cancel()

		var id int64
		err := pool.QueryRow(ctx,
			`INSERT INTO messages (user_id, chat_id, message_id)
			 VALUES ($1, $2, $3)
			 ON CONFLICT (chat_id, message_id) DO NOTHING
			 RETURNING id`,
			sender, c.Chat().ID, c.Message().ID,
		).Scan(&id)
		if err != nil {
			// A conflict on (chat_id, message_id) yields no row from RETURNING;
			// look up the existing row's id instead of failing the save.
			lookupErr := pool.QueryRow(ctx,
				`SELECT id FROM messages WHERE chat_id = $1 AND message_id = $2`,
				c.Chat().ID, c.Message().ID,
			).Scan(&id)
			if lookupErr != nil {
				log.Printf("save message failed: user_id=%d chat_id=%d message_id=%d err=%v",
					sender, c.Chat().ID, c.Message().ID, err)
				return c.Send(genericError)
			}
		}

		return c.Send(savedMessage, ackMarkup(id))
	}

	b.Handle(tele.OnText, saveMessage)
	b.Handle(tele.OnMedia, saveMessage)

	b.Handle(&btnAck, func(c tele.Context) error {
		id, err := strconv.ParseInt(c.Data(), 10, 64)
		if err != nil {
			log.Printf("ack parse failed: data=%q err=%v", c.Data(), err)
			return c.Respond(&tele.CallbackResponse{Text: genericError})
		}

		ctx, cancel := context.WithTimeout(context.Background(), dbQueryTimeout)
		defer cancel()

		_, err = pool.Exec(ctx,
			`UPDATE messages SET acknowledged_at = now() WHERE id = $1 AND user_id = $2`,
			id, c.Sender().ID,
		)
		if err != nil {
			log.Printf("ack update failed: id=%d user_id=%d err=%v", id, c.Sender().ID, err)
			return c.Respond(&tele.CallbackResponse{Text: genericError})
		}

		if _, err := c.Bot().EditReplyMarkup(c.Message(), nil); err != nil {
			log.Printf("ack remove keyboard failed: id=%d err=%v", id, err)
		}

		return c.Respond(&tele.CallbackResponse{Text: ackDoneText})
	})
}

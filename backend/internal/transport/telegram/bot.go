package telegram

import (
	"context"
	"log/slog"
	"time"
)

// Bot runs the long-poll loop and dispatches messages to Router.
type Bot struct {
	Client *Client
	Router *Router
	Logger *slog.Logger
	// PollTimeout is the getUpdates long-poll wait.
	PollTimeout time.Duration
}

// Start blocks until ctx is cancelled. Call in a goroutine from main.
func (b *Bot) Start(ctx context.Context) {
	log := b.Logger
	if log == nil {
		log = slog.Default()
	}
	if b.Client == nil || b.Router == nil {
		log.Error("telegram bot not configured")
		return
	}
	pollSec := int(b.PollTimeout.Seconds())
	if pollSec < 1 {
		pollSec = 25
	}

	username, err := b.Client.GetMe(ctx)
	if err != nil {
		log.Error("telegram getMe failed — check TELEGRAM_BOT_TOKEN", "err", err)
		return
	}
	log.Info("telegram bot authorized", "username", username)

	var offset int64
	log.Info("telegram polling for updates", "poll_timeout_sec", pollSec)
	for {
		if ctx.Err() != nil {
			log.Info("telegram bot stopped")
			return
		}
		updates, err := b.Client.GetUpdates(ctx, offset, pollSec)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			log.Error("telegram getUpdates failed", "err", err)
			select {
			case <-ctx.Done():
				return
			case <-time.After(2 * time.Second):
			}
			continue
		}
		for _, u := range updates {
			if u.UpdateID >= offset {
				offset = u.UpdateID + 1
			}
			if u.Message == nil || u.Message.Text == "" {
				continue
			}
			chatID := u.Message.Chat.ID
			userID := chatID
			if u.Message.From != nil {
				userID = u.Message.From.ID
			}
			text := u.Message.Text
			log.Info("telegram command", "chat_id", chatID, "user_id", userID, "text", truncate(text, 80))
			reply := b.Router.Handle(ctx, chatID, userID, text)
			if err := b.Client.SendMessage(ctx, chatID, reply); err != nil {
				log.Warn("telegram HTML send failed, retry plain", "err", err)
				_ = b.Client.SendMessageMode(ctx, chatID, PlainText(reply), "")
			}
		}
	}
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

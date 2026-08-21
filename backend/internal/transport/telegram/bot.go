package telegram

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/service/aiagent"
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
			if u.CallbackQuery != nil {
				b.dispatchCallback(ctx, u.CallbackQuery)
				continue
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
			// AI runs in a goroutine so long /ask turns do not block getUpdates.
			if b.Router.IsAIRequest(text) && b.Router.ai != nil {
				go b.handleAI(ctx, chatID, userID, text)
				continue
			}
			b.dispatch(ctx, chatID, userID, text)
		}
	}
}

func (b *Bot) dispatch(ctx context.Context, chatID, userID int64, text string) {
	log := b.Logger
	if log == nil {
		log = slog.Default()
	}

	reply := b.Router.HandleMessage(ctx, chatID, userID, text)
	markup := encodeInlineKeyboard(reply.Keyboard)
	if _, err := b.Client.SendMessageMarkup(ctx, chatID, reply.Text, "HTML", markup); err != nil {
		log.Warn("telegram HTML send failed, retry plain", "err", err)
		_, _ = b.Client.SendMessageMarkup(ctx, chatID, PlainText(reply.Text), "", markup)
	}
}

func (b *Bot) dispatchCallback(ctx context.Context, cq *CallbackQuery) {
	log := b.Logger
	if log == nil {
		log = slog.Default()
	}
	if cq == nil {
		return
	}
	var chatID int64
	var msgID int64
	if cq.Message != nil {
		chatID = cq.Message.Chat.ID
		msgID = cq.Message.MessageID
	}
	userID := chatID
	if cq.From != nil {
		userID = cq.From.ID
	}
	log.Info("telegram callback", "chat_id", chatID, "user_id", userID, "data", truncate(cq.Data, 40))
	reply := b.Router.HandleCallback(ctx, chatID, userID, cq.Data)
	_ = b.Client.AnswerCallbackQuery(ctx, cq.ID, reply.Toast)
	if msgID == 0 || reply.Text == "" {
		return
	}
	markup := encodeInlineKeyboard(reply.Keyboard)
	if reply.ClearKeyboard || len(reply.Keyboard) == 0 {
		markup = emptyInlineKeyboardJSON
	}
	if err := b.Client.EditMessageMarkup(ctx, chatID, msgID, reply.Text, "HTML", markup); err != nil {
		if err2 := b.Client.EditMessageMarkup(ctx, chatID, msgID, PlainText(reply.Text), "", markup); err2 != nil {
			log.Warn("telegram callback edit failed", "err", err, "plain_err", err2)
		}
	}
}

func (b *Bot) handleAI(ctx context.Context, chatID, userID int64, text string) {
	log := b.Logger
	if log == nil {
		log = slog.Default()
	}
	if !b.Router.Allowed(chatID) {
		_, _ = b.Client.SendMessage(ctx, chatID, "This bot is private. Your chat is not allowed.")
		return
	}
	if !b.Router.allowRate(chatID) {
		_, _ = b.Client.SendMessage(ctx, chatID, "Slow down — try again in a second.")
		return
	}

	q := b.Router.AIQuestion(text)
	if q == "" {
		_, _ = b.Client.SendMessage(ctx, chatID, "Usage: <code>/ask &lt;question&gt;</code>\nExample: <code>/ask What is BTC RSI?</code>")
		return
	}

	// Live progress card (HTML) — edited in place as tools run, then replaced with the answer.
	progressID, err := b.Client.SendMessage(ctx, chatID, FormatAIProgress("Planning…", nil))
	if err != nil {
		log.Warn("telegram progress send failed", "err", err)
	}

	var (
		mu       sync.Mutex
		tools    []string
		seen     = map[string]struct{}{}
		status   = "Planning…"
		lastEdit time.Time
	)
	session, serr := b.Router.clientIDForUser(ctx, userID)
	if serr != nil {
		log.Warn("telegram identity", "err", serr)
		_, _ = b.Client.SendMessage(ctx, chatID, "Could not resolve your account id.")
		return
	}
	aiCtx, cancel := context.WithTimeout(ctx, b.Router.opts.AITimeout)
	defer cancel()

	editProgress := func(force bool) {
		if progressID == 0 {
			return
		}
		mu.Lock()
		st := status
		toolCopy := append([]string(nil), tools...)
		since := time.Since(lastEdit)
		if !force && since < 350*time.Millisecond {
			mu.Unlock()
			return
		}
		lastEdit = time.Now()
		mu.Unlock()

		body := FormatAIProgress(st, toolCopy)
		if err := b.Client.EditMessageText(ctx, chatID, progressID, body, "HTML"); err != nil {
			// Fallback to plain if HTML parse fails for any reason.
			if err2 := b.Client.EditMessageText(ctx, chatID, progressID, PlainText(body), ""); err2 != nil {
				if !strings.Contains(err2.Error(), "not modified") {
					log.Debug("telegram edit progress", "err", err, "plain_err", err2)
				}
			}
		}
	}

	onEvent := func(ev aiagent.StreamEvent) {
		force := false
		mu.Lock()
		switch ev.Type {
		case "ping", "done", "tool_result", "tool_error":
			// Keep-alive + leaf tool noise — do not spam Telegram progress.
			mu.Unlock()
			return
		case "status", "thinking":
			if t := strings.TrimSpace(ev.Text); t != "" && IsMainAIStatus(t) {
				// Shorten "Running market_agent…" → "Running Market…"
				status = shortStatus(t)
			}
		case "tool":
			// Only top-level specialists (market/web/x/analyst), not get_ticker etc.
			if t := strings.TrimSpace(ev.Text); t != "" && IsMainAITool(t) {
				label := ShortMainToolLabel(t)
				if _, ok := seen[label]; !ok {
					seen[label] = struct{}{}
					tools = append(tools, label)
				}
				status = "Running " + label + "…"
				force = true
			}
		case "final":
			status = "Composing answer…"
			force = true
		}
		mu.Unlock()
		editProgress(force)
	}

	res, err := b.Router.ai.ChatStream(aiCtx, q, session, session, telegramAIChatOptions(), onEvent)
	if err != nil {
		log.Warn("telegram AI stream failed", "err", err)
		// Only fall back to non-stream Chat when the stream endpoint is missing
		// (old AI process). Never re-run a full multi-agent turn on timeout —
		// that reuses an expired context and leaves the user stuck on "working…".
		errStr := err.Error()
		isTimeout := aiCtx.Err() != nil ||
			strings.Contains(errStr, "context deadline exceeded") ||
			strings.Contains(errStr, "context canceled")
		isMissingStream := strings.Contains(errStr, "stream error (404)") ||
			strings.Contains(errStr, "not found")

		if !isTimeout && isMissingStream {
			mu.Lock()
			status = "Running (no live stream)…"
			mu.Unlock()
			editProgress(true)
			// Fresh timeout budget for the fallback call.
			fbCtx, fbCancel := context.WithTimeout(ctx, b.Router.opts.AITimeout)
			res, err = b.Router.ai.Chat(fbCtx, q, session, session, telegramAIChatOptions())
			fbCancel()
		}

		if err != nil {
			msg := formatAIFailure(err, isTimeout, b.Router.opts.AITimeout)
			b.deliverAIMessage(ctx, chatID, progressID, msg)
			return
		}
	}

	// Prefer main agents collected live; fill from result (filtered) if stream missed them.
	mu.Lock()
	finalTools := FilterMainAITools(append([]string(nil), tools...))
	if len(finalTools) == 0 {
		finalTools = FilterMainAITools(res.Tools)
	}
	mu.Unlock()

	if strings.TrimSpace(res.Reply) == "" {
		b.deliverAIMessage(ctx, chatID, progressID,
			"AI finished without a text answer. Try a narrower question (e.g. /ask JUVUSDT RSI on binance).")
		return
	}

	final := FormatAIAnswer(res.Reply, res.Thinking, finalTools) + FormatAIReferences(toRefLinks(res.References))
	b.deliverAIMessage(ctx, chatID, progressID, final)
}

func shortStatus(s string) string {
	lower := strings.ToLower(s)
	switch {
	case strings.Contains(lower, "market_agent"):
		return "Running Market…"
	case strings.Contains(lower, "web_agent"):
		return "Running Web…"
	case strings.Contains(lower, "x_agent"):
		return "Running X / social…"
	case strings.Contains(lower, "analyst"):
		return "Running Analyst…"
	case strings.Contains(lower, "planning"):
		return "Planning…"
	case strings.Contains(lower, "compos"):
		return "Composing answer…"
	case strings.Contains(lower, "synthes"):
		return "Synthesizing…"
	default:
		return clipRunes(strings.TrimSpace(s), 60)
	}
}

// deliverAIMessage edits the progress card when possible; always falls back to a new message.
func (b *Bot) deliverAIMessage(ctx context.Context, chatID, progressID int64, html string) {
	log := b.Logger
	if log == nil {
		log = slog.Default()
	}
	if progressID != 0 {
		if err := b.Client.EditMessageText(ctx, chatID, progressID, html, "HTML"); err == nil {
			return
		} else if err2 := b.Client.EditMessageText(ctx, chatID, progressID, PlainText(html), ""); err2 == nil {
			return
		}
	}
	if _, err := b.Client.SendMessage(ctx, chatID, html); err != nil {
		if _, err2 := b.Client.SendMessageMode(ctx, chatID, PlainText(html), ""); err2 != nil {
			log.Warn("telegram AI deliver failed", "err", err, "plain_err", err2)
		}
	}
}

func formatAIFailure(err error, timedOut bool, budget time.Duration) string {
	if timedOut {
		return "⏱ " + bold("Timed out") + "\n" +
			italic(fmt.Sprintf("Deep multi-agent runs can exceed %s.", budget.Round(time.Second))) + "\n\n" +
			"Try a narrower question, e.g.:\n" +
			code("/ask JUVUSDT price and RSI on binance") + "\n\n" +
			"Or raise " + code("AI_TIMEOUT") + " (e.g. 300s) in backend/.env."
	}
	return "⚠️ " + bold("AI unavailable") + "\n" +
		esc(err.Error()) + "\n\n" +
		italic("Ensure AI is running (AI_AUTOSTART) and try again.")
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

package telegram

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// Client is a minimal Telegram Bot API client (long polling).
type Client struct {
	token  string
	http   *http.Client
	apiURL string
}

// NewClient constructs a Telegram client. timeout must exceed long-poll wait.
func NewClient(token string, timeout time.Duration) *Client {
	if timeout <= 0 {
		timeout = 60 * time.Second
	}
	return &Client{
		token:  token,
		http:   &http.Client{Timeout: timeout},
		apiURL: "https://api.telegram.org",
	}
}

// Update is a Telegram update.
type Update struct {
	UpdateID      int64          `json:"update_id"`
	Message       *Message       `json:"message"`
	CallbackQuery *CallbackQuery `json:"callback_query"`
}

// CallbackQuery is an inline-button press.
type CallbackQuery struct {
	ID      string   `json:"id"`
	From    *User    `json:"from"`
	Message *Message `json:"message"`
	Data    string   `json:"data"`
}

// Message is an incoming message.
type Message struct {
	MessageID int64  `json:"message_id"`
	Text      string `json:"text"`
	Chat      Chat   `json:"chat"`
	From      *User  `json:"from"`
}

// Chat is a Telegram chat.
type Chat struct {
	ID int64 `json:"id"`
}

// User is a Telegram user.
type User struct {
	ID       int64  `json:"id"`
	Username string `json:"username"`
}

type apiResponse struct {
	OK          bool            `json:"ok"`
	Description string          `json:"description"`
	Result      json.RawMessage `json:"result"`
}

// GetUpdates long-polls for updates.
func (c *Client) GetUpdates(ctx context.Context, offset int64, pollTimeoutSec int) ([]Update, error) {
	q := url.Values{}
	q.Set("timeout", strconv.Itoa(pollTimeoutSec))
	if offset > 0 {
		q.Set("offset", strconv.FormatInt(offset, 10))
	}
	q.Set("allowed_updates", `["message","callback_query"]`)
	var updates []Update
	if err := c.call(ctx, "getUpdates", q, nil, &updates); err != nil {
		return nil, err
	}
	return updates, nil
}

// SendMessage sends HTML text; returns Telegram message_id (0 on parse failure of id).
func (c *Client) SendMessage(ctx context.Context, chatID int64, text string) (int64, error) {
	return c.SendMessageMode(ctx, chatID, text, "HTML")
}

// SendMessageMode sends with parse_mode (empty = plain text). Returns message_id.
func (c *Client) SendMessageMode(ctx context.Context, chatID int64, text, parseMode string) (int64, error) {
	return c.SendMessageMarkup(ctx, chatID, text, parseMode, "")
}

// SendMessageMarkup sends text with optional inline keyboard JSON (reply_markup).
func (c *Client) SendMessageMarkup(ctx context.Context, chatID int64, text, parseMode, replyMarkup string) (int64, error) {
	text = clipTelegram(text)
	form := url.Values{}
	form.Set("chat_id", strconv.FormatInt(chatID, 10))
	form.Set("text", text)
	if parseMode != "" {
		form.Set("parse_mode", parseMode)
	}
	form.Set("disable_web_page_preview", "true")
	if strings.TrimSpace(replyMarkup) != "" {
		form.Set("reply_markup", replyMarkup)
	}
	var msg struct {
		MessageID int64 `json:"message_id"`
	}
	if err := c.call(ctx, "sendMessage", nil, form, &msg); err != nil {
		return 0, err
	}
	return msg.MessageID, nil
}

// EditMessageText edits an existing message (plain or HTML).
func (c *Client) EditMessageText(ctx context.Context, chatID, messageID int64, text, parseMode string) error {
	return c.EditMessageMarkup(ctx, chatID, messageID, text, parseMode, "")
}

// EditMessageMarkup edits text and optional reply_markup JSON.
func (c *Client) EditMessageMarkup(ctx context.Context, chatID, messageID int64, text, parseMode, replyMarkup string) error {
	text = clipTelegram(text)
	form := url.Values{}
	form.Set("chat_id", strconv.FormatInt(chatID, 10))
	form.Set("message_id", strconv.FormatInt(messageID, 10))
	form.Set("text", text)
	if parseMode != "" {
		form.Set("parse_mode", parseMode)
	}
	form.Set("disable_web_page_preview", "true")
	if strings.TrimSpace(replyMarkup) != "" {
		form.Set("reply_markup", replyMarkup)
	}
	return c.call(ctx, "editMessageText", nil, form, nil)
}

// AnswerCallbackQuery acknowledges an inline button press (stops the spinner).
func (c *Client) AnswerCallbackQuery(ctx context.Context, callbackID, text string) error {
	form := url.Values{}
	form.Set("callback_query_id", callbackID)
	if strings.TrimSpace(text) != "" {
		form.Set("text", clipRunes(text, 180))
	}
	return c.call(ctx, "answerCallbackQuery", nil, form, nil)
}

func clipTelegram(text string) string {
	// Telegram hard limit 4096; leave margin for parse mode.
	const max = 4000
	if len(text) <= max {
		return text
	}
	// Prefer rune-safe cut
	r := []rune(text)
	if len(r) <= max {
		return text
	}
	return string(r[:max-1]) + "…"
}

// GetMe verifies the token.
func (c *Client) GetMe(ctx context.Context) (string, error) {
	var me struct {
		Username string `json:"username"`
	}
	if err := c.call(ctx, "getMe", nil, nil, &me); err != nil {
		return "", err
	}
	return me.Username, nil
}

func (c *Client) call(ctx context.Context, method string, query url.Values, form url.Values, dest any) error {
	u := fmt.Sprintf("%s/bot%s/%s", c.apiURL, c.token, method)
	if len(query) > 0 {
		u += "?" + query.Encode()
	}
	var body io.Reader
	methodHTTP := http.MethodGet
	if form != nil {
		methodHTTP = http.MethodPost
		body = strings.NewReader(form.Encode())
	}
	req, err := http.NewRequestWithContext(ctx, methodHTTP, u, body)
	if err != nil {
		return err
	}
	if form != nil {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return err
	}
	var ar apiResponse
	if err := json.Unmarshal(raw, &ar); err != nil {
		return fmt.Errorf("telegram decode: %w", err)
	}
	if !ar.OK {
		return fmt.Errorf("telegram: %s", ar.Description)
	}
	if dest == nil || len(ar.Result) == 0 || string(ar.Result) == "true" {
		return nil
	}
	return json.Unmarshal(ar.Result, dest)
}

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
	UpdateID int64    `json:"update_id"`
	Message  *Message `json:"message"`
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
	q.Set("allowed_updates", `["message"]`)
	var updates []Update
	if err := c.call(ctx, "getUpdates", q, nil, &updates); err != nil {
		return nil, err
	}
	return updates, nil
}

// SendMessage sends a text message. parseMode is "HTML", "Markdown", or "" for plain.
func (c *Client) SendMessage(ctx context.Context, chatID int64, text string) error {
	return c.SendMessageMode(ctx, chatID, text, "HTML")
}

// SendMessageMode sends with an explicit parse_mode (empty = plain text).
func (c *Client) SendMessageMode(ctx context.Context, chatID int64, text, parseMode string) error {
	if len(text) > 4000 {
		text = text[:3990] + "…"
	}
	form := url.Values{}
	form.Set("chat_id", strconv.FormatInt(chatID, 10))
	form.Set("text", text)
	if parseMode != "" {
		form.Set("parse_mode", parseMode)
	}
	form.Set("disable_web_page_preview", "true")
	return c.call(ctx, "sendMessage", nil, form, nil)
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

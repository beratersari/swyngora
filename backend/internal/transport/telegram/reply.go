package telegram

import "encoding/json"

// Reply is a bot response: HTML text plus optional inline keyboard.
type Reply struct {
	Text          string
	Keyboard      [][]InlineButton
	ClearKeyboard bool   // strip buttons when editing a confirmation card
	Toast         string // short callback-query popup (plain)
}

// InlineButton is one Telegram inline keyboard button (callback only).
type InlineButton struct {
	Text         string
	CallbackData string
}

func textReply(s string) Reply {
	return Reply{Text: s}
}

const emptyInlineKeyboardJSON = `{"inline_keyboard":[]}`

type tgInlineButton struct {
	Text         string `json:"text"`
	CallbackData string `json:"callback_data,omitempty"`
}

func encodeInlineKeyboard(rows [][]InlineButton) string {
	if len(rows) == 0 {
		return ""
	}
	out := make([][]tgInlineButton, 0, len(rows))
	for _, row := range rows {
		line := make([]tgInlineButton, 0, len(row))
		for _, b := range row {
			line = append(line, tgInlineButton{Text: b.Text, CallbackData: b.CallbackData})
		}
		out = append(out, line)
	}
	raw, err := json.Marshal(map[string]any{"inline_keyboard": out})
	if err != nil {
		return ""
	}
	return string(raw)
}

func confirmCancelKeyboard(id string) [][]InlineButton {
	return [][]InlineButton{{
		{Text: "✅ Confirm", CallbackData: "pf:c:" + id},
		{Text: "❌ Cancel", CallbackData: "pf:x:" + id},
	}}
}

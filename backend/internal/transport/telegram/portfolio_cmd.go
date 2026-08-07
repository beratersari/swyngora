package telegram

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
	"gitlab.com/trace-analysis/swyngora/backend/internal/service/portfolio"
)

func telegramClientID(userID int64) string {
	return fmt.Sprintf("tg-%d", userID)
}

func (r *Router) cmdPortfolio(ctx context.Context, userID int64, args []string) Reply {
	if r.portfolio == nil {
		return textReply("Paper trading is not configured on this bot.")
	}
	clientID := telegramClientID(userID)
	if len(args) > 0 && strings.EqualFold(args[0], "create") {
		return r.cmdPortfolioCreate(ctx, clientID, args[1:])
	}
	if len(args) > 0 && strings.EqualFold(args[0], "deposit") {
		return r.cmdCashMove(ctx, userID, domain.CashMovementDeposit, args[1:])
	}
	if len(args) > 0 && strings.EqualFold(args[0], "withdraw") {
		return r.cmdCashMove(ctx, userID, domain.CashMovementWithdrawal, args[1:])
	}
	if len(args) > 0 && (strings.EqualFold(args[0], "cash") || strings.EqualFold(args[0], "history")) {
		return r.cmdCashHistory(ctx, userID, args[1:])
	}
	view, err := r.portfolio.View(ctx, clientID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return textReply("No paper portfolio yet.\n\nCreate one:\n" + code("/portfolio create 10000") +
				"\n\n" + italic("Simulated only — not real money."))
		}
		return textReply(friendlyErr(err))
	}
	return textReply(FormatPaperPortfolio(view))
}

func (r *Router) cmdPortfolioCreate(ctx context.Context, clientID string, args []string) Reply {
	bal := 10000.0
	if len(args) >= 1 {
		v, err := strconv.ParseFloat(strings.ReplaceAll(args[0], ",", ""), 64)
		if err != nil || v <= 0 {
			return textReply("Usage: /portfolio create [startingBalance]\nExample: " + code("/portfolio create 10000"))
		}
		bal = v
	}
	p, err := r.portfolio.Create(ctx, portfolio.CreateInput{
		ClientID: clientID, StartingBalance: bal, Currency: domain.DefaultPaperCurrency,
	})
	if err != nil {
		return textReply(friendlyErr(err))
	}
	view, err := r.portfolio.View(ctx, p.ClientID)
	if err != nil {
		return textReply("✅ Paper portfolio created with " + code(Float(p.StartingBalance, 2)+" "+p.Currency) + ".")
	}
	return textReply("✅ " + bold("Paper portfolio created") + "\n\n" + FormatPaperPortfolio(view))
}

func (r *Router) cmdCashMove(ctx context.Context, userID int64, kind domain.CashMovementKind, args []string) Reply {
	if r.portfolio == nil {
		return textReply("Paper trading is not configured on this bot.")
	}
	verb := "deposit"
	if kind == domain.CashMovementWithdrawal {
		verb = "withdraw"
	}
	if len(args) < 1 {
		return textReply(fmt.Sprintf("Usage: /%s <amount> [note]\nExample: %s", verb, code("/"+verb+" 500")))
	}
	amt, err := strconv.ParseFloat(strings.ReplaceAll(args[0], ",", ""), 64)
	if err != nil || amt <= 0 {
		return textReply("Amount must be a positive number.")
	}
	note := strings.TrimSpace(strings.Join(args[1:], " "))
	in := portfolio.CashMoveInput{ClientID: telegramClientID(userID), Amount: amt, Note: note}
	var (
		m *domain.CashMovement
		v *domain.PortfolioView
	)
	if kind == domain.CashMovementDeposit {
		m, v, err = r.portfolio.Deposit(ctx, in)
	} else {
		m, v, err = r.portfolio.Withdraw(ctx, in)
	}
	if err != nil {
		return textReply(friendlyErr(err))
	}
	return textReply(FormatCashMoved(m, v))
}

func (r *Router) cmdCashHistory(ctx context.Context, userID int64, args []string) Reply {
	if r.portfolio == nil {
		return textReply("Paper trading is not configured on this bot.")
	}
	limit := 10
	if len(args) >= 1 {
		if n, err := strconv.Atoi(args[0]); err == nil && n > 0 {
			limit = n
			if limit > 25 {
				limit = 25
			}
		}
	}
	list, total, err := r.portfolio.ListCashMovements(ctx, telegramClientID(userID), limit, 0)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return textReply("No paper portfolio yet. Create one:\n" + code("/portfolio create 10000"))
		}
		return textReply(friendlyErr(err))
	}
	return textReply(FormatCashHistory(list, total))
}

func (r *Router) cmdTradePreview(ctx context.Context, chatID, userID int64, side domain.TradeSide, args []string) Reply {
	if r.portfolio == nil {
		return textReply("Paper trading is not configured on this bot.")
	}
	if len(args) < 2 {
		verb := "buy"
		if side == domain.TradeSideSell {
			verb = "sell"
		}
		return textReply(fmt.Sprintf("Usage: /%s <symbol> <quantity> [exchange]\nExample: %s",
			verb, code(fmt.Sprintf("/%s BTCUSDT 0.01", verb))))
	}
	symbol := strings.ToUpper(args[0])
	qty, err := strconv.ParseFloat(strings.ReplaceAll(args[1], ",", ""), 64)
	if err != nil || qty <= 0 {
		return textReply("Quantity must be a positive number.\nExample: " + code("/buy BTCUSDT 0.01"))
	}
	exchange := r.defaultExchange()
	if len(args) >= 3 {
		exchange = strings.ToLower(args[2])
	}

	clientID := telegramClientID(userID)
	view, err := r.portfolio.View(ctx, clientID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return textReply("No paper portfolio yet. Create one first:\n" + code("/portfolio create 10000"))
		}
		return textReply(friendlyErr(err))
	}

	tkr, err := r.market.GetTicker24h(ctx, exchange, symbol)
	if err != nil {
		return textReply(friendlyErr(err))
	}
	price, err := strconv.ParseFloat(strings.TrimSpace(tkr.LastPrice), 64)
	if err != nil || price <= 0 {
		return textReply("Could not read last price for " + code(symbol) + ".")
	}
	ex, _ := r.market.ResolveExchange(exchange)
	exName := string(ex)
	notional := qty * price

	if side == domain.TradeSideBuy && notional > view.AvailableCash+1e-9 {
		return textReply(fmt.Sprintf("Not enough available cash.\nNeed %s, available %s %s.",
			code(Float(notional, 4)), code(Float(view.AvailableCash, 4)), esc(view.Currency)))
	}
	if side == domain.TradeSideSell {
		avail := 0.0
		for _, pos := range view.Positions {
			if string(pos.Exchange) == exName && strings.EqualFold(pos.Symbol, tkr.Symbol) {
				avail = pos.AvailableQuantity
				break
			}
		}
		if qty > avail+1e-9 {
			return textReply(fmt.Sprintf("Not enough %s to sell.\nNeed %s, available %s.",
				code(tkr.Symbol), code(Float(qty, 8)), code(Float(avail, 8))))
		}
	}

	id := newPendingID()
	r.pending.put(&pendingTrade{
		ID: id, ChatID: chatID, UserID: userID, ClientID: clientID,
		Side: side, Exchange: exName, Symbol: tkr.Symbol,
		Quantity: qty, QuotePrice: price, Notional: notional,
		ExpiresAt: time.Now().Add(pendingTradeTTL),
	})
	body := FormatTradePreview(side, exName, tkr.Symbol, qty, price, notional, view.Currency)
	return Reply{Text: body, Keyboard: confirmCancelKeyboard(id)}
}

func (r *Router) HandleCallback(ctx context.Context, chatID, userID int64, data string) Reply {
	if !r.Allowed(chatID) {
		return Reply{Text: "This bot is private. Your chat is not allowed.", Toast: "Not allowed"}
	}
	data = strings.TrimSpace(data)
	parts := strings.Split(data, ":")
	if len(parts) != 3 || parts[0] != "pf" {
		return Reply{Text: "Unknown button.", Toast: "Unknown"}
	}
	action, id := parts[1], parts[2]
	switch action {
	case "c":
		return r.confirmPendingTrade(ctx, chatID, userID, id)
	case "x":
		return r.cancelPendingTrade(chatID, userID, id)
	default:
		return Reply{Text: "Unknown button.", Toast: "Unknown"}
	}
}

func (r *Router) confirmPendingTrade(ctx context.Context, chatID, userID int64, id string) Reply {
	if r.portfolio == nil {
		return Reply{Text: "Paper trading is not configured.", ClearKeyboard: true, Toast: "Unavailable"}
	}
	p := r.pending.take(id)
	if p == nil {
		return Reply{Text: "This confirmation expired or was already used.\nSend /buy or /sell again.", ClearKeyboard: true, Toast: "Expired"}
	}
	if p.ChatID != chatID || p.UserID != userID {
		r.pending.restore(p)
		return Reply{Text: "This confirmation belongs to another user.", Toast: "Not yours"}
	}
	tr, view, err := r.portfolio.PlaceOrder(ctx, portfolio.OrderInput{
		ClientID: p.ClientID, Exchange: p.Exchange, Symbol: p.Symbol,
		Side: string(p.Side), Quantity: p.Quantity,
	})
	if err != nil {
		r.pending.restore(p)
		return Reply{Text: friendlyErr(err), ClearKeyboard: true, Toast: "Failed"}
	}
	cur := "USDT"
	if view != nil && view.Currency != "" {
		cur = view.Currency
	}
	return Reply{
		Text:          FormatTradeFilled(tr, p.QuotePrice, cur, view),
		ClearKeyboard: true,
		Toast:         "Filled",
	}
}

func (r *Router) cancelPendingTrade(chatID, userID int64, id string) Reply {
	p := r.pending.take(id)
	if p == nil {
		return Reply{Text: "Nothing to cancel — this confirmation is gone.", ClearKeyboard: true, Toast: "Gone"}
	}
	if p.ChatID != chatID || p.UserID != userID {
		r.pending.restore(p)
		return Reply{Text: "This confirmation belongs to another user.", Toast: "Not yours"}
	}
	return Reply{Text: FormatTradeCanceled(p.Side, p.Symbol), ClearKeyboard: true, Toast: "Canceled"}
}

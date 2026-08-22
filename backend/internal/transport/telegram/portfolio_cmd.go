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

func (r *Router) selectedBookID(userID int64) string {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.selectedPF == nil {
		return ""
	}
	return r.selectedPF[userID]
}

func (r *Router) setSelectedBook(userID int64, id string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.selectedPF == nil {
		r.selectedPF = map[int64]string{}
	}
	if id == "" {
		delete(r.selectedPF, userID)
		return
	}
	r.selectedPF[userID] = id
}

func (r *Router) resolveBook(ctx context.Context, userID int64) (clientID, bookID string, err error) {
	clientID, err = r.clientIDForUser(ctx, userID)
	if err != nil {
		return "", "", err
	}
	list, err := r.portfolio.List(ctx, clientID)
	if err != nil {
		return clientID, "", err
	}
	sel := r.selectedBookID(userID)
	if sel != "" {
		for _, p := range list {
			if p.ID == sel {
				return clientID, p.ID, nil
			}
		}
		if p, gerr := r.portfolio.Get(ctx, clientID, sel); gerr == nil && p != nil {
			return clientID, p.ID, nil
		}
	}
	if len(list) == 1 {
		r.setSelectedBook(userID, list[0].ID)
		return clientID, list[0].ID, nil
	}
	if len(list) > 1 {
		r.setSelectedBook(userID, list[0].ID)
		return clientID, list[0].ID, nil
	}
	shared, err := r.portfolio.ListSharedWithMe(ctx, clientID)
	if err != nil {
		return clientID, "", err
	}
	if len(shared) == 0 {
		return clientID, "", domain.ErrNotFound
	}
	r.setSelectedBook(userID, shared[0].Portfolio.ID)
	return clientID, shared[0].Portfolio.ID, nil
}

func (r *Router) cmdPortfolio(ctx context.Context, userID int64, args []string) Reply {
	if r.portfolio == nil {
		return textReply("Paper trading is not configured on this bot.")
	}
	clientID, err := r.clientIDForUser(ctx, userID)
	if err != nil {
		return textReply(friendlyErr(err))
	}
	if len(args) > 0 {
		switch strings.ToLower(args[0]) {
		case "create":
			return r.cmdPortfolioCreate(ctx, userID, clientID, args[1:])
		case "list":
			return r.cmdPortfolioList(ctx, userID, clientID)
		case "use", "select":
			return r.cmdPortfolioUse(ctx, userID, clientID, args[1:])
		case "rename":
			return r.cmdPortfolioRename(ctx, userID, args[1:])
		case "delete":
			return r.cmdPortfolioDelete(ctx, userID, args[1:])
		case "share":
			return r.cmdPortfolioShare(ctx, userID, args[1:])
		case "unshare", "revoke":
			return r.cmdPortfolioUnshare(ctx, userID, args[1:])
		case "shares":
			return r.cmdPortfolioShares(ctx, userID)
		case "shared":
			return r.cmdPortfolioShared(ctx, userID)
		case "deposit":
			return r.cmdCashMove(ctx, userID, domain.CashMovementDeposit, args[1:])
		case "withdraw":
			return r.cmdCashMove(ctx, userID, domain.CashMovementWithdrawal, args[1:])
		case "cash", "history":
			return r.cmdCashHistory(ctx, userID, args[1:])
		case "transfer":
			return r.cmdPortfolioTransfer(ctx, userID, args[1:])
		}
	}
	_, bookID, err := r.resolveBook(ctx, userID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return textReply("No paper portfolio yet.\n\nCreate one:\n" + code("/portfolio create 10000") +
				"\nor a named book:\n" + code("/portfolio create 5000 Risky") +
				"\n\n" + italic("Simulated only — not real money."))
		}
		return textReply(friendlyErr(err))
	}
	view, err := r.portfolio.View(ctx, clientID, bookID)
	if err != nil {
		return textReply(friendlyErr(err))
	}
	return textReply(FormatPaperPortfolio(view))
}

func (r *Router) cmdPortfolioList(ctx context.Context, userID int64, clientID string) Reply {
	list, err := r.portfolio.List(ctx, clientID)
	if err != nil {
		return textReply(friendlyErr(err))
	}
	if len(list) == 0 {
		return textReply("No paper portfolios yet.\n" + code("/portfolio create 10000"))
	}
	sel := r.selectedBookID(userID)
	var b strings.Builder
	b.WriteString(bold("Paper portfolios") + "\n")
	for _, p := range list {
		mark := "•"
		if p.ID == sel || (sel == "" && len(list) == 1) {
			mark = "▸"
		}
		b.WriteString(fmt.Sprintf("%s %s  %s %s\n", mark, code(p.Name), code(Float(p.CashBalance, 2)), esc(p.Currency)))
	}
	b.WriteString("\n" + italic("Select: ") + code("/portfolio use NAME"))
	return textReply(b.String())
}

func (r *Router) cmdPortfolioUse(ctx context.Context, userID int64, clientID string, args []string) Reply {
	if len(args) < 1 {
		return textReply("Usage: /portfolio use <name or id>")
	}
	ref := strings.Join(args, " ")
	p, err := r.portfolio.Get(ctx, clientID, ref)
	if err != nil {
		return textReply(friendlyErr(err))
	}
	r.setSelectedBook(userID, p.ID)
	view, err := r.portfolio.View(ctx, clientID, p.ID)
	if err != nil {
		return textReply("✅ Selected " + code(p.Name) + ".")
	}
	return textReply("✅ Selected " + code(p.Name) + "\n\n" + FormatPaperPortfolio(view))
}

func (r *Router) cmdPortfolioRename(ctx context.Context, userID int64, args []string) Reply {
	if len(args) < 1 {
		return textReply("Usage: /portfolio rename <new name>")
	}
	clientID, bookID, err := r.resolveBook(ctx, userID)
	if err != nil {
		return textReply(friendlyErr(err))
	}
	p, err := r.portfolio.Rename(ctx, clientID, bookID, strings.Join(args, " "))
	if err != nil {
		return textReply(friendlyErr(err))
	}
	return textReply("✅ Renamed to " + code(p.Name) + ".")
}

func (r *Router) cmdPortfolioDelete(ctx context.Context, userID int64, args []string) Reply {
	clientID, err := r.clientIDForUser(ctx, userID)
	if err != nil {
		return textReply(friendlyErr(err))
	}
	ref := strings.Join(args, " ")
	if ref == "" {
		_, bookID, err := r.resolveBook(ctx, userID)
		if err != nil {
			return textReply(friendlyErr(err))
		}
		ref = bookID
	}
	p, err := r.portfolio.Get(ctx, clientID, ref)
	if err != nil {
		return textReply(friendlyErr(err))
	}
	if err := r.portfolio.Delete(ctx, clientID, p.ID); err != nil {
		return textReply(friendlyErr(err))
	}
	if r.selectedBookID(userID) == p.ID {
		r.setSelectedBook(userID, "")
	}
	return textReply("✅ Deleted " + code(p.Name) + ".")
}

func (r *Router) cmdPortfolioShare(ctx context.Context, userID int64, args []string) Reply {
	if len(args) < 2 {
		return textReply("Usage: /portfolio share <clientId> viewer|trader\nExample: " + code("/portfolio share bob trader"))
	}
	clientID, bookID, err := r.resolveBook(ctx, userID)
	if err != nil {
		return textReply(friendlyErr(err))
	}
	role := args[len(args)-1]
	grantee := strings.Join(args[:len(args)-1], " ")
	sh, err := r.portfolio.Share(ctx, clientID, bookID, grantee, role)
	if err != nil {
		return textReply(friendlyErr(err))
	}
	return textReply(fmt.Sprintf("✅ Shared with %s as %s.", code(sh.GranteeClientID), code(string(sh.Role))))
}

func (r *Router) cmdPortfolioUnshare(ctx context.Context, userID int64, args []string) Reply {
	if len(args) < 1 {
		return textReply("Usage: /portfolio unshare <clientId>")
	}
	clientID, bookID, err := r.resolveBook(ctx, userID)
	if err != nil {
		return textReply(friendlyErr(err))
	}
	grantee := strings.Join(args, " ")
	if err := r.portfolio.RevokeShare(ctx, clientID, bookID, grantee); err != nil {
		return textReply(friendlyErr(err))
	}
	return textReply("✅ Revoked access for " + code(grantee) + ".")
}

func (r *Router) cmdPortfolioShares(ctx context.Context, userID int64) Reply {
	clientID, bookID, err := r.resolveBook(ctx, userID)
	if err != nil {
		return textReply(friendlyErr(err))
	}
	list, err := r.portfolio.ListShares(ctx, clientID, bookID)
	if err != nil {
		return textReply(friendlyErr(err))
	}
	if len(list) == 0 {
		return textReply("No one else can access this book.\n" + code("/portfolio share clientId trader"))
	}
	var b strings.Builder
	b.WriteString(bold("Shared with") + "\n")
	for _, sh := range list {
		b.WriteString(fmt.Sprintf("• %s  %s\n", code(sh.GranteeClientID), esc(string(sh.Role))))
	}
	return textReply(b.String())
}

func (r *Router) cmdPortfolioShared(ctx context.Context, userID int64) Reply {
	clientID, err := r.clientIDForUser(ctx, userID)
	if err != nil {
		return textReply(friendlyErr(err))
	}
	list, err := r.portfolio.ListSharedWithMe(ctx, clientID)
	if err != nil {
		return textReply(friendlyErr(err))
	}
	if len(list) == 0 {
		return textReply("No paper books have been shared with you.")
	}
	var b strings.Builder
	b.WriteString(bold("Shared with you") + "\n")
	for _, item := range list {
		b.WriteString(fmt.Sprintf("• %s  %s  (%s)\n", code(item.Portfolio.Name), esc(string(item.Role)), code(item.Portfolio.ClientID)))
	}
	b.WriteString("\n" + italic("Select: ") + code("/portfolio use NAME"))
	return textReply(b.String())
}

func (r *Router) cmdPortfolioCreate(ctx context.Context, userID int64, clientID string, args []string) Reply {
	bal := 10000.0
	name := ""
	if len(args) >= 1 {
		v, err := strconv.ParseFloat(strings.ReplaceAll(args[0], ",", ""), 64)
		if err == nil && v > 0 {
			bal = v
			if len(args) >= 2 {
				name = strings.Join(args[1:], " ")
			}
		} else {
			name = strings.Join(args, " ")
		}
	}
	p, err := r.portfolio.Create(ctx, portfolio.CreateInput{
		ClientID: clientID, Name: name, StartingBalance: bal, Currency: domain.DefaultPaperCurrency,
	})
	if err != nil {
		return textReply(friendlyErr(err))
	}
	r.setSelectedBook(userID, p.ID)
	view, err := r.portfolio.View(ctx, p.ClientID, p.ID)
	if err != nil {
		return textReply("✅ Paper portfolio created with " + code(Float(p.StartingBalance, 2)+" "+p.Currency) + ".")
	}
	return textReply("✅ " + bold("Paper portfolio created") + " — " + code(p.Name) + "\n\n" + FormatPaperPortfolio(view))
}

func (r *Router) cmdPortfolioTransfer(ctx context.Context, userID int64, args []string) Reply {
	if r.portfolio == nil {
		return textReply("Paper trading is not configured on this bot.")
	}
	if len(args) < 2 {
		return textReply("Usage: /transfer <amount> <toName>\nExample: " + code("/transfer 500 Risky") +
			"\nMoves cash from the selected book to another of yours.")
	}
	amt, err := strconv.ParseFloat(strings.ReplaceAll(args[0], ",", ""), 64)
	if err != nil || amt <= 0 {
		return textReply("Amount must be a positive number.")
	}
	toRef := strings.Join(args[1:], " ")
	clientID, fromID, err := r.resolveBook(ctx, userID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return textReply("No paper portfolio yet. Create one:\n" + code("/portfolio create 10000"))
		}
		return textReply(friendlyErr(err))
	}
	out, in, fromV, toV, err := r.portfolio.Transfer(ctx, portfolio.TransferInput{
		ClientID: clientID, FromPortfolioID: fromID, ToPortfolioID: toRef, Amount: amt,
	})
	if err != nil {
		return textReply(friendlyErr(err))
	}
	return textReply(FormatCashTransferred(out, in, fromV, toV))
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
	_, bookID, err2 := r.resolveBook(ctx, userID)
	if err2 != nil {
		if errors.Is(err2, domain.ErrNotFound) {
			return textReply("No paper portfolio yet. Create one:\n" + code("/portfolio create 10000"))
		}
		return textReply(friendlyErr(err2))
	}
	clientID, cerr := r.clientIDForUser(ctx, userID)
	if cerr != nil {
		return textReply(friendlyErr(cerr))
	}
	in := portfolio.CashMoveInput{ClientID: clientID, PortfolioID: bookID, Amount: amt, Note: note}
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
	_, bookID, err := r.resolveBook(ctx, userID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return textReply("No paper portfolio yet. Create one:\n" + code("/portfolio create 10000"))
		}
		return textReply(friendlyErr(err))
	}
	clientID, cerr := r.clientIDForUser(ctx, userID)
	if cerr != nil {
		return textReply(friendlyErr(cerr))
	}
	list, total, err := r.portfolio.ListCashMovements(ctx, clientID, limit, 0, bookID)
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
		extra := ""
		if side == domain.TradeSideSell {
			extra = " [fifo|lifo]"
		}
		return textReply(fmt.Sprintf("Usage: /%s <symbol> <quantity> [exchange]%s\nExample: %s",
			verb, extra, code(fmt.Sprintf("/%s BTCUSDT 0.01", verb))))
	}
	symbol := strings.ToUpper(args[0])
	qty, err := strconv.ParseFloat(strings.ReplaceAll(args[1], ",", ""), 64)
	if err != nil || qty <= 0 {
		return textReply("Quantity must be a positive number.\nExample: " + code("/buy BTCUSDT 0.01"))
	}
	exchange := r.defaultExchange()
	lotMethod := ""
	if len(args) >= 3 {
		a2 := strings.ToLower(args[2])
		if a2 == "fifo" || a2 == "lifo" {
			lotMethod = a2
		} else {
			exchange = a2
		}
	}
	if len(args) >= 4 {
		a3 := strings.ToLower(args[3])
		if a3 == "fifo" || a3 == "lifo" {
			lotMethod = a3
		}
	}

	clientID, bookID, err := r.resolveBook(ctx, userID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return textReply("No paper portfolio yet. Create one first:\n" + code("/portfolio create 10000"))
		}
		return textReply(friendlyErr(err))
	}
	view, err := r.portfolio.View(ctx, clientID, bookID)
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
		ID: id, ChatID: chatID, UserID: userID, ClientID: clientID, PortfolioID: bookID,
		Side: side, Exchange: exName, Symbol: tkr.Symbol,
		Quantity: qty, QuotePrice: price, Notional: notional, LotMethod: lotMethod,
		ExpiresAt: time.Now().Add(pendingTradeTTL),
	})
	body := FormatTradePreview(side, exName, tkr.Symbol, qty, price, notional, view.Currency)
	if side == domain.TradeSideSell {
		m := lotMethod
		if m == "" {
			m = "fifo"
		}
		body += "\nLot matching: " + code(m)
	}
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
	if r.opts.Accounts != nil {
		if err := r.opts.Accounts.RequireActive(ctx, p.ClientID); err != nil {
			return Reply{Text: friendlyErr(err), ClearKeyboard: true, Toast: "Closed"}
		}
	}
	tr, view, err := r.portfolio.PlaceOrder(ctx, portfolio.OrderInput{
		ClientID: p.ClientID, PortfolioID: p.PortfolioID, Exchange: p.Exchange, Symbol: p.Symbol,
		Side: string(p.Side), Quantity: p.Quantity, LotMethod: p.LotMethod,
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

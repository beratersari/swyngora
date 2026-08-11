package telegram

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
	"gitlab.com/trace-analysis/swyngora/backend/internal/service/aiagent"
	"gitlab.com/trace-analysis/swyngora/backend/internal/service/market"
	"gitlab.com/trace-analysis/swyngora/backend/internal/service/portfolio"
	"gitlab.com/trace-analysis/swyngora/backend/internal/service/watchlist"
)

// Options configures the command router.
type Options struct {
	DefaultExchange string
	LowMcapLimit    int
	// AllowedChatIDs restricts chats when non-empty.
	AllowedChatIDs map[int64]struct{}
	// AllowAll permits every chat when AllowedChatIDs is empty.
	// When both empty and AllowAll is false, Allowed rejects everyone (fail closed).
	AllowAll bool
	// AI is optional multi-agent assistant (Python service).
	AI *aiagent.Client
	// AITimeout bounds /ask orchestration (default 120s).
	AITimeout time.Duration
	// Identities maps Telegram user ids to unguessable clientIds. Required for /watch.
	Identities domain.TelegramIdentityPort
	// Portfolio enables paper /portfolio /buy /sell (optional).
	Portfolio *portfolio.Service
}

// Router dispatches Telegram text commands to application services.
type Router struct {
	market    *market.Service
	watch     *watchlist.Service
	portfolio *portfolio.Service
	ai        *aiagent.Client
	opts      Options
	now       func() time.Time
	mu         sync.Mutex
	lastAt     map[int64]time.Time
	pending    *pendingStore
	selectedPF map[int64]string // userID → selected paper book id
}

// NewRouter constructs a command router (thin transport → services).
func NewRouter(marketSvc *market.Service, watchSvc *watchlist.Service, opts Options) *Router {
	if opts.DefaultExchange == "" {
		opts.DefaultExchange = string(domain.DefaultExchange)
	}
	if opts.LowMcapLimit <= 0 {
		opts.LowMcapLimit = 10
	}
	if opts.LowMcapLimit > 25 {
		opts.LowMcapLimit = 25
	}
	if opts.AITimeout <= 0 {
		opts.AITimeout = 120 * time.Second
	}
	return &Router{
		market:     marketSvc,
		watch:      watchSvc,
		portfolio:  opts.Portfolio,
		ai:         opts.AI,
		opts:       opts,
		now:        time.Now,
		lastAt:     map[int64]time.Time{},
		pending:    newPendingStore(),
		selectedPF: map[int64]string{},
	}
}

// Allowed reports whether chatID may use the bot.
// Fail closed: empty allowlist only permits traffic when AllowAll is true.
func (r *Router) Allowed(chatID int64) bool {
	if len(r.opts.AllowedChatIDs) == 0 {
		return r.opts.AllowAll
	}
	_, ok := r.opts.AllowedChatIDs[chatID]
	return ok
}

// Handle processes one message text and returns the reply body.
func (r *Router) Handle(ctx context.Context, chatID, userID int64, text string) string {
	return r.HandleMessage(ctx, chatID, userID, text).Text
}

// HandleMessage is Handle plus optional inline keyboard (paper trade confirm).
func (r *Router) HandleMessage(ctx context.Context, chatID, userID int64, text string) Reply {
	if !r.Allowed(chatID) {
		return textReply("This bot is private. Your chat is not allowed.")
	}
	if !r.allowRate(chatID) {
		return textReply("Slow down — try again in a second.")
	}

	text = strings.TrimSpace(text)
	if text == "" {
		return textReply(HelpText())
	}

	parts := strings.Fields(text)
	cmd := strings.ToLower(parts[0])
	if i := strings.IndexByte(cmd, '@'); i > 0 {
		cmd = cmd[:i]
	}
	args := parts[1:]

	switch cmd {
	case "/start", "/help":
		return textReply(HelpText())
	case "/price":
		return textReply(r.cmdPrice(ctx, args))
	case "/spot":
		return textReply(r.cmdSpot(ctx, args))
	case "/lowmcap", "/lowcap":
		return textReply(r.cmdLowMcap(ctx, args))
	case "/mcap", "/supply":
		return textReply(r.cmdMcap(ctx, args))
	case "/rsi":
		return textReply(r.cmdRSI(ctx, args))
	case "/exchanges":
		return textReply(r.cmdExchanges())
	case "/watch":
		return textReply(r.cmdWatch(ctx, userID, args))
	case "/ask", "/ai":
		return textReply(r.cmdAsk(ctx, userID, args))
	case "/portfolio", "/pf":
		return r.cmdPortfolio(ctx, userID, args)
	case "/buy":
		return r.cmdTradePreview(ctx, chatID, userID, domain.TradeSideBuy, args)
	case "/sell":
		return r.cmdTradePreview(ctx, chatID, userID, domain.TradeSideSell, args)
	case "/deposit":
		return r.cmdCashMove(ctx, userID, domain.CashMovementDeposit, args)
	case "/withdraw":
		return r.cmdCashMove(ctx, userID, domain.CashMovementWithdrawal, args)
	case "/cash":
		return r.cmdCashHistory(ctx, userID, args)
	case "/transfer":
		return r.cmdPortfolioTransfer(ctx, userID, args)
	default:
		if strings.HasPrefix(cmd, "/") {
			return textReply("Unknown command. Try /help")
		}
		// Free-text → AI when configured (e.g. "deep analysis for JUV").
		if r.ai != nil {
			return textReply(r.runAI(ctx, userID, text))
		}
		return textReply("Send a command, e.g. /price BTCUSDT or /ask what is BTC RSI? — try /help")
	}
}

// cmdAsk routes natural-language questions to the multi-agent AI service.
func (r *Router) cmdAsk(ctx context.Context, userID int64, args []string) string {
	q := strings.TrimSpace(strings.Join(args, " "))
	if q == "" {
		return "Usage: /ask <question>\nExample: /ask What is BTC RSI on binance 1h and any recent news?"
	}
	return r.runAI(ctx, userID, q)
}

func (r *Router) runAI(ctx context.Context, userID int64, q string) string {
	if r.ai == nil {
		return "AI is not configured. Set AI_SERVICE_URL / AI_AUTOSTART + AI_PYTHON, then restart the backend."
	}
	session, err := r.clientIDForUser(ctx, userID)
	if err != nil {
		return friendlyErr(err)
	}
	aiCtx, cancel := context.WithTimeout(ctx, r.opts.AITimeout)
	defer cancel()
	res, err := r.ai.Chat(aiCtx, q, session, session)
	if err != nil {
		return "AI unavailable: " + esc(err.Error()) + "\n\nEnsure the AI service is running (backend can auto-start it) and try again."
	}
	return FormatAIAnswer(res.Reply, res.Thinking, res.Tools) + FormatAIReferences(toRefLinks(res.References))
}


// IsAIRequest reports whether text should be handled by the multi-agent AI path.
func (r *Router) IsAIRequest(text string) bool {
	text = strings.TrimSpace(text)
	if text == "" {
		return false
	}
	parts := strings.Fields(text)
	cmd := strings.ToLower(parts[0])
	if i := strings.IndexByte(cmd, '@'); i > 0 {
		cmd = cmd[:i]
	}
	if cmd == "/ask" || cmd == "/ai" {
		return true
	}
	// Free-text questions go to AI when configured.
	return !strings.HasPrefix(cmd, "/") && r.ai != nil
}

// AIQuestion extracts the user question from /ask|/ai args or free-text.
func (r *Router) AIQuestion(text string) string {
	text = strings.TrimSpace(text)
	parts := strings.Fields(text)
	if len(parts) == 0 {
		return ""
	}
	cmd := strings.ToLower(parts[0])
	if i := strings.IndexByte(cmd, '@'); i > 0 {
		cmd = cmd[:i]
	}
	if cmd == "/ask" || cmd == "/ai" {
		return strings.TrimSpace(strings.Join(parts[1:], " "))
	}
	if strings.HasPrefix(cmd, "/") {
		return ""
	}
	return text
}

func (r *Router) allowRate(chatID int64) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	now := r.now()
	if t, ok := r.lastAt[chatID]; ok && now.Sub(t) < 400*time.Millisecond {
		return false
	}
	r.lastAt[chatID] = now
	return true
}

func (r *Router) defaultExchange() string {
	return r.opts.DefaultExchange
}

func (r *Router) cmdPrice(ctx context.Context, args []string) string {
	if len(args) < 1 {
		return "Usage: /price <symbol> [exchange]\nExample: /price BTCUSDT\nExample: /price BTC-USD coinbase"
	}
	symbol := strings.ToUpper(args[0])
	exchange := r.defaultExchange()
	if len(args) >= 2 {
		exchange = strings.ToLower(args[1])
	}
	tkr, err := r.market.GetTicker24h(ctx, exchange, symbol)
	if err != nil {
		return friendlyErr(err)
	}
	ex, _ := r.market.ResolveExchange(exchange)
	return FormatTicker(string(ex), tkr)
}

func (r *Router) cmdSpot(ctx context.Context, args []string) string {
	exchange := r.defaultExchange()
	q := ""
	if len(args) >= 1 {
		if isExchange(args[0]) {
			exchange = strings.ToLower(args[0])
			if len(args) >= 2 {
				q = strings.Join(args[1:], " ")
			}
		} else {
			q = strings.Join(args, " ")
		}
	}
	query := domain.SpotListQuery{
		Query:  q,
		SortBy: domain.SpotSortQuoteVolume,
		Order:  domain.SortDesc,
		Limit:  10,
	}
	if exchange == "binance" || exchange == "bybit" {
		query.QuoteAsset = "USDT"
	}
	res, err := r.market.ListSpotMarkets(ctx, exchange, query)
	if err != nil {
		return friendlyErr(err)
	}
	title := "Top by quote volume"
	if q != "" {
		title = "Search: " + q
	}
	return FormatSpotList(title, string(res.Exchange), res.Items, res.Total)
}

func (r *Router) cmdLowMcap(ctx context.Context, args []string) string {
	limit := r.opts.LowMcapLimit
	exchange := r.defaultExchange()
	all := false
	for _, a := range args {
		al := strings.ToLower(a)
		if al == "all" {
			all = true
			continue
		}
		if isExchange(al) {
			exchange = al
			continue
		}
		if n, err := strconv.Atoi(a); err == nil && n > 0 {
			limit = n
			if limit > 25 {
				limit = 25
			}
		}
	}
	if all {
		return r.lowMcapAll(ctx, limit)
	}
	return r.lowMcapOne(ctx, exchange, limit)
}

func (r *Router) lowMcapOne(ctx context.Context, exchange string, limit int) string {
	query := domain.SpotListQuery{
		SortBy: domain.SpotSortMarketCapCirculating,
		Order:  domain.SortAsc,
		Limit:  limit,
	}
	if exchange == "binance" || exchange == "bybit" {
		query.QuoteAsset = "USDT"
	}
	res, err := r.market.ListSpotMarkets(ctx, exchange, query)
	if err != nil {
		return friendlyErr(err)
	}
	return FormatSpotList(fmt.Sprintf("Lowest circ. mcap (top %d)", limit), string(res.Exchange), res.Items, res.Total)
}

func (r *Router) lowMcapAll(ctx context.Context, limit int) string {
	exchanges := []string{"binance", "coinbase", "bybit"}
	sections := make([]LowMcapSection, 0, len(exchanges))
	for _, ex := range exchanges {
		query := domain.SpotListQuery{
			SortBy: domain.SpotSortMarketCapCirculating,
			Order:  domain.SortAsc,
			Limit:  limit,
		}
		if ex == "binance" || ex == "bybit" {
			query.QuoteAsset = "USDT"
		}
		res, err := r.market.ListSpotMarkets(ctx, ex, query)
		if err != nil {
			sections = append(sections, LowMcapSection{Exchange: ex, Err: friendlyErr(err)})
			continue
		}
		sections = append(sections, LowMcapSection{
			Exchange: ex,
			Items:    res.Items,
			Total:    res.Total,
		})
	}
	return FormatLowMcapAll(limit, sections)
}

func (r *Router) cmdMcap(ctx context.Context, args []string) string {
	if len(args) < 1 {
		return "Usage: /mcap <asset|pair>\nExample: /mcap BTC\nExample: /mcap BTCUSDT"
	}
	sup, err := r.market.GetSupply(ctx, strings.ToUpper(args[0]))
	if err != nil {
		return friendlyErr(err)
	}
	return FormatSupply(sup)
}

func (r *Router) cmdRSI(ctx context.Context, args []string) string {
	if len(args) < 1 {
		return "Usage: /rsi <symbol> [interval] [exchange]\nExample: /rsi BTCUSDT 1h binance\nExample: /rsi BTCUSDT binance 1h"
	}
	symbol := strings.ToUpper(args[0])
	interval := "1h"
	exchange := r.defaultExchange()
	// Accept [interval] [exchange], [exchange] [interval], or either alone.
	for _, a := range args[1:] {
		if isExchange(a) {
			exchange = strings.ToLower(a)
			continue
		}
		interval = a
	}
	ser, err := r.market.GetIndicators(ctx, exchange, symbol, interval, 60, 14, []int{12, 26})
	if err != nil {
		return friendlyErr(err)
	}
	return FormatIndicators(ser)
}

func (r *Router) cmdExchanges() string {
	exs := r.market.ListExchanges()
	names := make([]string, len(exs))
	for i, e := range exs {
		names[i] = string(e)
	}
	return FormatExchanges(names, r.defaultExchange())
}

func (r *Router) clientIDForUser(ctx context.Context, userID int64) (string, error) {
	if r.opts.Identities == nil {
		return "", fmt.Errorf("%w: telegram identity store not configured", domain.ErrUpstream)
	}
	return r.opts.Identities.ClientIDForTelegramUser(ctx, userID)
}

func (r *Router) cmdWatch(ctx context.Context, userID int64, args []string) string {
	if r.watch == nil {
		return "Watchlist is not configured."
	}
	clientID, err := r.clientIDForUser(ctx, userID)
	if err != nil {
		return friendlyErr(err)
	}
	if len(args) == 0 {
		wl, err := r.watch.Get(ctx, clientID, "")
		if err != nil {
			return friendlyErr(err)
		}
		return FormatWatchlist(wl.Items)
	}
	sub := strings.ToLower(args[0])
	rest := args[1:]
	switch sub {
	case "add":
		if len(rest) < 1 {
			return "Usage: /watch add <symbol> [exchange]"
		}
		symbol := strings.ToUpper(rest[0])
		exchange := r.defaultExchange()
		if len(rest) >= 2 {
			exchange = strings.ToLower(rest[1])
		}
		wl, err := r.watch.Add(ctx, clientID, "", exchange, symbol, "", domain.WatchlistUnconditionalVersion)
		if err != nil {
			return friendlyErr(err)
		}
		return "✅ " + bold("Added") + "\n\n" + FormatWatchlist(wl.Items)
	case "del", "rm", "remove":
		if len(rest) < 1 {
			return "Usage: /watch del <symbol> [exchange]"
		}
		symbol := strings.ToUpper(rest[0])
		exchange := r.defaultExchange()
		if len(rest) >= 2 {
			exchange = strings.ToLower(rest[1])
		}
		wl, err := r.watch.Remove(ctx, clientID, "", exchange, symbol, domain.WatchlistUnconditionalVersion)
		if err != nil {
			return friendlyErr(err)
		}
		return "🗑 " + bold("Removed") + "\n\n" + FormatWatchlist(wl.Items)
	case "top", "prices":
		return r.cmdWatchTop(ctx, clientID)
	default:
		return "Usage: /watch | /watch add | /watch del | /watch top"
	}
}

func (r *Router) cmdWatchTop(ctx context.Context, clientID string) string {
	wl, err := r.watch.Get(ctx, clientID, "")
	if err != nil {
		return friendlyErr(err)
	}
	if len(wl.Items) == 0 {
		return FormatWatchlist(nil)
	}
	const max = 15
	items := wl.Items
	if len(items) > max {
		items = items[:max]
	}
	lines := make([]string, len(items))
	var wg sync.WaitGroup
	sem := make(chan struct{}, 5)
	for i, it := range items {
		wg.Add(1)
		go func(i int, it domain.WatchlistItem) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			tkr, err := r.market.GetTicker24h(ctx, string(it.Exchange), it.Symbol)
			if err != nil {
				lines[i] = FormatWatchTopRow(i+1, it.Symbol, string(it.Exchange), "", "", "error fetching ticker")
				return
			}
			lines[i] = FormatWatchTopRow(i+1, tkr.Symbol, string(it.Exchange), tkr.LastPrice, tkr.PriceChangePercent, "")
		}(i, it)
	}
	wg.Wait()
	return FormatWatchTop(lines, len(wl.Items), len(items))
}

func isExchange(s string) bool {
	switch strings.ToLower(s) {
	case "binance", "coinbase", "bybit":
		return true
	default:
		return false
	}
}

func friendlyErr(err error) string {
	if err == nil {
		return "Unknown error"
	}
	switch {
	case errors.Is(err, domain.ErrNotFound):
		return "Not found — check symbol / exchange."
	case errors.Is(err, domain.ErrRateLimited):
		return "Rate limited — try again shortly."
	case errors.Is(err, domain.ErrUpstream):
		return "Upstream market data unavailable — try again later."
	case errors.Is(err, domain.ErrInvalidArgument):
		// esc() so HTML parse_mode cannot be broken by validation text.
		return "Error: " + esc(err.Error())
	default:
		return "Error: " + esc(err.Error())
	}
}

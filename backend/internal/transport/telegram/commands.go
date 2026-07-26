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
	"gitlab.com/trace-analysis/swyngora/backend/internal/service/market"
	"gitlab.com/trace-analysis/swyngora/backend/internal/service/watchlist"
)

// Options configures the command router.
type Options struct {
	DefaultExchange string
	LowMcapLimit    int
	// AllowedChatIDs empty = allow all chats.
	AllowedChatIDs map[int64]struct{}
}

// Router dispatches Telegram text commands to application services.
type Router struct {
	market *market.Service
	watch  *watchlist.Service
	opts   Options
	now    func() time.Time
	mu     sync.Mutex
	lastAt map[int64]time.Time
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
	return &Router{
		market: marketSvc,
		watch:  watchSvc,
		opts:   opts,
		now:    time.Now,
		lastAt: map[int64]time.Time{},
	}
}

// Allowed reports whether chatID may use the bot.
func (r *Router) Allowed(chatID int64) bool {
	if len(r.opts.AllowedChatIDs) == 0 {
		return true
	}
	_, ok := r.opts.AllowedChatIDs[chatID]
	return ok
}

// Handle processes one message text and returns the reply body.
func (r *Router) Handle(ctx context.Context, chatID, userID int64, text string) string {
	if !r.Allowed(chatID) {
		return "This bot is private. Your chat is not allowed."
	}
	if !r.allowRate(chatID) {
		return "Slow down — try again in a second."
	}

	text = strings.TrimSpace(text)
	if text == "" {
		return HelpText()
	}

	parts := strings.Fields(text)
	cmd := strings.ToLower(parts[0])
	if i := strings.IndexByte(cmd, '@'); i > 0 {
		cmd = cmd[:i]
	}
	args := parts[1:]

	switch cmd {
	case "/start", "/help":
		return HelpText()
	case "/price":
		return r.cmdPrice(ctx, args)
	case "/spot":
		return r.cmdSpot(ctx, args)
	case "/lowmcap", "/lowcap":
		return r.cmdLowMcap(ctx, args)
	case "/mcap", "/supply":
		return r.cmdMcap(ctx, args)
	case "/rsi":
		return r.cmdRSI(ctx, args)
	case "/exchanges":
		return r.cmdExchanges()
	case "/watch":
		return r.cmdWatch(ctx, userID, args)
	default:
		if strings.HasPrefix(cmd, "/") {
			return "Unknown command. Try /help"
		}
		return r.cmdPrice(ctx, parts)
	}
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
		return "Usage: /rsi <symbol> [interval] [exchange]\nExample: /rsi BTCUSDT 1h binance"
	}
	symbol := strings.ToUpper(args[0])
	interval := "1h"
	exchange := r.defaultExchange()
	if len(args) >= 2 {
		if isExchange(args[1]) {
			exchange = strings.ToLower(args[1])
		} else {
			interval = args[1]
		}
	}
	if len(args) >= 3 {
		exchange = strings.ToLower(args[2])
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

func (r *Router) cmdWatch(ctx context.Context, userID int64, args []string) string {
	if r.watch == nil {
		return "Watchlist is not configured."
	}
	clientID := fmt.Sprintf("tg-%d", userID)
	if len(args) == 0 {
		wl, err := r.watch.Get(ctx, clientID)
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
		wl, err := r.watch.Add(ctx, clientID, exchange, symbol, "")
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
		wl, err := r.watch.Remove(ctx, clientID, exchange, symbol)
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
	wl, err := r.watch.Get(ctx, clientID)
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
		return "Error: " + err.Error()
	default:
		return "Error: " + err.Error()
	}
}

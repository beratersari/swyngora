package telegram

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

// Disclaimer footer (HTML).
const Disclaimer = "<i>ℹ️ Informational only — not financial advice.</i>"

// Num formats a numeric string for display (with grouping when useful).
func Num(s string, maxFrac int) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return "—"
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return s
	}
	return Float(f, maxFrac)
}

// Float formats a float with optional scientific notation for dust prices.
func Float(f float64, maxFrac int) string {
	if f == 0 {
		return "0"
	}
	abs := f
	if abs < 0 {
		abs = -abs
	}
	if abs > 0 && abs < 1e-6 {
		return strconv.FormatFloat(f, 'e', 3, 64)
	}
	// Group thousands for large magnitudes.
	if abs >= 1000 {
		return formatGrouped(f, maxFrac)
	}
	return strconv.FormatFloat(f, 'f', maxFrac, 64)
}

func formatGrouped(f float64, maxFrac int) string {
	neg := f < 0
	if neg {
		f = -f
	}
	s := strconv.FormatFloat(f, 'f', maxFrac, 64)
	// trim trailing zeros after decimal
	if i := strings.IndexByte(s, '.'); i >= 0 {
		s = strings.TrimRight(strings.TrimRight(s, "0"), ".")
	}
	parts := strings.SplitN(s, ".", 2)
	intPart := parts[0]
	var b strings.Builder
	n := len(intPart)
	for i, c := range intPart {
		if i > 0 && (n-i)%3 == 0 {
			b.WriteByte(',')
		}
		b.WriteRune(c)
	}
	out := b.String()
	if len(parts) == 2 && parts[1] != "" {
		out += "." + parts[1]
	}
	if neg {
		return "-" + out
	}
	return out
}

// CompactMcap shortens large market caps.
func CompactMcap(v *float64) string {
	if v == nil {
		return "—"
	}
	n := *v
	abs := n
	if abs < 0 {
		abs = -abs
	}
	switch {
	case abs >= 1e12:
		return fmt.Sprintf("$%.2fT", n/1e12)
	case abs >= 1e9:
		return fmt.Sprintf("$%.2fB", n/1e9)
	case abs >= 1e6:
		return fmt.Sprintf("$%.2fM", n/1e6)
	case abs >= 1e3:
		return fmt.Sprintf("$%.2fK", n/1e3)
	default:
		return "$" + Float(n, 2)
	}
}

// Pct formats a percent change with sign.
func Pct(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return "—"
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return esc(s) + "%"
	}
	sign := ""
	if f > 0 {
		sign = "+"
	}
	return fmt.Sprintf("%s%s%%", sign, Float(f, 2))
}

// pctLine returns HTML for a percent with direction emoji.
func pctLine(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return "—"
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return code(s + "%")
	}
	emoji := "●"
	switch {
	case f > 0:
		emoji = "🟢"
	case f < 0:
		emoji = "🔴"
	}
	sign := ""
	if f > 0 {
		sign = "+"
	}
	return fmt.Sprintf("%s %s", emoji, code(fmt.Sprintf("%s%s%%", sign, Float(f, 2))))
}

func esc(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	return s
}

func bold(s string) string   { return "<b>" + esc(s) + "</b>" }
func italic(s string) string { return "<i>" + esc(s) + "</i>" }
func code(s string) string   { return "<code>" + esc(s) + "</code>" }

func header(emoji, title string) string {
	return fmt.Sprintf("%s %s\n", emoji, bold(title))
}

func row(label, value string) string {
	return fmt.Sprintf("  %s  %s\n", italic(label), value)
}

func divider() string {
	return "────────────\n"
}

func footer() string {
	return "\n" + divider() + Disclaimer
}

// FormatTicker formats a 24h ticker as a readable card.
func FormatTicker(exchange string, t *domain.Ticker24h) string {
	if t == nil {
		return "No ticker data."
	}
	var b strings.Builder
	b.WriteString(header("📈", t.Symbol+" · "+exchange))
	b.WriteString(divider())
	b.WriteString(row("Last", code(Num(t.LastPrice, 8))))
	b.WriteString(row("24h", pctLine(t.PriceChangePercent)))
	b.WriteString(row("High", code(Num(t.HighPrice, 8))))
	b.WriteString(row("Low", code(Num(t.LowPrice, 8))))
	b.WriteString(row("Volume", code(Num(t.Volume, 4))))
	b.WriteString(row("Quote vol", code(Num(t.QuoteVolume, 2))))
	if t.TradeCount > 0 {
		b.WriteString(row("Trades", code(formatGrouped(float64(t.TradeCount), 0))))
	}
	b.WriteString(footer())
	return b.String()
}

// FormatSpotList formats a ranked list with columns.
func FormatSpotList(title, exchange string, items []domain.SpotMarket, total int) string {
	if len(items) == 0 {
		return bold(title) + "\n\n" + italic("No markets found.")
	}
	var b strings.Builder
	b.WriteString(header("🏆", title))
	fmt.Fprintf(&b, "%s · showing %s of %s\n",
		code(exchange), code(strconv.Itoa(len(items))), code(strconv.Itoa(total)))
	b.WriteString(divider())
	for i, m := range items {
		rank := fmt.Sprintf("%d.", i+1)
		fmt.Fprintf(&b, "%s %s\n", bold(rank), code(m.Symbol))
		fmt.Fprintf(&b, "    %s  %s  mcap %s\n",
			code(Num(m.LastPrice, 8)),
			pctLine(m.PriceChangePercent),
			code(CompactMcap(m.MarketCapCirculating)),
		)
		if i < len(items)-1 {
			b.WriteString("\n")
		}
	}
	b.WriteString(footer())
	return b.String()
}

// FormatLowMcapAll formats multi-exchange lowest mcap sections.
func FormatLowMcapAll(limit int, sections []LowMcapSection) string {
	var b strings.Builder
	b.WriteString(header("💎", fmt.Sprintf("Lowest circ. mcap · %d each", limit)))
	b.WriteString(divider())
	for si, sec := range sections {
		if sec.Err != "" {
			fmt.Fprintf(&b, "%s %s\n%s\n\n", bold(strings.ToUpper(sec.Exchange)), italic("error"), esc(sec.Err))
			continue
		}
		fmt.Fprintf(&b, "%s  %s\n", bold(strings.ToUpper(sec.Exchange)), italic(fmt.Sprintf("%d of %d", len(sec.Items), sec.Total)))
		for i, m := range sec.Items {
			if i >= limit {
				break
			}
			fmt.Fprintf(&b, "  %s %s  %s  %s\n",
				bold(fmt.Sprintf("%d.", i+1)),
				code(m.Symbol),
				code(Num(m.LastPrice, 6)),
				code(CompactMcap(m.MarketCapCirculating)),
			)
		}
		if si < len(sections)-1 {
			b.WriteString("\n")
		}
	}
	b.WriteString(footer())
	return b.String()
}

// LowMcapSection is one exchange block for FormatLowMcapAll.
type LowMcapSection struct {
	Exchange string
	Items    []domain.SpotMarket
	Total    int
	Err      string
}

// FormatSupply formats supply snapshot as a card.
func FormatSupply(s *domain.AssetSupply) string {
	if s == nil {
		return "No supply data."
	}
	name := s.Name
	if name == "" {
		name = "—"
	}
	var b strings.Builder
	b.WriteString(header("🪙", s.Asset))
	fmt.Fprintf(&b, "%s\n", italic(name))
	b.WriteString(divider())
	b.WriteString(row("Circulating", code(ptrFloat(s.CirculatingSupply, 0))))
	b.WriteString(row("Total", code(ptrFloat(s.TotalSupply, 0))))
	b.WriteString(row("Max", code(ptrFloat(s.MaxSupply, 0))))
	if s.CurrentPriceUSD != nil {
		b.WriteString(row("Price USD", code(Float(*s.CurrentPriceUSD, 4))))
	}
	if !s.AsOf.IsZero() {
		b.WriteString(row("As of", code(s.AsOf.UTC().Format(time.RFC3339))))
		b.WriteString(row("Note", italic("daily snapshot")))
	}
	b.WriteString(footer())
	return b.String()
}

// FormatIndicators formats RSI/EMA latest.
func FormatIndicators(ser *domain.IndicatorSeries) string {
	if ser == nil {
		return "No indicators."
	}
	var b strings.Builder
	b.WriteString(header("📉", ser.Symbol))
	fmt.Fprintf(&b, "%s · %s\n", code(string(ser.Exchange)), code(string(ser.Interval)))
	b.WriteString(divider())
	rsi := "—"
	rsiNote := ""
	if ser.LatestRSI != nil {
		rsi = Float(*ser.LatestRSI, 2)
		switch {
		case *ser.LatestRSI >= 70:
			rsiNote = "  " + italic("overbought zone")
		case *ser.LatestRSI <= 30:
			rsiNote = "  " + italic("oversold zone")
		}
	}
	fmt.Fprintf(&b, "  %s  %s%s\n", italic(fmt.Sprintf("RSI(%d)", ser.RSIPeriod)), code(rsi), rsiNote)
	e12, e26 := "—", "—"
	if v := ser.LatestEMA[12]; v != nil {
		e12 = Float(*v, 6)
	}
	if v := ser.LatestEMA[26]; v != nil {
		e26 = Float(*v, 6)
	}
	b.WriteString(row("EMA 12", code(e12)))
	b.WriteString(row("EMA 26", code(e26)))
	b.WriteString(footer())
	return b.String()
}

// FormatWatchlist formats watchlist items.
func FormatWatchlist(items []domain.WatchlistItem) string {
	if len(items) == 0 {
		var b strings.Builder
		b.WriteString(header("⭐", "Watchlist"))
		b.WriteString(divider())
		b.WriteString(italic("Empty.") + "\n\n")
		b.WriteString("Add with:\n")
		b.WriteString(code("/watch add BTCUSDT") + "\n")
		b.WriteString(code("/watch add BTC-USD coinbase") + "\n")
		return b.String()
	}
	var b strings.Builder
	b.WriteString(header("⭐", fmt.Sprintf("Watchlist · %d", len(items))))
	b.WriteString(divider())
	for i, it := range items {
		fmt.Fprintf(&b, "  %s  %s  ·  %s\n",
			bold(fmt.Sprintf("%d.", i+1)),
			code(it.Symbol),
			italic(string(it.Exchange)),
		)
	}
	b.WriteString("\n")
	b.WriteString(italic("Tip: ") + code("/watch top") + italic(" for live prices"))
	return b.String()
}

// FormatWatchTop formats priced watchlist rows.
func FormatWatchTop(lines []string, total, shown int) string {
	var b strings.Builder
	b.WriteString(header("⭐", "Watchlist prices"))
	b.WriteString(divider())
	for _, line := range lines {
		b.WriteString(line)
		if !strings.HasSuffix(line, "\n") {
			b.WriteByte('\n')
		}
	}
	if total > shown {
		fmt.Fprintf(&b, "\n%s\n", italic(fmt.Sprintf("Showing first %d of %d.", shown, total)))
	}
	b.WriteString(footer())
	return b.String()
}

// FormatWatchTopRow is one line for /watch top.
func FormatWatchTopRow(i int, symbol, exchange, last, pct string, errMsg string) string {
	if errMsg != "" {
		return fmt.Sprintf("  %s  %s  ·  %s\n      %s\n",
			bold(fmt.Sprintf("%d.", i)), code(symbol), italic(exchange), italic(errMsg))
	}
	return fmt.Sprintf("  %s  %s  ·  %s\n      %s  %s\n",
		bold(fmt.Sprintf("%d.", i)), code(symbol), italic(exchange),
		code(Num(last, 8)), pctLine(pct))
}

// FormatExchanges formats venue list.
func FormatExchanges(names []string, def string) string {
	var b strings.Builder
	b.WriteString(header("🏦", "Exchanges"))
	b.WriteString(divider())
	for _, n := range names {
		mark := ""
		if n == def {
			mark = "  " + italic("(default)")
		}
		fmt.Fprintf(&b, "  • %s%s\n", code(n), mark)
	}
	b.WriteString("\n")
	b.WriteString(italic("Example: ") + code("/price BTCUSDT bybit"))
	return b.String()
}

// HelpText is the /help body (HTML).
func HelpText() string {
	var b strings.Builder
	b.WriteString(header("🤖", "Swyngora"))
	b.WriteString(italic("Market data in Telegram") + "\n")
	b.WriteString(divider())

	b.WriteString(bold("Market") + "\n")
	b.WriteString(cmdLine("/price", "<symbol> [exchange]", "24h ticker"))
	b.WriteString(cmdLine("/spot", "[exchange] [query]", "top by quote volume"))
	b.WriteString(cmdLine("/lowmcap", "[exchange|all] [n]", "lowest circ. mcap"))
	b.WriteString(cmdLine("/mcap", "<asset|pair>", "supply snapshot"))
	b.WriteString(cmdLine("/rsi", "<symbol> [interval] [ex]", "RSI + EMA"))
	b.WriteString(cmdLine("/exchanges", "", "list venues"))
	b.WriteString("\n")

	b.WriteString(bold("AI assistant") + "\n")
	b.WriteString(cmdLine("/ask", "<question>", "multi-agent AI (market + web + X)"))
	b.WriteString(cmdLine("/ai", "<question>", "alias of /ask"))
	b.WriteString("\n")

	b.WriteString(bold("Watchlist") + "\n")
	b.WriteString(cmdLine("/watch", "", "show list"))
	b.WriteString(cmdLine("/watch add", "<symbol> [exchange]", "add pair"))
	b.WriteString(cmdLine("/watch del", "<symbol> [exchange]", "remove"))
	b.WriteString(cmdLine("/watch top", "", "live prices"))
	b.WriteString("\n")

	b.WriteString(bold("Examples") + "\n")
	b.WriteString("  " + code("/price BTCUSDT") + "\n")
	b.WriteString("  " + code("/price BTC-USD coinbase") + "\n")
	b.WriteString("  " + code("/lowmcap all 5") + "\n")
	b.WriteString("  " + code("/spot bybit SOL") + "\n")
	b.WriteString("  " + code("/rsi ETHUSDT 1h") + "\n")
	b.WriteString("  " + code("/ask What is BTC RSI and recent news?") + "\n")
	b.WriteString("\n")
	b.WriteString(italic("Venues: ") + code("binance") + ", " + code("coinbase") + ", " + code("bybit"))
	b.WriteString(footer())
	return b.String()
}

func cmdLine(cmd, args, desc string) string {
	if args == "" {
		return fmt.Sprintf("  %s\n    %s\n", code(cmd), italic(desc))
	}
	return fmt.Sprintf("  %s %s\n    %s\n", code(cmd), italic(args), italic(desc))
}

// Main specialist agents shown in Telegram progress (not leaf tools).
var mainAIAgentNames = []string{
	"market_agent",
	"web_agent",
	"x_agent",
	"analyst_agent",
}

// IsMainAITool reports orchestrator-level specialists only (hides get_ticker, etc.).
func IsMainAITool(line string) bool {
	line = strings.TrimSpace(line)
	if line == "" {
		return false
	}
	// Nested leaf calls look like "market_agent → get_ticker(...)" or "↳ get_ticker".
	if strings.Contains(line, "→") || strings.Contains(line, "↳") {
		return false
	}
	lower := strings.ToLower(line)
	for _, name := range mainAIAgentNames {
		if strings.Contains(lower, name) {
			return true
		}
	}
	return false
}

// ShortMainToolLabel maps a tool event line to a short human label for Telegram.
func ShortMainToolLabel(line string) string {
	lower := strings.ToLower(strings.TrimSpace(line))
	switch {
	case strings.Contains(lower, "market_agent"):
		return "Market"
	case strings.Contains(lower, "web_agent"):
		return "Web"
	case strings.Contains(lower, "x_agent"):
		return "X / social"
	case strings.Contains(lower, "analyst_agent"):
		return "Analyst"
	default:
		return clipRunes(strings.TrimSpace(line), 40)
	}
}

// FilterMainAITools keeps unique main-agent labels in order (max a few).
func FilterMainAITools(tools []string) []string {
	var out []string
	seen := map[string]struct{}{}
	for _, t := range tools {
		if !IsMainAITool(t) {
			// Allow already-short labels from live stream.
			label := strings.TrimSpace(t)
			switch strings.ToLower(label) {
			case "market", "web", "x / social", "x/social", "analyst":
				// keep
			default:
				continue
			}
		} else {
			label := ShortMainToolLabel(t)
			if _, ok := seen[label]; ok {
				continue
			}
			seen[label] = struct{}{}
			out = append(out, label)
			if len(out) >= 6 {
				break
			}
			continue
		}
		label := ShortMainToolLabel(t)
		// normalize short labels
		switch strings.ToLower(label) {
		case "x/social":
			label = "X / social"
		}
		if _, ok := seen[label]; ok {
			continue
		}
		seen[label] = struct{}{}
		out = append(out, label)
		if len(out) >= 6 {
			break
		}
	}
	return out
}

// IsMainAIStatus keeps progress status readable (planning / specialist / done).
func IsMainAIStatus(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" {
		return false
	}
	lower := strings.ToLower(s)
	if strings.Contains(lower, "planning") ||
		strings.Contains(lower, "compos") ||
		strings.Contains(lower, "orchestrat") ||
		strings.Contains(lower, "synthes") {
		return true
	}
	for _, name := range mainAIAgentNames {
		if strings.Contains(lower, name) {
			return true
		}
	}
	// Short human status lines we set ourselves
	for _, p := range []string{"market", "web", "social", "analyst", "running"} {
		if strings.Contains(lower, p) && len(s) < 80 {
			return true
		}
	}
	return false
}

// FormatAIProgress is a live status card (HTML; all dynamic text is escaped).
// tools should already be main-agent labels only (Market, Web, …).
func FormatAIProgress(status string, tools []string) string {
	tools = FilterMainAITools(tools)
	var b strings.Builder
	b.WriteString(header("⏳", "Swyngora AI · working"))
	b.WriteString(divider())
	if status != "" {
		b.WriteString(italic(status) + "\n")
	}
	if len(tools) > 0 {
		b.WriteString("\n" + bold("Agents") + "\n")
		for _, t := range tools {
			fmt.Fprintf(&b, "  • %s\n", code(t))
		}
	} else {
		b.WriteString("\n" + italic("Planning which agents to run…") + "\n")
	}
	return b.String()
}

// FormatAIAnswer is the final answer card (HTML; dynamic content escaped).
// Only main agents are listed (not every leaf tool).
func FormatAIAnswer(reply string, thinking, tools []string) string {
	var b strings.Builder
	b.WriteString(header("🤖", "Swyngora AI"))
	b.WriteString(divider())
	body := stripLightMarkdown(strings.TrimSpace(reply))
	if body == "" {
		body = "No answer produced. Try rephrasing your question."
	}
	// Escaped plain body keeps Telegram HTML valid even if the model used < or &.
	b.WriteString(esc(body))
	main := FilterMainAITools(tools)
	if len(main) > 0 {
		b.WriteString("\n\n")
		b.WriteString(divider())
		b.WriteString(italic("Agents: ") + code(strings.Join(main, ", ")))
		b.WriteString("\n")
	}
	b.WriteString(footer())
	return b.String()
}

// stripLightMarkdown removes common markdown markers so plain/HTML cards stay clean.
func stripLightMarkdown(s string) string {
	repl := strings.NewReplacer(
		"**", "",
		"__", "",
		"```", "",
		"``", "",
		"`", "",
		"### ", "",
		"## ", "",
		"# ", "",
		"~~", "",
	)
	return strings.TrimSpace(repl.Replace(s))
}

func clipRunes(s string, max int) string {
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	if max < 2 {
		return string(r[:max])
	}
	return string(r[:max-1]) + "…"
}

func ptrFloat(v *float64, frac int) string {
	if v == nil {
		return "—"
	}
	return Float(*v, frac)
}

// PlainText strips HTML tags for plain fallback sends.
func PlainText(html string) string {
	repl := strings.NewReplacer(
		"<b>", "", "</b>", "",
		"<i>", "", "</i>", "",
		"<code>", "", "</code>", "",
		"<pre>", "", "</pre>", "",
		"&amp;", "&", "&lt;", "<", "&gt;", ">",
		"&quot;", "\"",
	)
	return repl.Replace(html)
}

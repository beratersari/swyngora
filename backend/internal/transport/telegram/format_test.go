package telegram

import (
	"strings"
	"testing"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

func TestPct(t *testing.T) {
	if Pct("1.5") != "+1.50%" {
		t.Fatalf("%s", Pct("1.5"))
	}
}

func TestCompactMcap(t *testing.T) {
	v := 1.5e9
	if CompactMcap(&v) != "$1.50B" {
		t.Fatalf("%s", CompactMcap(&v))
	}
}

func TestFormatTicker(t *testing.T) {
	s := FormatTicker("binance", &domain.Ticker24h{Symbol: "BTCUSDT", LastPrice: "100", PriceChangePercent: "1"})
	if !strings.Contains(s, "BTCUSDT") || !strings.Contains(s, "<b>") || !strings.Contains(s, "Informational") {
		t.Fatalf("%s", s)
	}
}

func TestHelpHasLowmcap(t *testing.T) {
	help := HelpText()
	if !strings.Contains(help, "/lowmcap") || !strings.Contains(help, "<code>") {
		t.Fatal("missing lowmcap or HTML")
	}
	if !strings.Contains(help, "/ask") {
		t.Fatal("help must document /ask AI command")
	}
}

func TestPlainTextStripsTags(t *testing.T) {
	p := PlainText("<b>Hi</b> <code>x</code>")
	if strings.Contains(p, "<") || !strings.Contains(p, "Hi") {
		t.Fatalf("%q", p)
	}
}

func TestFormatAIProgressHTML(t *testing.T) {
	// Leaf tools must be filtered; only main agents shown.
	s := FormatAIProgress("Planning…", []string{
		`get_ticker(symbol=BTCUSDT)`,
		`market_agent(task=price)`,
		`market_agent → get_indicators(...)`,
		`web_agent(task=news)`,
	})
	if !strings.Contains(s, "<b>") || !strings.Contains(s, "Agents") {
		t.Fatalf("expected HTML agents card: %s", s)
	}
	if strings.Contains(s, "get_ticker") || strings.Contains(s, "get_indicators") {
		t.Fatalf("leaf tools must not appear: %s", s)
	}
	if !strings.Contains(s, "Market") || !strings.Contains(s, "Web") {
		t.Fatalf("expected main agents: %s", s)
	}
}

func TestFormatAIAnswerEscapesAndFocuses(t *testing.T) {
	s := FormatAIAnswer("RSI is **45** for <BTC>", []string{"think hard"}, []string{
		"market_agent(task=x)",
		"get_ticker(symbol=BTC)",
		"x_agent(task=y)",
	})
	if !strings.Contains(s, "Swyngora AI") {
		t.Fatal(s)
	}
	// Angle brackets from model must not break HTML.
	if strings.Contains(s, "<BTC>") {
		t.Fatalf("must escape model angle brackets: %s", s)
	}
	if !strings.Contains(s, "&lt;BTC&gt;") {
		t.Fatalf("expected escaped BTC: %s", s)
	}
	// Markdown bold markers stripped.
	if strings.Contains(s, "**") {
		t.Fatalf("markdown markers should be stripped: %s", s)
	}
	// Thinking should not dump into final card.
	if strings.Contains(s, "think hard") {
		t.Fatalf("thinking should not appear in final answer: %s", s)
	}
	// Only main agents in footer.
	if !strings.Contains(s, "Market") || !strings.Contains(s, "X / social") {
		t.Fatalf("expected main agent summary: %s", s)
	}
	if strings.Contains(s, "get_ticker") {
		t.Fatalf("leaf tools must not appear in answer: %s", s)
	}
}

func TestFormatAIReferences(t *testing.T) {
	s := FormatAIReferences([]RefLink{
		{Title: "Bitcoin", URL: "https://coinmarketcap.com/currencies/bitcoin/"},
	})
	if !strings.Contains(s, "Sources") || !strings.Contains(s, "coinmarketcap.com") {
		t.Fatalf("got %s", s)
	}
}

func TestFilterMainAITools(t *testing.T) {
	got := FilterMainAITools([]string{
		"market_agent(task=a)",
		"market_agent → get_ticker(x)",
		"↳ get_ticker → ok",
		"web_agent(task=b)",
		"Market", // already short
	})
	if len(got) != 2 { // Market, Web (deduped)
		t.Fatalf("got %v", got)
	}
}

func TestFormatGroupedThousands(t *testing.T) {
	if Float(1234567.8, 1) != "1,234,567.8" {
		t.Fatalf("%s", Float(1234567.8, 1))
	}
}

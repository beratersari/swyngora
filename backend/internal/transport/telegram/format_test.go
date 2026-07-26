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
	if !strings.Contains(HelpText(), "/lowmcap") || !strings.Contains(HelpText(), "<code>") {
		t.Fatal("missing lowmcap or HTML")
	}
}

func TestPlainTextStripsTags(t *testing.T) {
	p := PlainText("<b>Hi</b> <code>x</code>")
	if strings.Contains(p, "<") || !strings.Contains(p, "Hi") {
		t.Fatalf("%q", p)
	}
}

func TestFormatGroupedThousands(t *testing.T) {
	if Float(1234567.8, 1) != "1,234,567.8" {
		t.Fatalf("%s", Float(1234567.8, 1))
	}
}

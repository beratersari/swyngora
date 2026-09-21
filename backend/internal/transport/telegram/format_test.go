package telegram

import (
	"strings"
	"testing"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

func TestPct(t *testing.T) {
	if Pct("1.5") != "+1.50%" {
		t.Fatalf("%s", Pct("1.5"))
	}
}

func TestFormatLongShort_Gap(t *testing.T) {
	now := time.Date(2026, 9, 20, 16, 0, 0, 0, time.UTC)
	out := FormatLongShort(&domain.LongShortSnapshot{
		Symbol: "BTCUSDT", Exchange: "binance", Period: "5m",
		Venues: []domain.LongShortVenueSnap{{
			Exchange:     "binance",
			Current:      domain.LongShortLevel{LongPct: "55.00", ShortPct: "45.00", Ratio: "1.2222", Bias: "long"},
			TopPositions: &domain.LongShortLevel{LongPct: "65.00", ShortPct: "35.00", Bias: "long"},
			Gap: &domain.LongShortGap{
				GapPct: "+10", Lean: domain.LongShortGapLeanTopLong,
				Headline: "Top traders are 10pp more long than other accounts.",
			},
			GapWindows: []domain.LongShortGapWindow{
				{Window: domain.SnapshotWindow1h, Kind: domain.LongShortGapWidened, ChangePct: "+6"},
			},
		}},
		AsOf: now,
	})
	if !strings.Contains(out, "+10") || !strings.Contains(out, "more long") || !strings.Contains(out, "widened") {
		t.Fatalf("%s", out)
	}
}

func TestFormatLongShort_PriceDivergence(t *testing.T) {
	out := FormatLongShort(&domain.LongShortSnapshot{
		Symbol: "BTCUSDT", Exchange: "binance", Period: "5m",
		Venues: []domain.LongShortVenueSnap{{
			Exchange: "binance",
			Current:  domain.LongShortLevel{LongPct: "55", ShortPct: "45", Ratio: "1.22", Bias: "long"},
			Divergence: &domain.LongShortPriceDivergence{
				Kind:     domain.LongShortPriceDivTopLongPriceDown,
				Duration: "30m",
				Since:    time.Date(2026, 9, 20, 15, 30, 0, 0, time.UTC),
				Headline: "Top traders got more long while price dropped 1.5% over 30m.",
			},
		}},
	})
	if !strings.Contains(out, "dropped 1.5%") || !strings.Contains(out, "30m") || !strings.Contains(out, "15:30 UTC") {
		t.Fatalf("%s", out)
	}
}

func TestCompactMcap(t *testing.T) {
	v := 1.5e9
	if CompactMcap(&v) != "$1.50B" {
		t.Fatalf("%s", CompactMcap(&v))
	}
}

func TestFormatRSIHeatmap(t *testing.T) {
	if FormatRSIHeatmap(nil) != "No RSI heatmap rows." {
		t.Fatal("nil")
	}
	if FormatRSIHeatmap(&domain.RSIHeatmap{}) != "No RSI heatmap rows." {
		t.Fatal("empty")
	}
	low, mid, high := 22.0, 50.0, 81.0
	out := FormatRSIHeatmap(&domain.RSIHeatmap{
		Exchange: domain.ExchangeBinance, Quote: "USDT", Interval: "1h", Period: 14,
		AverageRSI: &mid, OversoldCount: 1, OverboughtCount: 1,
		Items: []domain.RSIHeatmapRow{
			{Rank: 1, Base: "AAA", RSI: &low, Zone: domain.RSIZoneOversold},
			{Rank: 2, Symbol: "BBBUSDT", RSI: &high, Zone: domain.RSIZoneOverbought},
			{Rank: 3, Base: "CCC", RSI: &mid, Zone: domain.RSIZoneNeutral},
			{Rank: 4, Base: "DDD"},
		},
	})
	if !strings.Contains(out, "RSI heatmap") || !strings.Contains(out, "AAA") || !strings.Contains(out, "BBBUSDT") {
		t.Fatalf("%s", out)
	}
	if !strings.Contains(out, "avg") || !strings.Contains(out, "oversold 1") {
		t.Fatalf("%s", out)
	}
	items := make([]domain.RSIHeatmapRow, 0, 25)
	for i := 0; i < 25; i++ {
		v := 20.0
		items = append(items, domain.RSIHeatmapRow{Rank: i + 1, Base: "X", RSI: &v, Zone: domain.RSIZoneOversold})
	}
	long := FormatRSIHeatmap(&domain.RSIHeatmap{
		Exchange: domain.ExchangeBinance, Quote: "USDT", Interval: "1h", Period: 14,
		Items: items,
	})
	if !strings.Contains(long, "X") {
		t.Fatalf("%s", long)
	}
	neutral := 50.0
	skip := make([]domain.RSIHeatmapRow, 0, 16)
	for i := 0; i < 12; i++ {
		v := 20.0
		skip = append(skip, domain.RSIHeatmapRow{Rank: i + 1, Base: "OS", RSI: &v, Zone: domain.RSIZoneOversold})
	}
	skip = append(skip, domain.RSIHeatmapRow{Rank: 13, Base: "NEUT", RSI: &neutral, Zone: domain.RSIZoneNeutral})
	skipped := FormatRSIHeatmap(&domain.RSIHeatmap{
		Exchange: domain.ExchangeBinance, Quote: "USDT", Interval: "1h", Period: 14, Items: skip,
	})
	if strings.Contains(skipped, "NEUT") {
		t.Fatalf("neutral after 12 rows must be omitted: %s", skipped)
	}
}

func TestFormatTicker(t *testing.T) {
	s := FormatTicker("binance", &domain.Ticker24h{Symbol: "BTCUSDT", LastPrice: "100", PriceChangePercent: "1"})
	if !strings.Contains(s, "BTCUSDT") || !strings.Contains(s, "<b>") || !strings.Contains(s, "Informational") {
		t.Fatalf("%s", s)
	}
}

func TestFormatFunding(t *testing.T) {
	s := &domain.FundingSnapshot{
		Symbol: "BTCUSDT", Exchange: "binance", VenueCount: 1,
		Current: &domain.FundingCurrent{Rate: "0.0001", RatePct: "0.01", Payer: "long", IntervalHours: 8},
		Venues: []domain.FundingVenueSnap{{
			Exchange: "binance",
			Current:  domain.FundingCurrent{Rate: "0.0001", RatePct: "0.01", Payer: "long", IntervalHours: 8},
		}},
	}
	got := FormatFunding(s)
	if !strings.Contains(got, "BTCUSDT") || !strings.Contains(got, "0.01") {
		t.Fatalf("%s", got)
	}
}

func TestFormatFunding_Pressure(t *testing.T) {
	s := &domain.FundingSnapshot{
		Symbol: "BTCUSDT", Exchange: "all", VenueCount: 2,
		Pressure: &domain.FundingPressure{
			RatePct: "0.007778", Payer: "long",
			Summary: "Binance 88.9% of OI at 0.01%; Bybit 11.1% of OI at -0.01%. Combined pressure 0.007778% (longs pay).",
		},
		PressureWindows: []domain.FundingPressureWindow{
			{Window: "1h", ChangePct: "+0.001", Kind: "steeper"},
		},
		Venues: []domain.FundingVenueSnap{{
			Exchange: "binance",
			Current:  domain.FundingCurrent{RatePct: "0.01", Payer: "long"},
		}},
	}
	got := FormatFunding(s)
	if !strings.Contains(got, "OI-weighted") || !strings.Contains(got, "0.007778") || !strings.Contains(got, "88.9%") {
		t.Fatalf("%s", got)
	}
}

func TestFormatOpenInterest(t *testing.T) {
	s := &domain.OpenInterestSnapshot{
		Symbol: "BTCUSDT", Exchange: "all", Unit: "BTC", VenueCount: 2,
		Current: domain.OpenInterestLevel{Contracts: "150", Value: "15000000000"},
		Windows: []domain.OpenInterestWindow{
			{Window: "1h", Change: "+20", ChangePct: "+0.15", ChangeValue: "+2000000", ChangeValuePct: "+14.3", Direction: "up", Flow: "opened", OpenedValue: "2000000", Complete: true},
		},
		Venues: []domain.OpenInterestVenueSnap{{
			Exchange: "binance",
			Current:  domain.OpenInterestLevel{Value: "8000000000"},
			Windows: []domain.OpenInterestWindow{
				{Window: "1h", ChangeValue: "+1500000", ChangeValuePct: "+10", Flow: "opened"},
			},
		}},
	}
	got := FormatOpenInterest(s)
	if !strings.Contains(got, "BTCUSDT") || !strings.Contains(got, "opened") || !strings.Contains(got, "$15.00B") {
		t.Fatalf("%s", got)
	}
	if !strings.Contains(got, "binance") {
		t.Fatalf("venues %s", got)
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
	if !strings.Contains(help, "/oi") {
		t.Fatal("help must document /oi open interest")
	}
	if !strings.Contains(help, "/perpliq") {
		t.Fatal("help must document /perpliq")
	}
	if !strings.Contains(help, "/funding") {
		t.Fatal("help must document /funding")
	}
	if !strings.Contains(help, "/fundingarb") {
		t.Fatal("help must document /fundingarb")
	}
	if !strings.Contains(help, "/ls") {
		t.Fatal("help must document /ls long/short")
	}
	if !strings.Contains(help, "/portfolio") || !strings.Contains(help, "/buy") || !strings.Contains(help, "/sell") {
		t.Fatal("help must document paper portfolio commands")
	}
	if !strings.Contains(help, "/deposit") || !strings.Contains(help, "/withdraw") || !strings.Contains(help, "/cash") {
		t.Fatal("help must document paper cash commands")
	}
	if !strings.Contains(help, "/transfer") {
		t.Fatal("help must document paper transfer")
	}
}

func TestFormatFundingArb(t *testing.T) {
	out := FormatFundingArb(&domain.FundingArbReport{
		Symbol: "BTCUSDT", Notional: "10000", HoldHours: "24",
		Trade: &domain.FundingArbTradeView{
			Title:                    "Long Binance, short Bybit",
			LongExchange:             "binance",
			ShortExchange:            "bybit",
			LongRatePct:              "0.01",
			ShortRatePct:             "0.04",
			NextFundingAmount:        "+3",
			HorizonFundingAmount:     "+9",
			RoundTripFeeAmount:       "40",
			NetHorizonAfterRoundTrip: "-31",
		},
	})
	if !strings.Contains(out, "BTCUSDT") || !strings.Contains(out, "binance") || !strings.Contains(out, "bybit") {
		t.Fatalf("%s", out)
	}
}

func TestFormatPaperPortfolioAndPreview(t *testing.T) {
	v := &domain.PortfolioView{
		Currency: "USDT", StartingBalance: 10000, CashBalance: 9000,
		AvailableCash: 9000, Equity: 11000, UnrealizedPnL: 1000, TotalPnL: 1000,
		Positions: []domain.PositionView{{
			Symbol: "BTCUSDT", Quantity: 0.1, AvgCost: 100, MarkPrice: 110, UnrealizedPnL: 1,
		}},
	}
	s := FormatPaperPortfolio(v)
	if !strings.Contains(s, "BTCUSDT") || !strings.Contains(s, "Paper") {
		t.Fatalf("%s", s)
	}
	p := FormatTradePreview(domain.TradeSideBuy, "binance", "ETHUSDT", 2, 50, 100, "USDT")
	if !strings.Contains(p, "ETHUSDT") || !strings.Contains(p, "BUY") || !strings.Contains(p, "100") {
		t.Fatalf("%s", p)
	}
	c := FormatTradeCanceled(domain.TradeSideSell, "ETHUSDT")
	if !strings.Contains(c, "Canceled") || !strings.Contains(c, "ETHUSDT") {
		t.Fatalf("%s", c)
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

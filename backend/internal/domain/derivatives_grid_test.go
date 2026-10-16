package domain

import (
	"math"
	"strings"
	"testing"
	"time"
)

func TestParseDerivativesGrid(t *testing.T) {
	got, err := ParseDerivativesGrid("funding", "")
	if err != nil || got.Metric != GridMetricFunding || got.Range != GridRange24h || got.Step != time.Hour {
		t.Fatalf("%+v %v", got, err)
	}
	got, err = ParseDerivativesGrid("open-interest", "7d")
	if err != nil || got.Metric != GridMetricOI || got.Step != 24*time.Hour {
		t.Fatalf("%+v %v", got, err)
	}
	if _, err = ParseDerivativesGrid("price", "24h"); err == nil {
		t.Fatal("expected bad metric")
	}
	if _, err = ParseDerivativesGrid("cvd", "30d"); err == nil {
		t.Fatal("expected bad range")
	}
}

func TestGridColumns_ClockHoursAndUTCDays(t *testing.T) {
	now := time.Date(2026, 10, 10, 15, 40, 0, 0, time.UTC)
	hours := GridColumns(GridSpec{Range: GridRange24h, Step: time.Hour}, now)
	if len(hours) != 24 || !hours[0].Equal(time.Date(2026, 10, 9, 16, 0, 0, 0, time.UTC)) || !hours[23].Equal(time.Date(2026, 10, 10, 15, 0, 0, 0, time.UTC)) {
		t.Fatalf("hours %v", hours)
	}
	days := GridColumns(GridSpec{Range: GridRange7d, Step: 24 * time.Hour}, now)
	if len(days) != 7 || !days[6].Equal(time.Date(2026, 10, 10, 0, 0, 0, 0, time.UTC)) || !days[0].Equal(time.Date(2026, 10, 4, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("days %v", days)
	}
}

func TestBuildDerivativesGrid_FundingWithholdsMissingVenue(t *testing.T) {
	now := time.Date(2026, 10, 10, 15, 40, 0, 0, time.UTC)
	col := time.Date(2026, 10, 10, 14, 0, 0, 0, time.UTC)
	at := col.Add(30 * time.Minute)
	rows := []GridSnapshot{
		{Metric: GridMetricFunding, Exchange: ExchangeBinance, Symbol: "BTCUSDT", Bucket: col, SampledAt: at, FundingRate: 0.0001},
		{Metric: GridMetricFunding, Exchange: ExchangeBybit, Symbol: "BTCUSDT", Bucket: col, SampledAt: at, FundingRate: 0.0003},
	}
	got := BuildDerivativesGrid(GridBuildInput{
		Exchange: "all", Now: now, Symbols: []string{"BTCUSDT"},
		Spec:      GridSpec{Metric: GridMetricFunding, Range: GridRange24h, Step: time.Hour},
		Snapshots: rows,
	})
	cell := gridTestCell(t, got, "BTCUSDT", col)
	if cell.Definite || cell.Text != "" || len(cell.Missing) != 1 || cell.Missing[0] != "okx" {
		t.Fatalf("cell %+v", cell)
	}
	if len(cell.Venues) != 3 {
		t.Fatalf("venues %+v", cell.Venues)
	}
	binance := requireGridVenue(t, cell, "binance", true, 0.0001)
	if binance.Text != "+0.01%" {
		t.Fatalf("binance text %s", binance.Text)
	}
	bybit := requireGridVenue(t, cell, "bybit", true, 0.0003)
	if bybit.Text != "+0.03%" {
		t.Fatalf("bybit text %s", bybit.Text)
	}
	requireGridVenue(t, cell, "okx", false, 0)
}

func TestBuildDerivativesGrid_FundingUsesSettlementAndOIWeight(t *testing.T) {
	now := time.Date(2026, 10, 10, 15, 40, 0, 0, time.UTC)
	col := time.Date(2026, 10, 10, 14, 0, 0, 0, time.UTC)
	at := col.Add(20 * time.Minute)
	var rows []GridSnapshot
	for _, ex := range []Exchange{ExchangeBinance, ExchangeBybit, ExchangeOKX} {
		rows = append(rows,
			GridSnapshot{Metric: GridMetricFunding, Exchange: ex, Symbol: "BTCUSDT", Bucket: col, SampledAt: at, Predicted: true, FundingRate: 0.01},
			GridSnapshot{Metric: GridMetricFunding, Exchange: ex, Symbol: "BTCUSDT", Bucket: col, SampledAt: at.Add(time.Minute), FundingRate: 0.0002},
		)
	}
	rows = append(rows,
		GridSnapshot{Metric: GridMetricOI, Exchange: ExchangeBinance, Symbol: "BTCUSDT", Bucket: col, SampledAt: at, Value: 8},
		GridSnapshot{Metric: GridMetricOI, Exchange: ExchangeBybit, Symbol: "BTCUSDT", Bucket: col, SampledAt: at, Value: 1},
		GridSnapshot{Metric: GridMetricOI, Exchange: ExchangeOKX, Symbol: "BTCUSDT", Bucket: col, SampledAt: at, Value: 1},
	)
	got := BuildDerivativesGrid(GridBuildInput{
		Exchange: "all", Now: now, Symbols: []string{"BTCUSDT"},
		Spec:      GridSpec{Metric: GridMetricFunding, Range: GridRange24h, Step: time.Hour},
		Snapshots: rows,
	})
	cell := gridTestCell(t, got, "BTCUSDT", col)
	want := (0.0002*8 + 0.0002*1 + 0.0002*1) / 10
	if !cell.Definite || cell.Weight != "oi" || math.Abs(cell.Value-want) > 1e-12 {
		t.Fatalf("cell %+v want %v", cell, want)
	}
	for _, ex := range []string{"binance", "bybit", "okx"} {
		venue := requireGridVenue(t, cell, ex, true, 0.0002)
		if venue.Text != "+0.02%" {
			t.Fatalf("%s text %s", ex, venue.Text)
		}
	}
}

func TestBuildDerivativesGrid_OpenFundingSampleGoesStale(t *testing.T) {
	now := time.Date(2026, 10, 10, 15, 40, 0, 0, time.UTC)
	col := time.Date(2026, 10, 10, 15, 0, 0, 0, time.UTC)
	got := BuildDerivativesGrid(GridBuildInput{
		Exchange: "binance", Now: now, Symbols: []string{"BTCUSDT"},
		Spec: GridSpec{Metric: GridMetricFunding, Range: GridRange24h, Step: time.Hour},
		Snapshots: []GridSnapshot{{
			Metric: GridMetricFunding, Exchange: ExchangeBinance, Symbol: "BTCUSDT",
			Bucket: col, SampledAt: now.Add(-6 * time.Minute), FundingRate: 0.0001,
		}},
	})
	cell := gridTestCell(t, got, "BTCUSDT", col)
	if cell.Definite || len(cell.Missing) != 1 {
		t.Fatalf("stale open cell %+v", cell)
	}
	requireGridVenue(t, cell, "binance", false, 0)
}

func TestBuildDerivativesGrid_OIChangeBlanksWithoutPrevious(t *testing.T) {
	now := time.Date(2026, 10, 10, 15, 40, 0, 0, time.UTC)
	col := time.Date(2026, 10, 10, 14, 0, 0, 0, time.UTC)
	prev := col.Add(-time.Hour)
	at := col.Add(10 * time.Minute)
	rows := []GridSnapshot{
		{Metric: FuturesMetricOpenInterest, Exchange: ExchangeBinance, Symbol: "BTCUSDT", Bucket: prev, SampledAt: prev.Add(time.Minute), Value: 100},
		{Metric: GridMetricOI, Exchange: ExchangeBybit, Symbol: "BTCUSDT", Bucket: prev, SampledAt: prev.Add(time.Minute), Value: 50},
		{Metric: GridMetricOI, Exchange: ExchangeOKX, Symbol: "BTCUSDT", Bucket: prev, SampledAt: prev.Add(time.Minute), Value: 50},
		{Metric: GridMetricOI, Exchange: ExchangeBinance, Symbol: "BTCUSDT", Bucket: col, SampledAt: at, Value: 110},
		{Metric: GridMetricOI, Exchange: ExchangeBybit, Symbol: "BTCUSDT", Bucket: col, SampledAt: at, Value: 50},
	}
	got := BuildDerivativesGrid(GridBuildInput{
		Exchange: "all", Now: now, Symbols: []string{"BTCUSDT"},
		Spec:      GridSpec{Metric: GridMetricOI, Range: GridRange24h, Step: time.Hour},
		Snapshots: rows,
	})
	cell := gridTestCell(t, got, "BTCUSDT", col)
	if cell.Definite || cell.Text != "" || len(cell.Missing) != 1 || cell.Missing[0] != "okx" {
		t.Fatalf("cell %+v", cell)
	}
	binance := requireGridVenue(t, cell, "binance", true, 10)
	if binance.Text != "+10.00%" {
		t.Fatalf("binance %+v", binance)
	}
	bybit := requireGridVenue(t, cell, "bybit", true, 0)
	if bybit.Text != "0.00%" {
		t.Fatalf("bybit %+v", bybit)
	}
	requireGridVenue(t, cell, "okx", false, 0)

	rows = append(rows, GridSnapshot{Metric: GridMetricOI, Exchange: ExchangeOKX, Symbol: "BTCUSDT", Bucket: col, SampledAt: at, Value: 40})
	got = BuildDerivativesGrid(GridBuildInput{
		Exchange: "all", Now: now, Symbols: []string{"BTCUSDT"},
		Spec:      GridSpec{Metric: GridMetricOI, Range: GridRange24h, Step: time.Hour},
		Snapshots: rows,
	})
	cell = gridTestCell(t, got, "BTCUSDT", col)
	// (200-200)/200 = 0
	if !cell.Definite || cell.Text != "0.00%" || cell.Value != 0 {
		t.Fatalf("combined %+v", cell)
	}
	okx := requireGridVenue(t, cell, "okx", true, -20)
	if okx.Text != "-20.00%" {
		t.Fatalf("okx %+v", okx)
	}
}

func TestBuildDerivativesGrid_CVDDoesNotZeroFill(t *testing.T) {
	now := time.Date(2026, 10, 10, 16, 10, 0, 0, time.UTC)
	col := time.Date(2026, 10, 10, 14, 0, 0, 0, time.UTC)
	full := func(ex Exchange, buy float64) GridTakerSum {
		return GridTakerSum{
			Exchange: ex, Symbol: "BTCUSDT", Bucket: col,
			Buy: buy, Sell: 10, Minutes: 60,
			First: col, Last: col.Add(59 * time.Minute),
		}
	}
	got := BuildDerivativesGrid(GridBuildInput{
		Exchange: "all", Now: now, Symbols: []string{"BTCUSDT"},
		Spec:  GridSpec{Metric: GridMetricCVD, Range: GridRange24h, Step: time.Hour},
		Taker: []GridTakerSum{full(ExchangeBinance, 40), full(ExchangeBybit, 20)},
	})
	cell := gridTestCell(t, got, "BTCUSDT", col)
	if cell.Definite || cell.Value != 0 {
		t.Fatalf("missing okx must stay blank: %+v", cell)
	}
	binance := requireGridVenue(t, cell, "binance", true, 30)
	if binance.Text != "+30" {
		t.Fatalf("binance %+v", binance)
	}
	bybit := requireGridVenue(t, cell, "bybit", true, 10)
	if bybit.Text != "+10" {
		t.Fatalf("bybit %+v", bybit)
	}
	requireGridVenue(t, cell, "okx", false, 0)
	got = BuildDerivativesGrid(GridBuildInput{
		Exchange: "all", Now: now, Symbols: []string{"BTCUSDT"},
		Spec: GridSpec{Metric: GridMetricCVD, Range: GridRange24h, Step: time.Hour},
		Taker: []GridTakerSum{
			full(ExchangeBinance, 40), full(ExchangeBybit, 20), full(ExchangeOKX, 10),
		},
	})
	cell = gridTestCell(t, got, "BTCUSDT", col)
	// (40-10)+(20-10)+(10-10) = 40
	if !cell.Definite || cell.Value != 40 || cell.Text != "+40" {
		t.Fatalf("sum %+v", cell)
	}
	okx := requireGridVenue(t, cell, "okx", true, 0)
	if okx.Text != "0" {
		t.Fatalf("okx %+v", okx)
	}
}

func TestBuildDerivativesGrid_ShortTapeIsNotADay(t *testing.T) {
	now := time.Date(2026, 10, 11, 1, 0, 0, 0, time.UTC)
	day := time.Date(2026, 10, 10, 0, 0, 0, 0, time.UTC)
	got := BuildDerivativesGrid(GridBuildInput{
		Exchange: "binance", Now: now, Symbols: []string{"BTCUSDT"},
		Spec: GridSpec{Metric: GridMetricCVD, Range: GridRange7d, Step: 24 * time.Hour},
		Taker: []GridTakerSum{{
			Exchange: ExchangeBinance, Symbol: "BTCUSDT", Bucket: day,
			Buy: 100, Sell: 40, Minutes: 60,
			First: day.Add(20 * time.Hour), Last: day.Add(21 * time.Hour),
		}},
	})
	cell := gridTestCell(t, got, "BTCUSDT", day)
	if cell.Definite {
		t.Fatalf("partial day counted: %+v", cell)
	}
	requireGridVenue(t, cell, "binance", false, 0)
}

func TestBuildDerivativesGrid_QuietWatchedHourIsZero(t *testing.T) {
	now := time.Date(2026, 10, 10, 16, 10, 0, 0, time.UTC)
	col := time.Date(2026, 10, 10, 14, 0, 0, 0, time.UTC)
	cover := []LiquidationCoverage{
		{Exchange: ExchangeBinance, FirstWatch: col.Add(-time.Hour), LastSeen: now},
		{Exchange: ExchangeBybit, FirstWatch: col.Add(-time.Hour), LastSeen: now},
		{Exchange: ExchangeOKX, FirstWatch: col.Add(-time.Hour), LastSeen: now},
	}
	got := BuildDerivativesGrid(GridBuildInput{
		Exchange: "all", Now: now, Symbols: []string{"ETHUSDT"},
		Spec:     GridSpec{Metric: GridMetricLiquidations, Range: GridRange24h, Step: time.Hour},
		Coverage: cover,
		Liquidations: []GridLiquidationSum{
			{Exchange: ExchangeBinance, Symbol: "ETHUSDT", Bucket: col, Notional: 1000},
		},
	})
	cell := gridTestCell(t, got, "ETHUSDT", col)
	if !cell.Definite || cell.Value != 1000 {
		t.Fatalf("quiet venues should add zero: %+v", cell)
	}
	binance := requireGridVenue(t, cell, "binance", true, 1000)
	if binance.Text != "1.0K" {
		t.Fatalf("binance %+v", binance)
	}
	for _, ex := range []string{"bybit", "okx"} {
		venue := requireGridVenue(t, cell, ex, true, 0)
		if venue.Text != "0" {
			t.Fatalf("%s %+v", ex, venue)
		}
	}

	cover[2].FirstWatch = col.Add(time.Hour)
	got = BuildDerivativesGrid(GridBuildInput{
		Exchange: "all", Now: now, Symbols: []string{"ETHUSDT"},
		Spec:     GridSpec{Metric: GridMetricLiquidations, Range: GridRange24h, Step: time.Hour},
		Coverage: cover,
		Liquidations: []GridLiquidationSum{
			{Exchange: ExchangeBinance, Symbol: "ETHUSDT", Bucket: col, Notional: 1000},
		},
	})
	cell = gridTestCell(t, got, "ETHUSDT", col)
	if cell.Definite || len(cell.Missing) != 1 || cell.Missing[0] != "okx" {
		t.Fatalf("unwatched %+v", cell)
	}
	requireGridVenue(t, cell, "binance", true, 1000)
	requireGridVenue(t, cell, "bybit", true, 0)
	requireGridVenue(t, cell, "okx", false, 0)
}

func TestBuildDerivativesGrid_HyperliquidIsFundingAndOpenInterestOnly(t *testing.T) {
	now := time.Date(2026, 10, 10, 15, 40, 0, 0, time.UTC)
	col := time.Date(2026, 10, 10, 14, 0, 0, 0, time.UTC)
	prev := col.Add(-time.Hour)
	at := col.Add(20 * time.Minute)
	spec := func(metric string) GridSpec {
		return GridSpec{Metric: metric, Range: GridRange24h, Step: time.Hour}
	}
	snaps := []GridSnapshot{
		{Metric: GridMetricFunding, Exchange: ExchangeHyperliquid, Symbol: "BTCUSDT", Bucket: col, SampledAt: at, FundingRate: 0.0000125},
		{Metric: GridMetricOI, Exchange: ExchangeHyperliquid, Symbol: "BTCUSDT", Bucket: prev, SampledAt: prev.Add(time.Minute), Value: 100},
		{Metric: GridMetricOI, Exchange: ExchangeHyperliquid, Symbol: "BTCUSDT", Bucket: col, SampledAt: at, Value: 110},
	}
	funding := BuildDerivativesGrid(GridBuildInput{
		Exchange: "hyperliquid", Now: now, Symbols: []string{"BTCUSDT"},
		Spec: spec(GridMetricFunding), Snapshots: snaps,
	})
	if !strings.Contains(funding.Note, "not zeros") {
		t.Fatalf("note %s", funding.Note)
	}
	cell := gridTestCell(t, funding, "BTCUSDT", col)
	if !cell.Definite || math.Abs(cell.Value-0.0000125) > 1e-12 || len(cell.Missing) != 0 {
		t.Fatalf("funding %+v", cell)
	}
	fundingVenue := requireGridVenue(t, cell, "hyperliquid", true, 0.0000125)
	if fundingVenue.Text != cell.Text || len(cell.Venues) != 1 {
		t.Fatalf("funding venues %+v", cell.Venues)
	}
	oi := BuildDerivativesGrid(GridBuildInput{
		Exchange: "hyperliquid", Now: now, Symbols: []string{"BTCUSDT"},
		Spec: spec(GridMetricOI), Snapshots: snaps,
	})
	cell = gridTestCell(t, oi, "BTCUSDT", col)
	if !cell.Definite || math.Abs(cell.Value-10) > 1e-9 {
		t.Fatalf("oi %+v", cell)
	}
	oiVenue := requireGridVenue(t, cell, "hyperliquid", true, 10)
	if oiVenue.Text != "+10.00%" || len(cell.Venues) != 1 {
		t.Fatalf("oi venue %+v", oiVenue)
	}
	for _, metric := range []string{GridMetricCVD, GridMetricLiquidations} {
		got := BuildDerivativesGrid(GridBuildInput{
			Exchange: "hyperliquid", Now: now, Symbols: []string{"BTCUSDT"},
			Spec: spec(metric),
		})
		cell = gridTestCell(t, got, "BTCUSDT", col)
		if cell.Definite || cell.Text != "" || cell.Value != 0 || len(cell.Missing) != 1 || cell.Missing[0] != "hyperliquid" {
			t.Fatalf("%s %+v", metric, cell)
		}
		requireGridVenue(t, cell, "hyperliquid", false, 0)
		if len(cell.Venues) != 1 {
			t.Fatalf("%s venues %+v", metric, cell.Venues)
		}
	}
	cvd := BuildDerivativesGrid(GridBuildInput{
		Exchange: "hyperliquid", Now: now, Symbols: []string{"BTCUSDT"},
		Spec: spec(GridMetricCVD),
		Taker: []GridTakerSum{{
			Exchange: ExchangeHyperliquid, Symbol: "BTCUSDT", Bucket: col,
			Buy: 80, Sell: 30, Minutes: 50,
			First: col, Last: col.Add(59 * time.Minute),
		}},
	})
	cell = gridTestCell(t, cvd, "BTCUSDT", col)
	if !cell.Definite || math.Abs(cell.Value-50) > 1e-9 || len(cell.Missing) != 0 {
		t.Fatalf("cvd %+v", cell)
	}
	requireGridVenue(t, cell, "hyperliquid", true, 50)

	combined := []GridSnapshot{
		{Metric: GridMetricFunding, Exchange: ExchangeBinance, Symbol: "BTCUSDT", Bucket: col, SampledAt: at, FundingRate: 0.0001},
		{Metric: GridMetricFunding, Exchange: ExchangeBybit, Symbol: "BTCUSDT", Bucket: col, SampledAt: at, FundingRate: 0.0001},
		{Metric: GridMetricFunding, Exchange: ExchangeOKX, Symbol: "BTCUSDT", Bucket: col, SampledAt: at, FundingRate: 0.0001},
		{Metric: GridMetricFunding, Exchange: ExchangeHyperliquid, Symbol: "BTCUSDT", Bucket: col, SampledAt: at, FundingRate: 0.01},
	}
	all := BuildDerivativesGrid(GridBuildInput{
		Exchange: "all", Now: now, Symbols: []string{"BTCUSDT"},
		Spec: spec(GridMetricFunding), Snapshots: combined,
	})
	cell = gridTestCell(t, all, "BTCUSDT", col)
	if !cell.Definite || math.Abs(cell.Value-0.0001) > 1e-12 || len(cell.Missing) != 0 {
		t.Fatalf("combined must ignore hyperliquid %+v", cell)
	}
	for _, ex := range []string{"binance", "bybit", "okx"} {
		requireGridVenue(t, cell, ex, true, 0.0001)
	}
	for _, venue := range cell.Venues {
		if venue.Exchange == "hyperliquid" {
			t.Fatalf("hyperliquid folded in %+v", cell.Venues)
		}
	}
}

func requireGridVenue(t *testing.T, cell GridCell, exchange string, definite bool, value float64) GridVenue {
	t.Helper()
	for _, venue := range cell.Venues {
		if venue.Exchange != exchange {
			continue
		}
		if definite {
			if !venue.Definite || !venue.HasValue || venue.Text == "" || math.Abs(venue.Value-value) > 1e-9 {
				t.Fatalf("%s venue %+v want %v", exchange, venue, value)
			}
			return venue
		}
		if venue.Definite || venue.HasValue || venue.Text != "" || venue.Value != 0 {
			t.Fatalf("%s should be missing %+v", exchange, venue)
		}
		return venue
	}
	t.Fatalf("no %s venue in %+v", exchange, cell.Venues)
	return GridVenue{}
}

func gridTestCell(t *testing.T, grid *DerivativesGrid, symbol string, at time.Time) GridCell {
	t.Helper()
	for _, row := range grid.Rows {
		if row.Symbol != symbol {
			continue
		}
		for _, cell := range row.Cells {
			if cell.Time.Equal(at) {
				return cell
			}
		}
	}
	t.Fatalf("no cell %s %s", symbol, at)
	return GridCell{}
}

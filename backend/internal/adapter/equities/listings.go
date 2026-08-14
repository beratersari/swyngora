package equities

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"unicode"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

const (
	defaultNasdaqScreener = "https://api.nasdaq.com/api/screener/stocks?tableonly=true&limit=10000&exchange=NASDAQ&download=true"
	defaultBistListURL    = "https://bigpara.hurriyet.com.tr/api/v1/hisse/list"
	defaultBistScreener   = "https://scanner.tradingview.com/turkey/scan"
	maxNasdaqMcap         = 20e12
	maxBistMcap           = 50e12
)

// Public TradingView Turkey scanner — one request for the BIST tape + market cap.
// No API key. Column order must match tvBistScanBody.
const tvBistScanBody = `{"filter":[{"left":"type","operation":"in_range","right":["stock","dr","fund"]}],"options":{"lang":"tr"},"markets":["turkey"],"symbols":{"query":{"types":[]},"tickers":[]},"columns":["name","close","change","change_abs","volume","market_cap_basic","sector","description"],"sort":{"sortBy":"name","sortOrder":"asc"},"range":[0,2000]}`

type tvScanResponse struct {
	Data []tvScanRow `json:"data"`
}

type tvScanRow struct {
	Symbol string        `json:"s"`
	Values []tvScanValue `json:"d"`
}

// tvScanValue accepts the scanner's mixed JSON (string / number / null).
type tvScanValue struct {
	str string
	num float64
	ok  bool
}

func (v *tvScanValue) UnmarshalJSON(b []byte) error {
	s := strings.TrimSpace(string(b))
	if s == "" || s == "null" {
		return nil
	}
	if len(s) >= 2 && s[0] == '"' {
		var out string
		if err := json.Unmarshal(b, &out); err != nil {
			return err
		}
		v.str = out
		v.ok = true
		if n, err := strconv.ParseFloat(strings.ReplaceAll(out, ",", ""), 64); err == nil {
			v.num = n
		}
		return nil
	}
	n, err := strconv.ParseFloat(s, 64)
	if err != nil {
		v.str = strings.Trim(s, `"`)
		v.ok = v.str != ""
		return nil
	}
	v.num = n
	v.ok = true
	return nil
}

type nasdaqScreenerResponse struct {
	Data *struct {
		Rows []nasdaqScreenerRow `json:"rows"`
	} `json:"data"`
}

type nasdaqScreenerRow struct {
	Symbol    string `json:"symbol"`
	Name      string `json:"name"`
	LastSale  string `json:"lastsale"`
	NetChange string `json:"netchange"`
	PctChange string `json:"pctchange"`
	Volume    string `json:"volume"`
	MarketCap string `json:"marketCap"`
	Sector    string `json:"sector"`
	Industry  string `json:"industry"`
}

type bigparaListResponse struct {
	Data []struct {
		Kod string `json:"kod"`
		Tip string `json:"tip"`
	} `json:"data"`
}

func (c *Client) fetchNasdaqScreener(ctx context.Context) ([]domain.SpotMarket, error) {
	base := c.nasdaqURL
	if base == "" {
		base = defaultNasdaqScreener
	}
	body, err := c.get(ctx, base)
	if err != nil {
		return nil, err
	}
	var wrap nasdaqScreenerResponse
	if err := json.Unmarshal(body, &wrap); err != nil {
		return nil, fmt.Errorf("%w: nasdaq screener json: %v", domain.ErrUpstream, err)
	}
	if wrap.Data == nil {
		return nil, fmt.Errorf("%w: empty nasdaq screener", domain.ErrUpstream)
	}
	out := make([]domain.SpotMarket, 0, len(wrap.Data.Rows))
	for _, row := range wrap.Data.Rows {
		if m := spotFromNasdaqRow(row); m != nil {
			out = append(out, *m)
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("%w: nasdaq screener had no rows", domain.ErrUpstream)
	}
	return out, nil
}

func spotFromNasdaqRow(row nasdaqScreenerRow) *domain.SpotMarket {
	sym := sanitizeTicker(row.Symbol)
	if sym == "" {
		return nil
	}
	last := parseMoney(row.LastSale)
	if last <= 0 {
		return nil
	}
	chg := parseSigned(row.NetChange)
	pct := parsePct(row.PctChange)
	vol := parseMoney(row.Volume)
	mcap := parseMoney(row.MarketCap)
	var tags []string
	if s := strings.TrimSpace(row.Sector); s != "" && !strings.EqualFold(s, "n/a") {
		tags = []string{s}
	}
	m := &domain.SpotMarket{
		Symbol:             sym,
		BaseAsset:          sym,
		QuoteAsset:         "USD",
		Status:             "TRADING",
		LastPrice:          fmtFloat(last),
		PriceChange:        fmtFloat(chg),
		PriceChangePercent: fmtFloat(pct),
		Volume:             fmtFloat(vol),
		QuoteVolume:        fmtFloat(last * vol),
		Tags:               tags,
	}
	applyEquityMcap(m, mcap, last, maxNasdaqMcap)
	return m
}

func (c *Client) fetchBistScreener(ctx context.Context) ([]domain.SpotMarket, error) {
	base := c.bistScreenerURL
	if base == "" {
		base = defaultBistScreener
	}
	body, err := c.postJSON(ctx, base, []byte(tvBistScanBody))
	if err != nil {
		return nil, err
	}
	var wrap tvScanResponse
	if err := json.Unmarshal(body, &wrap); err != nil {
		return nil, fmt.Errorf("%w: bist screener json: %v", domain.ErrUpstream, err)
	}
	out := make([]domain.SpotMarket, 0, len(wrap.Data))
	for _, row := range wrap.Data {
		if m := spotFromBistScanRow(row); m != nil {
			out = append(out, *m)
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("%w: bist screener had no rows", domain.ErrUpstream)
	}
	return out, nil
}

func spotFromBistScanRow(row tvScanRow) *domain.SpotMarket {
	// columns: name, close, change, change_abs, volume, market_cap_basic, sector, description
	if len(row.Values) < 2 {
		return nil
	}
	sym := sanitizeTicker(row.Values[0].str)
	if sym == "" {
		sym = sanitizeTicker(strings.TrimPrefix(strings.ToUpper(row.Symbol), "BIST:"))
	}
	if sym == "" {
		return nil
	}
	last := row.Values[1].num
	if last <= 0 {
		return nil
	}
	var pct, chg, vol, mcap float64
	if len(row.Values) > 2 {
		pct = row.Values[2].num
	}
	if len(row.Values) > 3 {
		chg = row.Values[3].num
	}
	if len(row.Values) > 4 {
		vol = row.Values[4].num
	}
	if len(row.Values) > 5 {
		mcap = row.Values[5].num
	}
	var tags []string
	if len(row.Values) > 6 {
		if s := strings.TrimSpace(row.Values[6].str); s != "" && !strings.EqualFold(s, "n/a") {
			tags = []string{s}
		}
	}
	m := &domain.SpotMarket{
		Symbol:             sym,
		BaseAsset:          sym,
		QuoteAsset:         "TRY",
		Status:             "TRADING",
		LastPrice:          fmtFloat(last),
		PriceChange:        fmtFloat(chg),
		PriceChangePercent: fmtFloat(pct),
		Volume:             fmtFloat(vol),
		QuoteVolume:        fmtFloat(last * vol),
		Tags:               tags,
	}
	applyEquityMcap(m, mcap, last, maxBistMcap)
	return m
}

func applyEquityMcap(m *domain.SpotMarket, mcap, last, max float64) {
	if m == nil || mcap <= 0 || mcap >= max {
		return
	}
	cp := mcap
	m.MarketCapCirculating = &cp
	m.MarketCapTotal = &cp
	if last > 0 {
		circ := mcap / last
		m.CirculatingSupply = &circ
	}
}

func (c *Client) fetchBistUniverse(ctx context.Context) []string {
	base := c.bistListURL
	if base == "" {
		base = defaultBistListURL
	}
	u, err := url.Parse(base)
	if err != nil {
		return append([]string(nil), c.universe...)
	}
	body, err := c.get(ctx, u.String())
	if err != nil {
		return append([]string(nil), c.universe...)
	}
	var wrap bigparaListResponse
	if err := json.Unmarshal(body, &wrap); err != nil {
		return append([]string(nil), c.universe...)
	}
	var out []string
	for _, row := range wrap.Data {
		kod := sanitizeTicker(row.Kod)
		if kod == "" {
			continue
		}
		out = append(out, kod)
	}
	out = uniqueUpper(out)
	if len(out) < 20 {
		return uniqueUpper(append(append([]string{}, c.universe...), out...))
	}
	return out
}

func sanitizeTicker(s string) string {
	s = strings.ToUpper(strings.TrimSpace(s))
	s = strings.TrimSuffix(s, ".IS")
	if s == "" {
		return ""
	}
	for _, r := range s {
		if !(unicode.IsLetter(r) || unicode.IsDigit(r) || r == '.') {
			return ""
		}
	}
	return s
}

func parseMoney(s string) float64 {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "$")
	s = strings.ReplaceAll(s, ",", "")
	s = strings.TrimSpace(s)
	if s == "" || strings.EqualFold(s, "n/a") || s == "--" {
		return 0
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0
	}
	return v
}

func parseSigned(s string) float64 {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, ",", "")
	s = strings.TrimPrefix(s, "+")
	if s == "" || strings.EqualFold(s, "unch") {
		return 0
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0
	}
	return v
}

func parsePct(s string) float64 {
	s = strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(s), "%"))
	return parseSigned(s)
}

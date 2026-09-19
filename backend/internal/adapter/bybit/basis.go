package bybit

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

// GetBasisQuote loads Bybit linear last/mark vs index (and spot last when listed).
func (c *Client) GetBasisQuote(ctx context.Context, symbol string) (*domain.BasisQuote, error) {
	symbol, err := domain.ValidateOpenInterestSymbol(symbol)
	if err != nil {
		return nil, err
	}
	if c == nil {
		return nil, fmt.Errorf("%w: bybit client", domain.ErrUpstream)
	}
	if cached, ok := c.basisCache.Get(symbol); ok && cached != nil {
		return cached, nil
	}
	v, err, _ := c.basisSF.Do(symbol, func() (any, error) {
		if cached, ok := c.basisCache.Get(symbol); ok && cached != nil {
			return cached, nil
		}
		got, err := c.fetchBasis(ctx, symbol)
		if err != nil {
			return nil, err
		}
		c.basisCache.Set(symbol, got)
		return got, nil
	})
	if err != nil {
		return nil, err
	}
	got, _ := v.(*domain.BasisQuote)
	return got, nil
}

func (c *Client) fetchBasis(ctx context.Context, symbol string) (*domain.BasisQuote, error) {
	var (
		last, mark, index, spot float64
		at                      time.Time
		marks, indexes          []domain.PriceSample
		errT, errS, errM, errI  error
		wg                      sync.WaitGroup
	)
	wg.Add(4)
	go func() {
		defer wg.Done()
		last, mark, index, at, errT = c.fetchLinearPrices(ctx, symbol)
	}()
	go func() {
		defer wg.Done()
		spot, errS = c.fetchSpotLast(ctx, symbol)
	}()
	go func() {
		defer wg.Done()
		marks, errM = c.fetchPriceKlines(ctx, "/v5/market/mark-price-kline", symbol)
	}()
	go func() {
		defer wg.Done()
		indexes, errI = c.fetchPriceKlines(ctx, "/v5/market/index-price-kline", symbol)
	}()
	wg.Wait()
	if errT != nil {
		return nil, errT
	}
	if mark <= 0 || index <= 0 {
		return nil, fmt.Errorf("%w: linear mark/index", domain.ErrUpstream)
	}
	_ = errS
	if errM != nil {
		marks = nil
	}
	if errI != nil {
		indexes = nil
	}
	out := &domain.BasisQuote{
		Exchange:    domain.ExchangeBybit,
		Symbol:      symbol,
		FuturesLast: last,
		FuturesMark: mark,
		SpotIndex:   index,
		SpotLast:    spot,
		Time:        at,
		History:     domain.BuildBasisHistory(marks, indexes),
		Contracts:   c.fetchBasisContracts(ctx, symbol, last, mark, index),
	}
	if out.Time.IsZero() {
		out.Time = time.Now().UTC()
	}
	return out, nil
}

func (c *Client) fetchLinearPrices(ctx context.Context, symbol string) (last, mark, index float64, at time.Time, err error) {
	q := url.Values{}
	q.Set("category", "linear")
	q.Set("symbol", symbol)
	body, err := c.get(ctx, "/v5/market/tickers", q)
	if err != nil {
		return 0, 0, 0, time.Time{}, err
	}
	var raw struct {
		RetCode int    `json:"retCode"`
		RetMsg  string `json:"retMsg"`
		Time    int64  `json:"time"`
		Result  struct {
			List []struct {
				LastPrice  string `json:"lastPrice"`
				MarkPrice  string `json:"markPrice"`
				IndexPrice string `json:"indexPrice"`
			} `json:"list"`
		} `json:"result"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return 0, 0, 0, time.Time{}, fmt.Errorf("%w: ticker decode: %v", domain.ErrUpstream, err)
	}
	if raw.RetCode != 0 {
		return 0, 0, 0, time.Time{}, mapBybitError(raw.RetCode, raw.RetMsg)
	}
	if len(raw.Result.List) == 0 {
		return 0, 0, 0, time.Time{}, fmt.Errorf("%w: empty ticker", domain.ErrNotFound)
	}
	row := raw.Result.List[0]
	last, _ = strconv.ParseFloat(strings.TrimSpace(row.LastPrice), 64)
	mark, _ = strconv.ParseFloat(strings.TrimSpace(row.MarkPrice), 64)
	index, _ = strconv.ParseFloat(strings.TrimSpace(row.IndexPrice), 64)
	if raw.Time > 0 {
		at = time.UnixMilli(raw.Time).UTC()
	}
	return last, mark, index, at, nil
}

func (c *Client) fetchSpotLast(ctx context.Context, symbol string) (float64, error) {
	q := url.Values{}
	q.Set("category", "spot")
	q.Set("symbol", symbol)
	body, err := c.get(ctx, "/v5/market/tickers", q)
	if err != nil {
		return 0, err
	}
	var raw struct {
		RetCode int `json:"retCode"`
		Result  struct {
			List []struct {
				LastPrice string `json:"lastPrice"`
			} `json:"list"`
		} `json:"result"`
	}
	if json.Unmarshal(body, &raw) != nil || raw.RetCode != 0 || len(raw.Result.List) == 0 {
		return 0, fmt.Errorf("%w: spot last", domain.ErrNotFound)
	}
	return strconv.ParseFloat(strings.TrimSpace(raw.Result.List[0].LastPrice), 64)
}

func (c *Client) fetchPriceKlines(ctx context.Context, path, symbol string) ([]domain.PriceSample, error) {
	q := url.Values{}
	q.Set("category", "linear")
	q.Set("symbol", symbol)
	q.Set("interval", "5")
	q.Set("limit", "60")
	body, err := c.get(ctx, path, q)
	if err != nil {
		return nil, err
	}
	var raw struct {
		RetCode int    `json:"retCode"`
		RetMsg  string `json:"retMsg"`
		Result  struct {
			List [][]string `json:"list"`
		} `json:"result"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("%w: kline decode: %v", domain.ErrUpstream, err)
	}
	if raw.RetCode != 0 {
		return nil, mapBybitError(raw.RetCode, raw.RetMsg)
	}
	out := make([]domain.PriceSample, 0, len(raw.Result.List))
	for _, row := range raw.Result.List {
		if len(row) < 5 {
			continue
		}
		ms, err1 := strconv.ParseInt(strings.TrimSpace(row[0]), 10, 64)
		px, err2 := strconv.ParseFloat(strings.TrimSpace(row[4]), 64)
		if err1 != nil || err2 != nil || ms <= 0 || px <= 0 {
			continue
		}
		out = append(out, domain.PriceSample{Time: time.UnixMilli(ms).UTC(), Price: px})
	}
	return out, nil
}

type linContractMeta struct {
	Symbol   string
	Type     string
	Base     string
	Quote    string
	Status   string
	Delivery time.Time
}

type linPx struct {
	last, mark, index float64
}

func (c *Client) fetchBasisContracts(ctx context.Context, perp string, perpLast, perpMark, perpIndex float64) []domain.BasisContractQuote {
	if c == nil {
		return nil
	}
	base, quote := splitUSDTPair(perp)
	if base == "" {
		return []domain.BasisContractQuote{{
			Symbol: perp, Kind: domain.BasisContractPerp, Last: perpLast, Mark: perpMark, Index: perpIndex,
		}}
	}
	meta, err := c.listLinearContracts(ctx)
	if err != nil || len(meta) == 0 {
		return []domain.BasisContractQuote{{
			Symbol: perp, Kind: domain.BasisContractPerp, Last: perpLast, Mark: perpMark, Index: perpIndex,
		}}
	}
	px, _ := c.fetchAllLinearPrices(ctx)
	out := make([]domain.BasisContractQuote, 0, 6)
	for _, m := range meta {
		if !strings.EqualFold(m.Base, base) || !strings.EqualFold(m.Quote, quote) {
			continue
		}
		if m.Status != "" && !strings.EqualFold(m.Status, "Trading") {
			continue
		}
		kind := domain.BasisContractDated
		if strings.EqualFold(m.Type, "LinearPerpetual") || m.Delivery.IsZero() {
			kind = domain.BasisContractPerp
		}
		p := px[m.Symbol]
		if kind == domain.BasisContractPerp {
			if p.last <= 0 {
				p.last = perpLast
			}
			if p.mark <= 0 {
				p.mark = perpMark
			}
			if p.index <= 0 {
				p.index = perpIndex
			}
		}
		if p.last <= 0 && p.mark <= 0 {
			continue
		}
		out = append(out, domain.BasisContractQuote{
			Symbol: m.Symbol, Kind: kind, Expiry: m.Delivery,
			Last: p.last, Mark: p.mark, Index: p.index,
		})
	}
	if len(out) == 0 {
		out = append(out, domain.BasisContractQuote{
			Symbol: perp, Kind: domain.BasisContractPerp, Last: perpLast, Mark: perpMark, Index: perpIndex,
		})
	}
	return out
}

func splitUSDTPair(symbol string) (base, quote string) {
	s := strings.ToUpper(strings.TrimSpace(symbol))
	for _, q := range []string{"USDT", "USDC"} {
		if strings.HasSuffix(s, q) && len(s) > len(q) {
			return s[:len(s)-len(q)], q
		}
	}
	return "", ""
}

func (c *Client) listLinearContracts(ctx context.Context) ([]linContractMeta, error) {
	if c.linMeta != nil {
		if cached, ok := c.linMeta.Get("all"); ok {
			return cached, nil
		}
	}
	v, err, _ := c.linMetaSF.Do("all", func() (any, error) {
		if c.linMeta != nil {
			if cached, ok := c.linMeta.Get("all"); ok {
				return cached, nil
			}
		}
		q := url.Values{}
		q.Set("category", "linear")
		q.Set("limit", "1000")
		body, err := c.get(ctx, "/v5/market/instruments-info", q)
		if err != nil {
			return nil, err
		}
		var raw struct {
			RetCode int    `json:"retCode"`
			RetMsg  string `json:"retMsg"`
			Result  struct {
				List []struct {
					Symbol       string `json:"symbol"`
					ContractType string `json:"contractType"`
					BaseCoin     string `json:"baseCoin"`
					QuoteCoin    string `json:"quoteCoin"`
					Status       string `json:"status"`
					DeliveryTime string `json:"deliveryTime"`
				} `json:"list"`
			} `json:"result"`
		}
		if err := json.Unmarshal(body, &raw); err != nil {
			return nil, fmt.Errorf("%w: instruments decode: %v", domain.ErrUpstream, err)
		}
		if raw.RetCode != 0 {
			return nil, mapBybitError(raw.RetCode, raw.RetMsg)
		}
		out := make([]linContractMeta, 0, len(raw.Result.List))
		for _, s := range raw.Result.List {
			row := linContractMeta{
				Symbol: s.Symbol, Type: s.ContractType, Base: s.BaseCoin,
				Quote: s.QuoteCoin, Status: s.Status,
			}
			if ms, err := strconv.ParseInt(strings.TrimSpace(s.DeliveryTime), 10, 64); err == nil && ms > 0 && ms < 4_000_000_000_000 {
				row.Delivery = time.UnixMilli(ms).UTC()
			}
			out = append(out, row)
		}
		if c.linMeta != nil {
			c.linMeta.Set("all", out)
		}
		return out, nil
	})
	if err != nil {
		return nil, err
	}
	got, _ := v.([]linContractMeta)
	return got, nil
}

func (c *Client) fetchAllLinearPrices(ctx context.Context) (map[string]linPx, error) {
	q := url.Values{}
	q.Set("category", "linear")
	body, err := c.get(ctx, "/v5/market/tickers", q)
	if err != nil {
		return nil, err
	}
	var raw struct {
		RetCode int `json:"retCode"`
		Result  struct {
			List []struct {
				Symbol     string `json:"symbol"`
				LastPrice  string `json:"lastPrice"`
				MarkPrice  string `json:"markPrice"`
				IndexPrice string `json:"indexPrice"`
			} `json:"list"`
		} `json:"result"`
	}
	if err := json.Unmarshal(body, &raw); err != nil || raw.RetCode != 0 {
		return nil, fmt.Errorf("%w: linear tickers", domain.ErrUpstream)
	}
	out := make(map[string]linPx, len(raw.Result.List))
	for _, r := range raw.Result.List {
		if r.Symbol == "" {
			continue
		}
		last, _ := strconv.ParseFloat(strings.TrimSpace(r.LastPrice), 64)
		mark, _ := strconv.ParseFloat(strings.TrimSpace(r.MarkPrice), 64)
		idx, _ := strconv.ParseFloat(strings.TrimSpace(r.IndexPrice), 64)
		out[r.Symbol] = linPx{last: last, mark: mark, index: idx}
	}
	return out, nil
}

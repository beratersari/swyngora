package binance

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

// SetTakerWatch subscribes a symbol on the USD-M trade hub.
func (c *Client) SetTakerWatch(fn func(string)) {
	if c != nil {
		c.takerWatch = fn
	}
}

// TakerBook is the rolling 1-minute measured tape (restore and the trade hub).
func (c *Client) TakerBook() *domain.TakerBook {
	if c == nil {
		return nil
	}
	return c.taker
}

// PrintTape is the recent individual fills behind that minute book.
func (c *Client) PrintTape() *domain.PrintTape {
	if c == nil {
		return nil
	}
	return c.tape
}

// GetTakerFlow returns measured Binance USD-M taker buy/sell (websocket plus one REST seed).
func (c *Client) GetTakerFlow(ctx context.Context, symbol string) (*domain.TakerVenueFlow, error) {
	symbol, err := domain.ValidateOpenInterestSymbol(symbol)
	if err != nil {
		return nil, err
	}
	if c == nil || c.taker == nil {
		return nil, fmt.Errorf("%w: binance taker book", domain.ErrUpstream)
	}
	if err := c.ensureMeasuredTape(ctx, symbol); err != nil && len(c.taker.Buckets(domain.ExchangeBinance, symbol)) == 0 {
		return nil, err
	}
	out := c.taker.Snapshot(domain.ExchangeBinance, symbol)
	return &out, nil
}

// GetTakerBuckets returns measured 1-minute buy/sell bars.
func (c *Client) GetTakerBuckets(ctx context.Context, symbol string) ([]domain.TakerBucket, error) {
	symbol, err := domain.ValidateOpenInterestSymbol(symbol)
	if err != nil {
		return nil, err
	}
	if c == nil || c.taker == nil {
		return nil, fmt.Errorf("%w: binance taker book", domain.ErrUpstream)
	}
	if err := c.ensureMeasuredTape(ctx, symbol); err != nil && len(c.taker.Buckets(domain.ExchangeBinance, symbol)) == 0 {
		return nil, err
	}
	return c.taker.Buckets(domain.ExchangeBinance, symbol), nil
}

// ensureMeasuredTape subscribes the symbol and, when the book is still empty,
// folds the newest aggTrades in once. A restored or live book is left alone
// so those minutes are not added twice.
func (c *Client) ensureMeasuredTape(ctx context.Context, symbol string) error {
	if c == nil {
		return fmt.Errorf("%w: binance client", domain.ErrUpstream)
	}
	if c.takerWatch != nil {
		c.takerWatch(symbol)
	}
	if c.bookHas(symbol) {
		return nil
	}
	c.seedMu.Lock()
	if c.seeded == nil {
		c.seeded = map[string]struct{}{}
	}
	if _, ok := c.seeded[symbol]; ok {
		c.seedMu.Unlock()
		return nil
	}
	c.seeded[symbol] = struct{}{}
	c.seedMu.Unlock()

	if _, _, err := c.pullAggPage(ctx, symbol); err != nil {
		c.seedMu.Lock()
		delete(c.seeded, symbol)
		c.seedMu.Unlock()
		return err
	}
	return nil
}

// PullAggTrades walks USD-M aggTrades forward from the last id we stored.
// It does nothing when a print arrived in the last few seconds, so a live
// websocket does not also burn REST weight. pages caps how far one call chases.
func (c *Client) PullAggTrades(ctx context.Context, symbol string, pages int) error {
	symbol = domain.NormalizeLiquidationSymbol(symbol)
	if c == nil || symbol == "" || pages <= 0 {
		return nil
	}
	if c.aggFresh(symbol) {
		return nil
	}
	for i := 0; i < pages; i++ {
		n, full, err := c.pullAggPage(ctx, symbol)
		if err != nil || n == 0 || !full {
			return err
		}
	}
	return nil
}

func (c *Client) aggFresh(symbol string) bool {
	if c == nil || c.tape == nil {
		return false
	}
	ps := c.tape.Prints(domain.ExchangeBinance, symbol)
	if len(ps) == 0 {
		return false
	}
	return time.Since(ps[len(ps)-1].Time) < 3*time.Second
}

// pullAggPage fetches one aggTrades page. full is true when the page hit the 1000-row cap.
func (c *Client) pullAggPage(ctx context.Context, symbol string) (int, bool, error) {
	if c == nil {
		return 0, false, fmt.Errorf("%w: binance client", domain.ErrUpstream)
	}
	q := url.Values{}
	q.Set("symbol", symbol)
	q.Set("limit", "1000")
	c.seedMu.Lock()
	if c.lastAgg == nil {
		c.lastAgg = map[string]int64{}
	}
	if id := c.lastAgg[symbol]; id > 0 {
		q.Set("fromId", strconv.FormatInt(id+1, 10))
	}
	c.seedMu.Unlock()
	body, err := c.getFutures(ctx, "/fapi/v1/aggTrades", q)
	if err != nil {
		return 0, false, err
	}
	var rows []struct {
		ID    int64  `json:"a"`
		Price string `json:"p"`
		Qty   string `json:"q"`
		Time  int64  `json:"T"`
		Maker bool   `json:"m"`
	}
	if err := json.Unmarshal(body, &rows); err != nil {
		return 0, false, fmt.Errorf("%w: aggTrades decode: %v", domain.ErrUpstream, err)
	}
	var maxID int64
	n := 0
	for _, row := range rows {
		px, e1 := strconv.ParseFloat(row.Price, 64)
		qty, e2 := strconv.ParseFloat(row.Qty, 64)
		if e1 != nil || e2 != nil || px <= 0 || qty <= 0 || row.Time <= 0 {
			continue
		}
		side := domain.TakerSideBuy
		if row.Maker {
			side = domain.TakerSideSell
		}
		domain.RecordMeasuredPrint(c.taker, c.tape, domain.TakerPrint{
			Exchange: domain.ExchangeBinance, Symbol: symbol, Side: side,
			Price: px, Quantity: qty, Notional: px * qty,
			Time: time.UnixMilli(row.Time).UTC(),
		})
		if row.ID > maxID {
			maxID = row.ID
		}
		n++
	}
	if maxID > 0 {
		c.seedMu.Lock()
		if maxID > c.lastAgg[symbol] {
			c.lastAgg[symbol] = maxID
		}
		c.seedMu.Unlock()
	}
	return n, n >= 1000, nil
}

func (c *Client) bookHas(symbol string) bool {
	if c == nil || c.taker == nil {
		return false
	}
	return len(c.taker.Buckets(domain.ExchangeBinance, symbol)) > 0
}

package okx

import (
	"context"
	"net/url"
	"sync"
	"time"
)

// SwapContractValue is the base-asset size of one USDT-swap contract.
func (c *Client) SwapContractValue(instID string) float64 {
	if c == nil {
		return 0
	}
	return c.contractValue(context.Background(), instID)
}

func (c *Client) contractValue(ctx context.Context, instID string) float64 {
	if c == nil || instID == "" {
		return 0
	}
	c.instMu.RLock()
	v, ok := c.ctVal[instID]
	c.instMu.RUnlock()
	if ok {
		return v
	}
	c.ensureInstruments(ctx)
	c.instMu.RLock()
	v = c.ctVal[instID]
	c.instMu.RUnlock()
	if v > 0 {
		return v
	}
	return c.fetchOneCtVal(ctx, instID)
}

func (c *Client) ensureInstruments(ctx context.Context) {
	c.instOnce.Do(func() {
		_ = c.refreshInstruments(ctx)
	})
}

func (c *Client) refreshInstruments(ctx context.Context) error {
	if c == nil {
		return nil
	}
	q := url.Values{}
	q.Set("instType", "SWAP")
	body, err := c.get(ctx, "/api/v5/public/instruments", q)
	if err != nil {
		return err
	}
	var rows []struct {
		InstID string `json:"instId"`
		CtVal  string `json:"ctVal"`
		CtType string `json:"ctType"`
		State  string `json:"state"`
	}
	if err := decodeOKX(body, &rows); err != nil {
		return err
	}
	next := make(map[string]float64, len(rows))
	for _, r := range rows {
		if r.InstID == "" {
			continue
		}
		v, ok := parseFloat(r.CtVal)
		if !ok || v <= 0 {
			continue
		}
		next[r.InstID] = v
	}
	if len(next) == 0 {
		return nil
	}
	c.instMu.Lock()
	c.ctVal = next
	c.instMu.Unlock()
	return nil
}

func (c *Client) fetchOneCtVal(ctx context.Context, instID string) float64 {
	q := url.Values{}
	q.Set("instType", "SWAP")
	q.Set("instId", instID)
	body, err := c.get(ctx, "/api/v5/public/instruments", q)
	if err != nil {
		return 0
	}
	var rows []struct {
		InstID string `json:"instId"`
		CtVal  string `json:"ctVal"`
	}
	if err := decodeOKX(body, &rows); err != nil || len(rows) == 0 {
		return 0
	}
	v, ok := parseFloat(rows[0].CtVal)
	if !ok || v <= 0 {
		return 0
	}
	c.instMu.Lock()
	if c.ctVal == nil {
		c.ctVal = map[string]float64{}
	}
	c.ctVal[instID] = v
	c.instMu.Unlock()
	return v
}

// StartInstrumentRefresh reloads SWAP contract sizes in the background.
func (c *Client) StartInstrumentRefresh(ctx context.Context) {
	if c == nil {
		return
	}
	_ = c.refreshInstruments(ctx)
	t := time.NewTicker(30 * time.Minute)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			_ = c.refreshInstruments(ctx)
		}
	}
}

// SetContractValue is a test helper.
func (c *Client) SetContractValue(instID string, v float64) {
	if c == nil {
		return
	}
	c.instMu.Lock()
	if c.ctVal == nil {
		c.ctVal = map[string]float64{}
	}
	c.ctVal[instID] = v
	c.instOnce = sync.Once{}
	c.instOnce.Do(func() {})
	c.instMu.Unlock()
}

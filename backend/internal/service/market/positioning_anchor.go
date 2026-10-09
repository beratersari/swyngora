package market

import (
	"context"
	"fmt"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

// GetPositioningAnchor restarts stored open interest, funding, and liquidation
// notional at one candle. from and to bound the stored window. An empty bucket
// follows that window the same way the coin chart does.
func (s *Service) GetPositioningAnchor(ctx context.Context, exchange, symbol, anchorRaw, bucketRaw string, from, to *time.Time) (*domain.PositioningAnchor, error) {
	if s == nil || s.futHist == nil {
		return nil, fmt.Errorf("%w: futures history not configured", domain.ErrUpstream)
	}
	symbol, err := domain.ValidateOpenInterestSymbol(symbol)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	anchor, err := domain.ParseCVDAnchor(anchorRaw, now)
	if err != nil {
		return nil, err
	}
	if err := domain.ValidatePositioningAnchor(anchor); err != nil {
		return nil, err
	}
	ex, err := domain.ParseOpenInterestExchange(exchange)
	if err != nil {
		return nil, err
	}
	var fromAt, toAt time.Time
	if from != nil {
		fromAt = *from
	}
	if to != nil {
		toAt = *to
	}
	fromAt, toAt = domain.ClampFuturesPaneWindow(fromAt, toAt, now)
	bucket, err := domain.PositioningAnchorBucket(bucketRaw, fromAt, toAt)
	if err != nil {
		return nil, err
	}
	load := func(metric string) (any, error) {
		return s.futHist.History(ctx, domain.FuturesHistoryQuery{
			Metric: metric, Exchange: ex, Symbol: symbol,
			From: fromAt, To: toAt, Bucket: bucket,
			Limit: domain.MaxFuturesHistoryLimit,
		})
	}
	oiRaw, err := load(domain.FuturesMetricOpenInterest)
	if err != nil {
		return nil, err
	}
	fundingRaw, err := load(domain.FuturesMetricFunding)
	if err != nil {
		return nil, err
	}
	liqRaw, err := load("liquidations")
	if err != nil {
		return nil, err
	}
	oi, _ := oiRaw.([]domain.FuturesSnapshot)
	funding, _ := fundingRaw.([]domain.FuturesSnapshot)
	liq, _ := liqRaw.([]domain.LiquidationEvent)
	s.noteFutures(symbol)
	return domain.BuildPositioningAnchor(domain.PositioningAnchorInput{
		Symbol: symbol, Exchange: ex, Anchor: anchor, Bucket: bucket,
		OpenInterest: oi, Funding: funding, Liquidations: liq,
	}), nil
}

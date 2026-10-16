package portfolio

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

// PortfolioCorrRequest is an optional extra line. Notional is absolute.
// Side is buy, sell, long, or short. Empty means read the open book only.
type PortfolioCorrRequest struct {
	Exchange domain.Exchange
	Symbol   string
	Side     string
	Notional float64
}

// BookCorrelation reads how the open coins moved together over the last day,
// and how one extra notional would change that book's day swing.
// A coin without enough shared minutes is named and left out.
func (s *Service) BookCorrelation(ctx context.Context, clientID, portfolioID, ownerID string, add PortfolioCorrRequest) (*domain.PortfolioCorrelation, error) {
	view, err := s.View(ctx, clientID, portfolioID, ownerID)
	if err != nil {
		return nil, err
	}
	return s.correlationFromView(ctx, view, add)
}

func (s *Service) correlationFromView(ctx context.Context, view *domain.PortfolioView, add PortfolioCorrRequest) (*domain.PortfolioCorrelation, error) {
	if view == nil {
		return nil, domain.ErrNotFound
	}
	signed, asset, err := signedAdd(add)
	if err != nil {
		return nil, err
	}
	lines := exposureLines(view)
	if signed != 0 {
		if asset == "" {
			return nil, fmt.Errorf("%w: symbol is required", domain.ErrInvalidArgument)
		}
		lines = mergeLine(lines, exposureLine{
			Asset: asset, Exchange: add.Exchange, Symbol: add.Symbol, Notional: 0,
		})
	}
	priced, missing, err := s.priceLines(ctx, lines)
	if err != nil {
		return nil, err
	}
	beforeRows, beforeMissing := joinReturns(priced)
	beforeMissing = append(beforeMissing, missing...)
	before := domain.BuildPortfolioCorrelation(beforeRows)
	before.Missing = mergeNames(before.Missing, beforeMissing)

	if signed == 0 {
		return &before, nil
	}
	afterPriced := addNotional(priced, asset, signed)
	afterRows, afterMissing := joinReturns(afterPriced)
	afterMissing = append(afterMissing, missing...)
	after := domain.BuildPortfolioCorrelation(afterRows)
	addRead := &domain.PortfolioCorrAdd{
		Asset: asset, Notional: signed, Side: domainSide(signed),
		Samples: after.Samples,
	}
	joined := !containsName(afterMissing, asset) && after.SwingPct != nil
	if joined {
		addRead.Complete = true
		addRead.Together = after.Together
		addRead.SwingPct = after.SwingPct
		addRead.ApartPct = after.ApartPct
	}
	before.Add = addRead
	return &before, nil
}

func domainSide(notional float64) string {
	if notional < 0 {
		return "short"
	}
	return "long"
}

func signedAdd(add PortfolioCorrRequest) (float64, string, error) {
	if add.Notional == 0 && strings.TrimSpace(add.Symbol) == "" {
		return 0, "", nil
	}
	if add.Notional <= 0 || math.IsNaN(add.Notional) || math.IsInf(add.Notional, 0) {
		return 0, "", fmt.Errorf("%w: addNotional must be positive", domain.ErrInvalidArgument)
	}
	symbol := strings.TrimSpace(add.Symbol)
	if symbol == "" {
		return 0, "", fmt.Errorf("%w: symbol is required", domain.ErrInvalidArgument)
	}
	ex := add.Exchange
	if ex == "" {
		ex = domain.ExchangeBinance
	}
	base, _ := domain.SplitBaseQuote(ex, symbol)
	if base == "" {
		return 0, "", fmt.Errorf("%w: symbol", domain.ErrInvalidArgument)
	}
	switch strings.ToLower(strings.TrimSpace(add.Side)) {
	case "", "buy", "long":
		return add.Notional, base, nil
	case "sell", "short":
		return -add.Notional, base, nil
	default:
		return 0, "", fmt.Errorf("%w: side must be buy, sell, long, or short", domain.ErrInvalidArgument)
	}
}

type exposureLine struct {
	Asset    string
	Exchange domain.Exchange
	Symbol   string
	Notional float64
	absPick  float64
}

func exposureLines(view *domain.PortfolioView) []exposureLine {
	if view == nil {
		return nil
	}
	by := map[string]*exposureLine{}
	add := func(ex domain.Exchange, symbol string, notional float64) {
		if math.Abs(notional) <= domain.PositionEpsilon || math.IsNaN(notional) {
			return
		}
		base, _ := domain.SplitBaseQuote(ex, symbol)
		if base == "" {
			return
		}
		cur := by[base]
		if cur == nil {
			cur = &exposureLine{Asset: base, Exchange: ex, Symbol: symbol}
			by[base] = cur
		}
		cur.Notional += notional
		if math.Abs(notional) > cur.absPick {
			cur.absPick = math.Abs(notional)
			cur.Exchange = ex
			cur.Symbol = symbol
		}
	}
	for _, pos := range view.Positions {
		add(pos.Exchange, pos.Symbol, pos.MarketValue)
	}
	for _, mp := range view.MarginPositions {
		if mp.Status != domain.MarginPositionOpen || mp.MarkPrice <= 0 || mp.Quantity <= 0 {
			continue
		}
		n := mp.Quantity * mp.MarkPrice
		if mp.Side == domain.MarginShort {
			n = -n
		}
		add(mp.Exchange, mp.Symbol, n)
	}
	out := make([]exposureLine, 0, len(by))
	for _, row := range by {
		if math.Abs(row.Notional) <= domain.PositionEpsilon {
			continue
		}
		out = append(out, *row)
	}
	sort.Slice(out, func(i, j int) bool {
		ai, aj := math.Abs(out[i].Notional), math.Abs(out[j].Notional)
		if ai != aj {
			return ai > aj
		}
		return out[i].Asset < out[j].Asset
	})
	return out
}

func mergeLine(lines []exposureLine, extra exposureLine) []exposureLine {
	for i := range lines {
		if lines[i].Asset == extra.Asset {
			return lines
		}
	}
	if extra.Exchange == "" {
		extra.Exchange = domain.ExchangeBinance
	}
	return append(lines, extra)
}

type pricedLine struct {
	exposureLine
	times []time.Time
	rets  []float64
	err   bool
}

func (s *Service) priceLines(ctx context.Context, lines []exposureLine) ([]pricedLine, []string, error) {
	if len(lines) == 0 {
		return nil, nil, nil
	}
	if s == nil || s.candles == nil {
		return nil, nil, fmt.Errorf("%w: candles not configured", domain.ErrUpstream)
	}
	spec := domain.PortfolioCorrSpec()
	out := make([]pricedLine, len(lines))
	var missing []string
	for i, line := range lines {
		out[i].exposureLine = line
		bars, err := s.candles.GetCandles(ctx, string(line.Exchange), line.Symbol, string(spec.Interval), spec.Bars, nil, nil)
		if err != nil {
			out[i].err = true
			if math.Abs(line.Notional) > domain.PositionEpsilon {
				missing = append(missing, line.Asset)
			}
			continue
		}
		times, rets := returnSeries(domain.PricePointsFromCandles(bars))
		out[i].times = times
		out[i].rets = rets
	}
	return out, missing, nil
}

func returnSeries(pts []domain.PricePoint) ([]time.Time, []float64) {
	if len(pts) < 2 {
		return nil, nil
	}
	times := make([]time.Time, 0, len(pts)-1)
	rets := make([]float64, 0, len(pts)-1)
	for i := 1; i < len(pts); i++ {
		prev := pts[i-1].Close
		if prev <= 0 || pts[i].Close <= 0 {
			return nil, nil
		}
		times = append(times, pts[i].Time.UTC())
		rets = append(rets, (pts[i].Close/prev-1)*100)
	}
	return times, rets
}

func joinReturns(rows []pricedLine) ([]domain.PortfolioCorrInput, []string) {
	var times []time.Time
	var used []pricedLine
	var missing []string
	order := append([]pricedLine(nil), rows...)
	sort.SliceStable(order, func(i, j int) bool {
		ai, aj := math.Abs(order[i].Notional), math.Abs(order[j].Notional)
		if ai != aj {
			return ai > aj
		}
		return order[i].Asset < order[j].Asset
	})
	for _, row := range order {
		if math.Abs(row.Notional) <= domain.PositionEpsilon {
			continue
		}
		if row.err || !domain.PortfolioCorrReady(len(row.rets)) {
			missing = append(missing, row.Asset)
			continue
		}
		if times == nil {
			times = row.times
			used = append(used, row)
			continue
		}
		next := sharedTimes(times, row.times)
		if !domain.PortfolioCorrReady(len(next)) {
			missing = append(missing, row.Asset)
			continue
		}
		times = next
		used = append(used, row)
	}
	inputs := make([]domain.PortfolioCorrInput, 0, len(used))
	for _, row := range used {
		rets := projectReturns(row.times, row.rets, times)
		if !domain.PortfolioCorrReady(len(rets)) {
			missing = append(missing, row.Asset)
			continue
		}
		inputs = append(inputs, domain.PortfolioCorrInput{
			Asset: row.Asset, Notional: row.Notional, Returns: rets,
		})
	}
	return inputs, missing
}

func sharedTimes(base, other []time.Time) []time.Time {
	has := map[int64]struct{}{}
	for _, t := range other {
		has[t.UTC().UnixMilli()] = struct{}{}
	}
	out := make([]time.Time, 0, len(base))
	for _, t := range base {
		if _, ok := has[t.UTC().UnixMilli()]; ok {
			out = append(out, t.UTC())
		}
	}
	return out
}

func projectReturns(times []time.Time, rets []float64, want []time.Time) []float64 {
	idx := map[int64]float64{}
	for i, t := range times {
		if i < len(rets) {
			idx[t.UTC().UnixMilli()] = rets[i]
		}
	}
	out := make([]float64, 0, len(want))
	for _, t := range want {
		v, ok := idx[t.UTC().UnixMilli()]
		if !ok {
			return nil
		}
		out = append(out, v)
	}
	return out
}

func addNotional(rows []pricedLine, asset string, signed float64) []pricedLine {
	out := append([]pricedLine(nil), rows...)
	for i := range out {
		if out[i].Asset != asset {
			continue
		}
		out[i].Notional += signed
		if math.Abs(out[i].Notional) <= domain.PositionEpsilon {
			out[i].err = true
		}
		return out
	}
	return out
}

func mergeNames(a, b []string) []string {
	seen := map[string]struct{}{}
	var out []string
	for _, name := range append(append([]string{}, a...), b...) {
		if name == "" {
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		out = append(out, name)
	}
	sort.Strings(out)
	if out == nil {
		return []string{}
	}
	return out
}

func containsName(names []string, asset string) bool {
	for _, name := range names {
		if name == asset {
			return true
		}
	}
	return false
}

func (s *Service) attachBookCorrelation(ctx context.Context, bookID string, out *domain.OrderPreview) {
	if s == nil || out == nil || s.store == nil || s.candles == nil || bookID == "" {
		return
	}
	notional := out.Notional
	if notional <= 0 || math.IsNaN(notional) {
		return
	}
	p, err := s.store.GetPortfolio(ctx, bookID)
	if err != nil || p == nil {
		return
	}
	view, err := s.buildView(ctx, p, domain.PortfolioRoleViewer)
	if err != nil || view == nil {
		return
	}
	rep, err := s.correlationFromView(ctx, view, PortfolioCorrRequest{
		Exchange: out.Exchange, Symbol: out.Symbol, Side: out.Side, Notional: notional,
	})
	if err != nil || rep == nil {
		return
	}
	out.Correlation = rep
}

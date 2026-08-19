package domain

import (
	"fmt"
	"math"
	"time"
)

const (
	SnapshotWindow1h  = LiquidationWindow1h
	SnapshotWindow4h  = LiquidationWindow4h
	SnapshotWindow24h = LiquidationWindow24h
)

// SnapshotWindows is 1h / 4h / 24h.
var SnapshotWindows = []struct {
	ID  string
	Dur time.Duration
}{
	{SnapshotWindow1h, time.Hour},
	{SnapshotWindow4h, 4 * time.Hour},
	{SnapshotWindow24h, 24 * time.Hour},
}

// SnapshotChange is current vs a lookback for one number.
type SnapshotChange struct {
	Window    string
	Current   float64
	Past      float64
	Change    float64
	ChangePct float64
	Direction string // up | down | flat
	Complete  bool
}

// SnapshotTaker is aggressive buy/sell in one window.
type SnapshotTaker struct {
	Window   string
	Buy      float64
	Sell     float64
	Delta    float64
	BuyShare float64
	Dominant string
	Complete bool
}

// SnapshotWindow is one lookback of the combined tape.
type SnapshotWindow struct {
	Window    string
	Price     SnapshotChange
	Volume    SnapshotChange // quote volume in this window vs the previous one
	MarketCap SnapshotChange
	OI        SnapshotChange
	Funding   SnapshotChange // rate as decimal; change is current−past
	LongPct   SnapshotChange // 0–100
	Taker     SnapshotTaker
}

// SnapshotVenue is futures metrics on one exchange.
type SnapshotVenue struct {
	Exchange Exchange
	OIValue  float64
	Funding  float64
	LongPct  float64
	Windows  []SnapshotWindow
	Summary  string
	Error    string
}

// SnapshotSpot is price, volume, and market cap (same coin, not per futures venue).
type SnapshotSpot struct {
	Price       float64
	Volume24h   float64
	MarketCap   float64
	Circulating float64
	Windows     []SnapshotWindow // price / volume / mcap filled; futures fields empty
}

// SnapshotReport is the combined tape for one coin.
type SnapshotReport struct {
	Symbol   string
	Exchange string
	AsOf     time.Time
	Spot     SnapshotSpot
	Venues   []SnapshotVenue
	Combined *SnapshotVenue
	Summary  string
	Note     string
}

// ChangeFromValues builds a window change.
func ChangeFromValues(window string, current, past float64, havePast bool) SnapshotChange {
	out := SnapshotChange{Window: window, Current: current}
	if !havePast || math.IsNaN(current) || math.IsNaN(past) {
		out.Direction = "flat"
		return out
	}
	out.Past = past
	out.Change = current - past
	if past != 0 {
		out.ChangePct = out.Change / math.Abs(past) * 100
	} else if current != 0 {
		out.ChangePct = 0
	}
	out.Direction = changeDir(out.ChangePct)
	out.Complete = true
	return out
}

func changeDir(pct float64) string {
	switch {
	case pct > 0.05:
		return "up"
	case pct < -0.05:
		return "down"
	default:
		return "flat"
	}
}

// PriceVolumeWindows computes price and quote-volume changes from OHLC bars.
func PriceVolumeWindows(bars []OHLCBar, now time.Time) []SnapshotWindow {
	out := make([]SnapshotWindow, 0, len(SnapshotWindows))
	if now.IsZero() {
		now = time.Now().UTC()
	}
	last := lastBar(bars)
	for _, w := range SnapshotWindows {
		row := SnapshotWindow{Window: w.ID}
		pastPx, okPx := closeAtOrBefore(bars, now.Add(-w.Dur))
		if last.Close > 0 && okPx && pastPx > 0 {
			row.Price = ChangeFromValues(w.ID, last.Close, pastPx, true)
		} else {
			row.Price = SnapshotChange{Window: w.ID, Current: last.Close, Direction: "flat"}
		}
		curVol := sumQuoteVol(bars, now.Add(-w.Dur), now)
		prevVol := sumQuoteVol(bars, now.Add(-2*w.Dur), now.Add(-w.Dur))
		row.Volume = ChangeFromValues(w.ID, curVol, prevVol, prevVol > 0 || curVol > 0)
		if prevVol <= 0 {
			row.Volume.Complete = curVol > 0
			row.Volume.ChangePct = 0
			row.Volume.Direction = "flat"
		}
		out = append(out, row)
	}
	return out
}

func lastBar(bars []OHLCBar) OHLCBar {
	if len(bars) == 0 {
		return OHLCBar{}
	}
	best := bars[0]
	for _, b := range bars[1:] {
		if b.Time.After(best.Time) {
			best = b
		}
	}
	return best
}

func closeAtOrBefore(bars []OHLCBar, target time.Time) (float64, bool) {
	var best OHLCBar
	found := false
	for _, b := range bars {
		if b.Time.IsZero() || b.Close <= 0 || b.Time.After(target) {
			continue
		}
		if !found || b.Time.After(best.Time) {
			best = b
			found = true
		}
	}
	if !found {
		return 0, false
	}
	// Allow one bar of slack so a 1h candle open just after the cut still counts.
	if best.Time.Before(target.Add(-90 * time.Minute)) {
		return best.Close, best.Time.After(target.Add(-2 * time.Hour))
	}
	return best.Close, true
}

func sumQuoteVol(bars []OHLCBar, from, to time.Time) float64 {
	var s float64
	for _, b := range bars {
		if b.Time.Before(from) || !b.Time.Before(to) {
			continue
		}
		if b.QuoteVol > 0 {
			s += b.QuoteVol
		}
	}
	return s
}

// ApplyMarketCap copies price-window percents onto circulating supply × price.
func ApplyMarketCap(windows []SnapshotWindow, circ, lastPx float64) {
	mcap := 0.0
	if circ > 0 && lastPx > 0 {
		mcap = circ * lastPx
	}
	for i := range windows {
		w := &windows[i]
		w.MarketCap.Window = w.Window
		w.MarketCap.Current = mcap
		if mcap > 0 && w.Price.Complete {
			w.MarketCap.ChangePct = w.Price.ChangePct
			w.MarketCap.Change = mcap * w.Price.ChangePct / (100 + w.Price.ChangePct)
			if math.IsNaN(w.MarketCap.Change) || math.IsInf(w.MarketCap.Change, 0) {
				w.MarketCap.Change = 0
			}
			w.MarketCap.Past = mcap - w.MarketCap.Change
			w.MarketCap.Direction = w.Price.Direction
			w.MarketCap.Complete = true
		}
	}
}

// OIWindowChange is current OI value vs ~dur ago.
func OIWindowChange(ser *OpenInterestSeries, window string, dur time.Duration, now time.Time) SnapshotChange {
	if ser == nil {
		return SnapshotChange{Window: window, Direction: "flat"}
	}
	cur := ser.Current.Value
	if cur <= 0 {
		cur = ser.Current.Contracts
	}
	past, ok := FindOpenInterestSample(ser.History, now.Add(-dur), OpenInterestSampleSlack(dur))
	pv := past.Value
	if pv <= 0 {
		pv = past.Contracts
	}
	return ChangeFromValues(window, cur, pv, ok && pv > 0)
}

// FundingWindowChange is current predicted rate vs a print ~dur ago.
func FundingWindowChange(ser *FundingSeries, window string, dur time.Duration, now time.Time) SnapshotChange {
	if ser == nil {
		return SnapshotChange{Window: window, Direction: "flat"}
	}
	cur := ser.Current.Rate
	past, ok := findFundingPast(ser, now.Add(-dur), OpenInterestSampleSlack(dur))
	ch := ChangeFromValues(window, cur, past, ok)
	// Funding is a small decimal; direction uses the rate delta, not % of rate.
	if ch.Complete {
		d := cur - past
		switch {
		case d > 1e-6:
			ch.Direction = "up"
		case d < -1e-6:
			ch.Direction = "down"
		default:
			ch.Direction = "flat"
		}
	}
	return ch
}

func findFundingPast(ser *FundingSeries, target time.Time, slack time.Duration) (float64, bool) {
	var best FundingPoint
	found := false
	pts := make([]FundingPoint, 0, len(ser.History)+1)
	pts = append(pts, ser.History...)
	if !ser.Current.Time.IsZero() && !ser.Current.Predicted {
		pts = append(pts, ser.Current)
	}
	for _, p := range pts {
		if p.Time.IsZero() || p.Time.After(target) {
			continue
		}
		if !found || p.Time.After(best.Time) {
			best = p
			found = true
		}
	}
	if !found {
		return 0, false
	}
	return best.Rate, !best.Time.Before(target.Add(-slack))
}

// LongPctWindowChange is account-long % vs ~dur ago.
func LongPctWindowChange(ser *LongShortSeries, window string, dur time.Duration, now time.Time) SnapshotChange {
	if ser == nil {
		return SnapshotChange{Window: window, Direction: "flat"}
	}
	cur := ser.Current.LongShare * 100
	past, ok := findLSPast(ser, now.Add(-dur), OpenInterestSampleSlack(dur))
	ch := ChangeFromValues(window, cur, past, ok)
	if ch.Complete {
		// Direction from percentage-point change of the long share.
		d := cur - past
		switch {
		case d > 0.3:
			ch.Direction = "up"
		case d < -0.3:
			ch.Direction = "down"
		default:
			ch.Direction = "flat"
		}
	}
	return ch
}

func findLSPast(ser *LongShortSeries, target time.Time, slack time.Duration) (float64, bool) {
	var best LongShortPoint
	found := false
	pts := append([]LongShortPoint{ser.Current}, ser.History...)
	for _, p := range pts {
		if p.Time.IsZero() || p.Time.After(target) {
			continue
		}
		if !found || p.Time.After(best.Time) {
			best = p
			found = true
		}
	}
	if !found {
		return 0, false
	}
	return best.LongShare * 100, !best.Time.Before(target.Add(-slack))
}

func takerForWindow(flow *TakerVenueFlow, window string) SnapshotTaker {
	out := SnapshotTaker{Window: window, Dominant: TakerSideEven}
	if flow == nil {
		return out
	}
	// 24h is not collected — leave incomplete.
	id := window
	if window == SnapshotWindow24h {
		return out
	}
	for _, w := range flow.Windows {
		if w.Window == id {
			return SnapshotTaker{
				Window: window, Buy: w.BuyNotional, Sell: w.SellNotional,
				Delta: w.Delta, BuyShare: w.BuyShare, Dominant: w.Dominant, Complete: w.Complete,
			}
		}
	}
	return out
}

// BuildSnapshotVenue fills futures windows for one venue.
func BuildSnapshotVenue(ex Exchange, spot []SnapshotWindow, oi *OpenInterestSeries, fund *FundingSeries, ls *LongShortSeries, taker *TakerVenueFlow, now time.Time) SnapshotVenue {
	out := SnapshotVenue{Exchange: ex, Windows: make([]SnapshotWindow, 0, len(SnapshotWindows))}
	if oi != nil {
		out.OIValue = oi.Current.Value
		if out.OIValue <= 0 {
			out.OIValue = oi.Current.Contracts
		}
	}
	if fund != nil {
		out.Funding = fund.Current.Rate
	}
	if ls != nil {
		out.LongPct = ls.Current.LongShare * 100
	}
	for i, w := range SnapshotWindows {
		row := SnapshotWindow{Window: w.ID}
		if i < len(spot) {
			row.Price = spot[i].Price
			row.Volume = spot[i].Volume
			row.MarketCap = spot[i].MarketCap
		}
		row.OI = OIWindowChange(oi, w.ID, w.Dur, now)
		row.Funding = FundingWindowChange(fund, w.ID, w.Dur, now)
		row.LongPct = LongPctWindowChange(ls, w.ID, w.Dur, now)
		row.Taker = takerForWindow(taker, w.ID)
		out.Windows = append(out.Windows, row)
	}
	out.Summary = ExplainSnapshotVenue(out)
	return out
}

// ExplainSnapshotVenue is a short 1h-led read of the tape.
func ExplainSnapshotVenue(v SnapshotVenue) string {
	ex := string(v.Exchange)
	if ex == string(ExchangeBinance) {
		ex = "Binance"
	} else if ex == string(ExchangeBybit) {
		ex = "Bybit"
	}
	var h1 *SnapshotWindow
	for i := range v.Windows {
		if v.Windows[i].Window == SnapshotWindow1h {
			h1 = &v.Windows[i]
		}
	}
	if h1 == nil {
		return ex + ": no snapshot window."
	}
	bits := make([]string, 0, 6)
	if h1.Price.Complete {
		bits = append(bits, "price "+h1.Price.Direction+" "+FormatSignedPct(h1.Price.ChangePct)+"%")
	}
	if h1.Volume.Complete {
		bits = append(bits, "volume "+h1.Volume.Direction)
	}
	if h1.OI.Complete {
		bits = append(bits, "OI "+h1.OI.Direction+" "+FormatSignedPct(h1.OI.ChangePct)+"%")
	}
	if h1.Taker.Complete && h1.Taker.Dominant != "" && h1.Taker.Dominant != TakerSideEven {
		bits = append(bits, "takers "+h1.Taker.Dominant)
	}
	if h1.Funding.Complete {
		bits = append(bits, "funding "+h1.Funding.Direction)
	}
	head := ex + " 1h: "
	if len(bits) == 0 {
		head += "not enough tape yet."
		return head
	}
	head += joinList(bits) + "."
	return head + " " + snapshotMeaning(*h1)
}

func snapshotMeaning(w SnapshotWindow) string {
	priceFlat := !w.Price.Complete || w.Price.Direction == "flat"
	oiUp := w.OI.Complete && w.OI.Direction == "up"
	oiDown := w.OI.Complete && w.OI.Direction == "down"
	volUp := w.Volume.Complete && w.Volume.Direction == "up"
	takerBuy := w.Taker.Complete && w.Taker.Dominant == TakerSideBuy
	takerSell := w.Taker.Complete && w.Taker.Dominant == TakerSideSell
	priceUp := w.Price.Complete && w.Price.Direction == "up"
	priceDown := w.Price.Complete && w.Price.Direction == "down"
	switch {
	case priceFlat && oiUp && volUp:
		return "Volume and open interest are building while price is still quiet — that can show up before a move."
	case priceFlat && oiUp && takerBuy:
		return "Open interest is rising and takers are buying while price is still quiet — that can show up before a move."
	case priceFlat && oiUp:
		return "Open interest is rising with a quiet price — positions are being added before the coin has moved much."
	case priceUp && oiUp && takerBuy:
		return "Price, OI, and taker buys are all up — longs are adding."
	case priceDown && oiUp && takerSell:
		return "Price is down with OI and taker sells up — shorts are adding."
	case priceUp && oiDown:
		return "Price is up while OI is down — more like shorts covering than new longs."
	case priceDown && oiDown:
		return "Price and OI are both down — longs are likely unwinding."
	case takerBuy && priceFlat:
		return "Takers are buying more than selling even though price has not gone far yet."
	case takerSell && priceFlat:
		return "Takers are selling more than buying even though price has not gone far yet."
	default:
		return "No strong lead from volume or OI versus price on this window."
	}
}

// ExplainSnapshotReport rolls spot + venues into one paragraph.
func ExplainSnapshotReport(symbol string, spot SnapshotSpot, venues []SnapshotVenue) string {
	name := prettyBase(symbol)
	var h1 *SnapshotWindow
	for i := range spot.Windows {
		if spot.Windows[i].Window == SnapshotWindow1h {
			h1 = &spot.Windows[i]
		}
	}
	head := name
	if h1 != nil && h1.Price.Complete {
		head += fmt.Sprintf(" 1h price %s%%", FormatSignedPct(h1.Price.ChangePct))
		if h1.Volume.Complete {
			head += ", volume " + h1.Volume.Direction
		}
	}
	head += "."
	for _, v := range venues {
		if v.Summary != "" && v.Error == "" {
			// Keep the first venue's 1h meaning if it has a lead.
			for _, w := range v.Windows {
				if w.Window == SnapshotWindow1h {
					mean := snapshotMeaning(w)
					if mean != "No strong lead from volume or OI versus price on this window." {
						return head + " " + mean
					}
				}
			}
		}
	}
	if len(venues) > 0 && venues[0].Summary != "" {
		return head + " " + venues[0].Summary
	}
	return head
}

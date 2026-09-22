package domain

import (
	"fmt"
	"time"
)

const (
	NetPositionWindow15m = "15m"
	NetPositionWindow1h  = LiquidationWindow1h
	NetPositionWindow4h  = LiquidationWindow4h
)

// NetPositionWindows are the lookbacks for the long/short split.
var NetPositionWindows = []struct {
	ID  string
	Dur time.Duration
}{
	{NetPositionWindow15m, 15 * time.Minute},
	{NetPositionWindow1h, time.Hour},
	{NetPositionWindow4h, 4 * time.Hour},
}

// NetPositionEstimate splits one window of taker buy/sell and the open-interest
// change into how much the long side and the short side grew or shrank.
//
// An aggressive buy is either a new long or a closed short. An aggressive sell
// is either a new short or a closed long. Open interest rises when opens exceed
// closes. Those three facts fix:
//
//	netLong  = (buy − sell + oiDelta) / 2
//	netShort = (sell − buy + oiDelta) / 2
//
// Positive means that side grew. Negative means that side shrank.
// The two nets add up to the open-interest change.
type NetPositionEstimate struct {
	Buy         float64
	Sell        float64
	OIDelta     float64
	NetLong     float64
	NetShort    float64
	Unexplained float64 // OI change beyond what taker volume can account for
	Complete    bool
}

// EstimateNetPosition splits buy, sell, and the USDT open-interest change.
// When |oiDelta| is larger than buy+sell, the extra is Unexplained and the
// split uses only the part taker volume can explain.
func EstimateNetPosition(buy, sell, oiDelta float64) NetPositionEstimate {
	if buy < 0 {
		buy = 0
	}
	if sell < 0 {
		sell = 0
	}
	out := NetPositionEstimate{Buy: buy, Sell: sell, OIDelta: oiDelta}
	vol := buy + sell
	if vol <= 0 {
		out.Unexplained = oiDelta
		return out
	}
	used := oiDelta
	if used > vol {
		out.Unexplained = used - vol
		used = vol
	} else if used < -vol {
		out.Unexplained = used + vol
		used = -vol
	}
	out.NetLong = (buy - sell + used) / 2
	out.NetShort = (sell - buy + used) / 2
	out.Complete = true
	return out
}

// NetPositionWindow is one lookback of the combined read.
type NetPositionWindow struct {
	Window      string  `json:"window"`
	NetLong     float64 `json:"netLong"`
	NetShort    float64 `json:"netShort"`
	Buy         float64 `json:"buy"`
	Sell        float64 `json:"sell"`
	OIDelta     float64 `json:"oiDelta"`
	Unexplained float64 `json:"unexplained,omitempty"`
	Complete    bool    `json:"complete"`
	Summary     string  `json:"summary"`
}

// NetPositionVenue is one exchange's windows.
type NetPositionVenue struct {
	Exchange string              `json:"exchange"`
	Windows  []NetPositionWindow `json:"windows"`
	Error    string              `json:"error,omitempty"`
}

// NetPositionReport is the combined long/short estimate.
type NetPositionReport struct {
	Symbol   string              `json:"symbol"`
	Exchange string              `json:"exchange"`
	AsOf     time.Time           `json:"asOf"`
	Windows  []NetPositionWindow `json:"windows"`
	Venues   []NetPositionVenue  `json:"venues,omitempty"`
	Summary  string              `json:"summary"`
	Note     string              `json:"note"`
}

// NetPositionVenueInput is one venue's open-interest series and taker bars.
type NetPositionVenueInput struct {
	Exchange Exchange
	Series   *OpenInterestSeries
	Buckets  []TakerBucket
	Bar      time.Duration
	Error    string
}

const netPositionNote = "Net long and net short are estimated from aggressive buy/sell volume and the open-interest change over the same window. A positive number means that side grew; a negative number means it shrank. They add up to the open-interest change. This is not the account long/short ratio. Informational only — not financial advice."

// BuildNetPosition sums venues into 15m / 1h / 4h net long and net short.
func BuildNetPosition(symbol, exchange string, venues []NetPositionVenueInput, now time.Time) *NetPositionReport {
	if now.IsZero() {
		now = time.Now()
	}
	now = now.UTC()
	out := &NetPositionReport{
		Symbol:   NormalizeLiquidationSymbol(symbol),
		Exchange: exchange,
		AsOf:     now,
		Windows:  netPositionWindows(venues, now),
		Note:     netPositionNote,
	}
	for _, v := range venues {
		row := NetPositionVenue{Exchange: string(v.Exchange), Error: v.Error}
		if v.Error == "" && v.Series != nil {
			row.Windows = netPositionWindows([]NetPositionVenueInput{v}, now)
		}
		out.Venues = append(out.Venues, row)
	}
	out.Summary = explainNetPositionReport(out.Windows)
	return out
}

func netPositionWindows(venues []NetPositionVenueInput, now time.Time) []NetPositionWindow {
	out := make([]NetPositionWindow, 0, len(NetPositionWindows))
	for _, spec := range NetPositionWindows {
		var buy, sell, oi float64
		used := 0
		complete := len(venues) > 0
		for _, v := range venues {
			if v.Error != "" || v.Series == nil {
				complete = false
				continue
			}
			delta, oiFound, oiFresh := openInterestValueDelta(v.Series, spec.Dur, now)
			tb, ts, takerOK := TakerNotionalInWindow(v.Buckets, now, spec.Dur, v.Bar)
			if !oiFound {
				complete = false
				continue
			}
			if !oiFresh || !takerOK {
				complete = false
			}
			buy += tb
			sell += ts
			oi += delta
			used++
		}
		w := NetPositionWindow{Window: spec.ID, Buy: buy, Sell: sell, OIDelta: oi}
		if used > 0 {
			est := EstimateNetPosition(buy, sell, oi)
			w.NetLong = est.NetLong
			w.NetShort = est.NetShort
			w.Unexplained = est.Unexplained
			w.Complete = complete && est.Complete
		}
		w.Summary = explainNetPositionWindow(w)
		out = append(out, w)
	}
	return out
}

func explainNetPositionWindow(w NetPositionWindow) string {
	if !w.Complete && w.Buy+w.Sell == 0 && w.OIDelta == 0 {
		return w.Window + ": not enough open interest and taker volume yet."
	}
	s := fmt.Sprintf("%s: longs %s, shorts %s (open interest %s).",
		w.Window, FormatSignedQty(w.NetLong), FormatSignedQty(w.NetShort), FormatSignedQty(w.OIDelta))
	if w.Unexplained != 0 {
		s += fmt.Sprintf(" %s of the open-interest change is outside taker volume.", FormatSignedQty(w.Unexplained))
	}
	return s
}

func explainNetPositionReport(windows []NetPositionWindow) string {
	for _, id := range []string{NetPositionWindow1h, NetPositionWindow15m, NetPositionWindow4h} {
		for _, w := range windows {
			if w.Window == id && w.Summary != "" {
				return w.Summary
			}
		}
	}
	return "Net long and net short need open interest and taker volume."
}

func openInterestValueDelta(s *OpenInterestSeries, dur time.Duration, now time.Time) (delta float64, found, fresh bool) {
	if s == nil {
		return 0, false, false
	}
	cur := s.Current
	if cur.Time.IsZero() && cur.Value == 0 && cur.Contracts == 0 {
		return 0, false, false
	}
	past, fresh := FindOpenInterestSample(s.History, now.Add(-dur), OpenInterestSampleSlack(dur))
	if past.Time.IsZero() {
		return 0, false, false
	}
	return cur.Value - past.Value, true, fresh
}

// TakerNotionalInWindow sums aggressive buy and sell whose bar overlaps the lookback.
// The bool is true when the bars reach back to the start of the window.
func TakerNotionalInWindow(buckets []TakerBucket, now time.Time, dur, bar time.Duration) (buy, sell float64, complete bool) {
	if bar <= 0 {
		bar = time.Minute
	}
	now = now.UTC()
	cut := now.Add(-dur)
	var earliest time.Time
	for _, b := range buckets {
		if b.Start.IsZero() {
			continue
		}
		end := b.Start.Add(bar)
		if !end.After(cut) || !b.Start.Before(now) {
			continue
		}
		buy += b.BuyNotional
		sell += b.SellNotional
		if earliest.IsZero() || b.Start.Before(earliest) {
			earliest = b.Start
		}
	}
	if earliest.IsZero() {
		return 0, 0, false
	}
	return buy, sell, !earliest.After(cut.Add(bar))
}

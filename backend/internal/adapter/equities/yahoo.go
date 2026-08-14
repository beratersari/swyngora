package equities

import (
	"encoding/json"
	"fmt"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

type yahooSparkResponse struct {
	Spark struct {
		Result []yahooSparkResult `json:"result"`
		Error  *struct {
			Description string `json:"description"`
		} `json:"error"`
	} `json:"spark"`
}

type yahooSparkResult struct {
	Symbol   string `json:"symbol"`
	Response []struct {
		Meta yahooChartMeta `json:"meta"`
	} `json:"response"`
}

type yahooChartMeta struct {
	Symbol               string  `json:"symbol"`
	Currency             string  `json:"currency"`
	ShortName            string  `json:"shortName"`
	InstrumentType       string  `json:"instrumentType"`
	RegularMarketPrice   float64 `json:"regularMarketPrice"`
	RegularMarketDayHigh float64 `json:"regularMarketDayHigh"`
	RegularMarketDayLow  float64 `json:"regularMarketDayLow"`
	RegularMarketVolume  float64 `json:"regularMarketVolume"`
	RegularMarketTime    int64   `json:"regularMarketTime"`
	ChartPreviousClose   float64 `json:"chartPreviousClose"`
}

func quoteFromMeta(meta yahooChartMeta) yahooQuote {
	prev := meta.ChartPreviousClose
	chg := 0.0
	pct := 0.0
	if prev > 0 && meta.RegularMarketPrice > 0 {
		chg = meta.RegularMarketPrice - prev
		pct = (chg / prev) * 100
	}
	return yahooQuote{
		Symbol:                     meta.Symbol,
		ShortName:                  meta.ShortName,
		QuoteType:                  meta.InstrumentType,
		Currency:                   meta.Currency,
		RegularMarketPrice:         meta.RegularMarketPrice,
		RegularMarketChange:        chg,
		RegularMarketChangePercent: pct,
		RegularMarketDayHigh:       meta.RegularMarketDayHigh,
		RegularMarketDayLow:        meta.RegularMarketDayLow,
		RegularMarketVolume:        meta.RegularMarketVolume,
		RegularMarketOpen:          prev,
		RegularMarketPreviousClose: prev,
		RegularMarketTime:          meta.RegularMarketTime,
	}
}

type yahooQuote struct {
	Symbol                     string  `json:"symbol"`
	ShortName                  string  `json:"shortName"`
	QuoteType                  string  `json:"quoteType"`
	Sector                     string  `json:"sector"`
	Currency                   string  `json:"currency"`
	RegularMarketPrice         float64 `json:"regularMarketPrice"`
	RegularMarketChange        float64 `json:"regularMarketChange"`
	RegularMarketChangePercent float64 `json:"regularMarketChangePercent"`
	RegularMarketDayHigh       float64 `json:"regularMarketDayHigh"`
	RegularMarketDayLow        float64 `json:"regularMarketDayLow"`
	RegularMarketVolume        float64 `json:"regularMarketVolume"`
	RegularMarketOpen          float64 `json:"regularMarketOpen"`
	RegularMarketPreviousClose float64 `json:"regularMarketPreviousClose"`
	RegularMarketTime          int64   `json:"regularMarketTime"`
	MarketCap                  float64 `json:"marketCap"`
}

type yahooChartResponse struct {
	Chart struct {
		Result []yahooChartResult `json:"result"`
		Error  *struct {
			Description string `json:"description"`
		} `json:"error"`
	} `json:"chart"`
}

type yahooChartResult struct {
	Timestamp  []int64 `json:"timestamp"`
	Indicators struct {
		Quote []struct {
			Open   []*float64 `json:"open"`
			High   []*float64 `json:"high"`
			Low    []*float64 `json:"low"`
			Close  []*float64 `json:"close"`
			Volume []*float64 `json:"volume"`
		} `json:"quote"`
	} `json:"indicators"`
}

func parseQuotes(body []byte) ([]yahooQuote, error) {
	var wrap yahooSparkResponse
	if err := json.Unmarshal(body, &wrap); err != nil {
		return nil, fmt.Errorf("%w: yahoo spark json: %v", domain.ErrUpstream, err)
	}
	if wrap.Spark.Error != nil && wrap.Spark.Error.Description != "" {
		return nil, fmt.Errorf("%w: yahoo spark: %s", domain.ErrUpstream, wrap.Spark.Error.Description)
	}
	out := make([]yahooQuote, 0, len(wrap.Spark.Result))
	for _, r := range wrap.Spark.Result {
		if len(r.Response) == 0 {
			continue
		}
		q := quoteFromMeta(r.Response[0].Meta)
		if q.Symbol == "" {
			q.Symbol = r.Symbol
		}
		out = append(out, q)
	}
	return out, nil
}

func parseChart(body []byte, limit int) ([]domain.Candle, error) {
	var wrap yahooChartResponse
	if err := json.Unmarshal(body, &wrap); err != nil {
		return nil, fmt.Errorf("%w: yahoo chart json: %v", domain.ErrUpstream, err)
	}
	if wrap.Chart.Error != nil && wrap.Chart.Error.Description != "" {
		return nil, fmt.Errorf("%w: yahoo chart: %s", domain.ErrUpstream, wrap.Chart.Error.Description)
	}
	if len(wrap.Chart.Result) == 0 {
		return nil, fmt.Errorf("%w: empty yahoo chart", domain.ErrNotFound)
	}
	res := wrap.Chart.Result[0]
	if len(res.Indicators.Quote) == 0 {
		return nil, fmt.Errorf("%w: empty yahoo chart quote", domain.ErrNotFound)
	}
	q := res.Indicators.Quote[0]
	n := len(res.Timestamp)
	out := make([]domain.Candle, 0, n)
	for i := 0; i < n; i++ {
		if i >= len(q.Close) || q.Close[i] == nil {
			continue
		}
		open, high, low, vol := 0.0, 0.0, 0.0, 0.0
		if i < len(q.Open) && q.Open[i] != nil {
			open = *q.Open[i]
		}
		if i < len(q.High) && q.High[i] != nil {
			high = *q.High[i]
		}
		if i < len(q.Low) && q.Low[i] != nil {
			low = *q.Low[i]
		}
		if i < len(q.Volume) && q.Volume[i] != nil {
			vol = *q.Volume[i]
		}
		closePx := *q.Close[i]
		ts := time.Unix(res.Timestamp[i], 0).UTC()
		out = append(out, domain.Candle{
			OpenTime:    ts,
			Open:        fmtFloat(open),
			High:        fmtFloat(high),
			Low:         fmtFloat(low),
			Close:       fmtFloat(closePx),
			Volume:      fmtFloat(vol),
			CloseTime:   ts,
			QuoteVolume: fmtFloat(closePx * vol),
		})
	}
	if limit > 0 && len(out) > limit {
		out = out[len(out)-limit:]
	}
	return out, nil
}

func yahooChartWindow(iv domain.CandleInterval) (interval, rng string) {
	switch iv {
	case domain.Interval1m:
		return "1m", "1d"
	case domain.Interval5m:
		return "5m", "5d"
	case domain.Interval15m:
		return "15m", "5d"
	case domain.Interval30m:
		return "30m", "1mo"
	case domain.Interval1h:
		return "60m", "1mo"
	case domain.Interval1d:
		return "1d", "2y"
	case domain.Interval1w:
		return "1wk", "5y"
	case domain.Interval1M:
		return "1mo", "10y"
	default:
		return "1d", "1y"
	}
}

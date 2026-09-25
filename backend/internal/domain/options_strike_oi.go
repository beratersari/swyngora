package domain

import (
	"sort"
	"strconv"
	"time"
)

// OptionsStrikeOIPoint is call and put open interest at one listed strike.
type OptionsStrikeOIPoint struct {
	Expiry time.Time
	Strike float64
	CallOI float64
	PutOI  float64
}

// OptionsStrikeBook is one sampled strike ladder for an asset.
type OptionsStrikeBook struct {
	SampledAt time.Time
	Points    []OptionsStrikeOIPoint
}

// OptionsStrikeOIRow is how call and put open interest at one strike moved.
type OptionsStrikeOIRow struct {
	Expiry     time.Time
	Label      string
	Strike     float64
	CallOI     float64
	PutOI      float64
	CallPast   float64
	PutPast    float64
	CallChange float64
	PutChange  float64
	NetChange  float64
	Opened     bool
	Closed     bool
}

// OptionsStrikeOIWindow compares the live strike book with one earlier sample.
type OptionsStrikeOIWindow struct {
	Window     string
	ComparedAt time.Time
	Complete   bool
	Strikes    []OptionsStrikeOIRow
}

// OptionsStrikeOI is open-interest change by strike over 1h / 4h / 24h / 7d.
type OptionsStrikeOI struct {
	Windows []OptionsStrikeOIWindow
	Note    string
}

// OptionsStrikePointsFromSnapshot copies listed call/put OI. Zero rows are dropped.
func OptionsStrikePointsFromSnapshot(snap *OptionsSnapshot) []OptionsStrikeOIPoint {
	if snap == nil {
		return nil
	}
	out := make([]OptionsStrikeOIPoint, 0)
	for _, e := range snap.Expiries {
		for _, s := range e.Strikes {
			if s.CallOI <= 0 && s.PutOI <= 0 {
				continue
			}
			out = append(out, OptionsStrikeOIPoint{
				Expiry: e.Expiry.UTC(),
				Strike: s.Strike,
				CallOI: s.CallOI,
				PutOI:  s.PutOI,
			})
		}
	}
	return out
}

// BuildOptionsStrikeOI diffs the live book against each window's stored ladder.
// A window stays incomplete when that ladder is missing or older than the slack.
func BuildOptionsStrikeOI(snap *OptionsSnapshot, books map[string]OptionsStrikeBook, now time.Time) *OptionsStrikeOI {
	if snap == nil {
		return nil
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	now = now.UTC()
	labels := map[int64]string{}
	for _, e := range snap.Expiries {
		labels[e.Expiry.UTC().Unix()] = e.Label
	}
	current := OptionsStrikePointsFromSnapshot(snap)
	out := &OptionsStrikeOI{
		Windows: make([]OptionsStrikeOIWindow, 0, len(OptionsHistoryWindows)),
		Note:    "Call and put open interest by strike is sampled while the desk runs (5-minute buckets). Change is the live book minus that earlier ladder. A strike that appears is opened; one that leaves is closed. Informational only — not financial advice.",
	}
	for _, spec := range OptionsHistoryWindows {
		win := OptionsStrikeOIWindow{Window: spec.ID}
		book, ok := books[spec.ID]
		target := now.Add(-spec.Dur)
		if ok && !book.SampledAt.IsZero() && target.Sub(book.SampledAt) <= optionsHistorySlack(spec.Dur) {
			win.Complete = true
			win.ComparedAt = book.SampledAt.UTC()
			win.Strikes = diffStrikeOI(current, book.Points, labels)
		}
		out.Windows = append(out.Windows, win)
	}
	return out
}

func diffStrikeOI(current, past []OptionsStrikeOIPoint, labels map[int64]string) []OptionsStrikeOIRow {
	type side struct {
		call float64
		put  float64
	}
	cur := map[string]side{}
	was := map[string]side{}
	meta := map[string]OptionsStrikeOIPoint{}
	for _, p := range current {
		k := strikeOIKey(p.Expiry, p.Strike)
		cur[k] = side{call: p.CallOI, put: p.PutOI}
		meta[k] = p
	}
	for _, p := range past {
		k := strikeOIKey(p.Expiry, p.Strike)
		was[k] = side{call: p.CallOI, put: p.PutOI}
		if _, ok := meta[k]; !ok {
			meta[k] = p
		}
	}
	rows := make([]OptionsStrikeOIRow, 0, len(meta))
	for k, p := range meta {
		c, w := cur[k], was[k]
		if c.call <= 0 && c.put <= 0 && w.call <= 0 && w.put <= 0 {
			continue
		}
		row := OptionsStrikeOIRow{
			Expiry:     p.Expiry.UTC(),
			Label:      labels[p.Expiry.UTC().Unix()],
			Strike:     p.Strike,
			CallOI:     c.call,
			PutOI:      c.put,
			CallPast:   w.call,
			PutPast:    w.put,
			CallChange: c.call - w.call,
			PutChange:  c.put - w.put,
			NetChange:  (c.call + c.put) - (w.call + w.put),
		}
		if row.Label == "" {
			row.Label = p.Expiry.UTC().Format("02 Jan 2006")
		}
		pastTotal := w.call + w.put
		curTotal := c.call + c.put
		row.Opened = pastTotal <= 0 && curTotal > 0
		row.Closed = curTotal <= 0 && pastTotal > 0
		rows = append(rows, row)
	}
	sort.Slice(rows, func(i, j int) bool {
		if !rows[i].Expiry.Equal(rows[j].Expiry) {
			return rows[i].Expiry.Before(rows[j].Expiry)
		}
		return rows[i].Strike < rows[j].Strike
	})
	return rows
}

func strikeOIKey(expiry time.Time, strike float64) string {
	return strconv.FormatInt(expiry.UTC().Unix(), 10) + "|" + strconv.FormatFloat(strike, 'f', 4, 64)
}

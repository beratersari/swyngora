package domain

import (
	"strings"
	"time"
)

const ForensicsDisclaimer = "One tape of what happened before, during, and after a chosen time — price, volume, takers, VWAP, vs typical, POC, sweeps, stored book/futures (open interest, funding, liquidations), and listed BTC/ETH options when those samples exist — plus past important-move setups that looked like the before-window, and how price moved after those. Informational only — not financial advice."

// ForensicsReport is the around-the-move tape plus similar past setups for one event.
type ForensicsReport struct {
	Symbol       string
	Exchange     string
	At           time.Time
	Window       string
	During       string
	AsOf         time.Time
	Tape         *AroundReport
	Similar      *AroundSimilarReport
	SimilarError string
	Why          ForensicsWhy
	Summary      string
	Note         string
}

// ForensicsWhy is a plain-language read of why the move may have happened.
type ForensicsWhy struct {
	Headline string
	Before   string
	During   string
	After    string
	Similar  string
	Caution  string
}

// ExplainForensics joins the why-story into one paragraph for summaries and tools.
func ExplainForensics(r ForensicsReport) string {
	w := r.Why
	if w.Headline == "" && w.Before == "" && w.During == "" {
		w = ExplainForensicsWhy(r)
	}
	var parts []string
	for _, s := range []string{w.Headline, w.Before, w.During, w.After, w.Similar} {
		if t := strings.TrimSpace(s); t != "" {
			parts = append(parts, t)
		}
	}
	if len(parts) == 0 {
		return "No forensics tape for this time."
	}
	return strings.Join(parts, " ")
}

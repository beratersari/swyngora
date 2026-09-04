package domain

import (
	"strings"
	"time"
)

const ForensicsDisclaimer = "One tape of what happened before, during, and after a chosen time — price, volume, takers, VWAP, vs typical, POC, sweeps, stored book/futures (open interest, funding, liquidations) — plus past important-move setups that looked like the before-window, and how price moved after those. Informational only — not financial advice."

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
	Summary      string
	Note         string
}

// ExplainForensics joins the tape and similar-move summaries.
func ExplainForensics(r ForensicsReport) string {
	var parts []string
	if r.Tape != nil && strings.TrimSpace(r.Tape.Summary) != "" {
		parts = append(parts, strings.TrimSpace(r.Tape.Summary))
	}
	if r.Similar != nil && strings.TrimSpace(r.Similar.Summary) != "" {
		parts = append(parts, strings.TrimSpace(r.Similar.Summary))
	} else if strings.TrimSpace(r.SimilarError) != "" {
		parts = append(parts, "Similar past moves could not be loaded.")
	}
	if len(parts) == 0 {
		return "No forensics tape for this time."
	}
	return strings.Join(parts, " ")
}

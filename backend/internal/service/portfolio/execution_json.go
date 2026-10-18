package portfolio

import (
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

// ExecutionComparePayload is the JSON body for the three prices. Nothing was written.
func ExecutionComparePayload(c *domain.ExecutionCompare) map[string]any {
	if c == nil {
		return map[string]any{}
	}
	rows := make([]any, 0, len(c.Rows))
	for _, row := range c.Rows {
		rows = append(rows, executionQuotePayload(row))
	}
	out := map[string]any{
		"side":           string(c.Side),
		"symbol":         c.Symbol,
		"quantity":       c.Quantity,
		"exchange":       string(c.Venue),
		"touchPrice":     c.TouchPrice,
		"touchExchange":  string(c.TouchExchange),
		"maxSlippagePct": c.MaxSlippagePct,
		"rows":           rows,
		"note":           c.Note,
	}
	if c.ID != "" {
		out["id"] = c.ID
	}
	return out
}

// FillCostPayload is the quote beside what the finished order paid.
// Deltas are actual minus expected. Positive means the fill cost more.
func FillCostPayload(c *domain.FillCostCheck) map[string]any {
	if c == nil || c.Status != domain.FillCostComplete {
		return nil
	}
	return map[string]any{
		"style":            string(c.Style),
		"expectedFee":      c.ExpectedFee,
		"actualFee":        c.ActualFee,
		"feeDelta":         c.FeeDelta(),
		"expectedSlippage": c.ExpectedSlippage,
		"actualSlippage":   c.ActualSlippage,
		"slippageDelta":    c.SlippageDelta(),
		"touchPrice":       c.TouchPrice,
		"quotedQuantity":   c.QuotedQuantity,
		"filledQuantity":   c.FilledQuantity,
		"quotedAt":         c.QuotedAt.UTC().Format(time.RFC3339Nano),
	}
}

func executionQuotePayload(row domain.ExecutionQuote) map[string]any {
	legs := make([]any, 0, len(row.Legs))
	for _, leg := range row.Legs {
		legs = append(legs, routeLegPayload(leg))
	}
	out := map[string]any{
		"style":      string(row.Style),
		"status":     string(row.Status),
		"exchange":   string(row.Exchange),
		"symbol":     row.Symbol,
		"quantity":   row.Quantity,
		"touchPrice": row.TouchPrice,
		"legs":       legs,
	}
	if row.Status == domain.ExecutionFilled {
		out["filledQuantity"] = row.FilledQuantity
		out["averagePrice"] = row.AveragePrice
		out["slippagePct"] = row.SlippagePct
		out["notional"] = row.Notional
		out["fee"] = row.Fee
		out["feePct"] = row.FeePct
		out["cashDebit"] = row.CashDebit
		out["cashCredit"] = row.CashCredit
		out["unitCash"] = row.UnitCash
		out["levelsUsed"] = row.LevelsUsed
	}
	if row.SavedVsMarket != nil {
		out["savedVsMarket"] = *row.SavedVsMarket
	}
	if len(row.Missing) > 0 {
		out["missing"] = row.Missing
	}
	if row.Detail != "" {
		out["detail"] = row.Detail
	}
	if row.Schedule != "" {
		out["schedule"] = row.Schedule
	}
	if row.Style == domain.ExecutionSpread {
		out["spreadMinutes"] = row.SpreadMinutes
		out["sliceCount"] = row.SliceCount
		if row.SliceQuantity > 0 {
			out["sliceQuantity"] = row.SliceQuantity
		}
		if row.Assumption != "" {
			out["assumption"] = row.Assumption
		}
	}
	return out
}

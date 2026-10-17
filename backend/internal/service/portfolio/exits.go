package portfolio

import (
	"fmt"
	"strconv"
	"strings"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

func targetsNeedReference(targets []domain.TakeProfitTarget) bool {
	for _, t := range targets {
		if t.OffsetPercent > 0 && t.Price <= 0 {
			return true
		}
	}
	return false
}

func targetKey(targets []domain.ResolvedTakeProfit) string {
	var b strings.Builder
	for _, t := range targets {
		b.WriteString(strconv.FormatFloat(t.Percent, 'g', 16, 64))
		b.WriteByte('@')
		b.WriteString(strconv.FormatFloat(t.Price, 'g', 16, 64))
		b.WriteByte(';')
	}
	return b.String()
}

func ocoIdempotencyHash(ex domain.Exchange, sym string, in OCOOrderInput, resolved []domain.ResolvedTakeProfit) string {
	if len(in.TakeProfitTargets) == 0 && !in.MoveStopToEntry {
		return hashParts("oco", string(ex), sym, in.Quantity, in.TakeProfitPrice, in.StopLossPrice, in.LotMethod)
	}
	if in.MoveStopToEntry {
		return hashParts("oco-targets", string(ex), sym, in.Quantity, in.StopLossPrice, in.LotMethod, targetKey(resolved), "move-stop")
	}
	return hashParts("oco-targets", string(ex), sym, in.Quantity, in.StopLossPrice, in.LotMethod, targetKey(resolved))
}

func bracketIdempotencyHash(ex domain.Exchange, sym string, in BracketOrderInput, resolved []domain.ResolvedTakeProfit) string {
	if len(in.TakeProfitTargets) == 0 && !in.MoveStopToEntry {
		return hashParts("bracket", string(ex), sym, in.Quantity, in.EntryPrice, in.TakeProfitPrice, in.StopLossPrice, in.LotMethod, in.Reason.HashPart())
	}
	if in.MoveStopToEntry {
		return hashParts("bracket-targets", string(ex), sym, in.Quantity, in.EntryPrice, in.StopLossPrice, in.LotMethod, in.Reason.HashPart(), targetKey(resolved), "move-stop")
	}
	return hashParts("bracket-targets", string(ex), sym, in.Quantity, in.EntryPrice, in.StopLossPrice, in.LotMethod, in.Reason.HashPart(), targetKey(resolved))
}

func ocoBuyPrice(move bool, avg float64) float64 {
	if !move {
		return 0
	}
	return avg
}

func rejectMoveStop(move bool, buy, stop float64, resolved []domain.ResolvedTakeProfit) error {
	if !move {
		return nil
	}
	first := 0.0
	if len(resolved) > 0 {
		first = resolved[0].Price
	}
	return domain.ValidateMoveStopToEntry(buy, stop, first)
}

func exitPreviewLegs(
	targets []domain.ResolvedTakeProfit,
	stop float64,
	fullQty float64,
	cost domain.TradingCost,
	lots []domain.TaxLot,
	bookID string,
	ex domain.Exchange,
	sym string,
	posQty, avg float64,
	lotMethod domain.LotMethod,
	skipPnL bool,
) []domain.OrderPreviewLeg {
	legs := make([]domain.OrderPreviewLeg, 0, len(targets)+1)
	for _, t := range targets {
		leg := sellLeg("take_profit", t.Price, t.Quantity, cost, lots, bookID, ex, sym, posQty, avg, lotMethod, skipPnL)
		leg.Quantity = t.Quantity
		leg.Percent = t.Percent
		leg.OffsetPercent = t.OffsetPercent
		legs = append(legs, leg)
	}
	stopLeg := sellLeg("stop_loss", stop, fullQty, cost, lots, bookID, ex, sym, posQty, avg, lotMethod, skipPnL)
	stopLeg.Quantity = fullQty
	stopLeg.Percent = 100
	legs = append(legs, stopLeg)
	return legs
}

func ocoOffsetReference(avg, last float64, targets []domain.TakeProfitTarget) (float64, error) {
	if avg > 0 {
		return avg, nil
	}
	if targetsNeedReference(targets) && last <= 0 {
		return 0, fmt.Errorf("%w: a reference price is required for a gain percent", domain.ErrInvalidArgument)
	}
	return last, nil
}

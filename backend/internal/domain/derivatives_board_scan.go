package domain

import (
	"fmt"
	"strings"
	"time"
)

// DayCVDDelta sums aggressive buy−sell since 00:00 UTC.
// ok is false until the tape reaches that open. newest is the last bucket in the day.
func DayCVDDelta(buckets []TakerBucket, now, started time.Time) (delta float64, newest time.Time, ok bool) {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	now = now.UTC()
	from := now.Truncate(24 * time.Hour)
	if !cvdCoverageComplete(started, from) {
		return 0, time.Time{}, false
	}
	for _, b := range buckets {
		if b.Start.Before(from) {
			continue
		}
		delta += b.BuyNotional - b.SellNotional
		if b.Start.After(newest) {
			newest = b.Start
		}
	}
	if newest.IsZero() {
		return 0, time.Time{}, false
	}
	return delta, newest, true
}

func applyBoardScan(row *DerivativesCoin, expected []Exchange, byEx map[Exchange]DerivativesVenueInput, liqCover MarketCoverage, now time.Time) {
	if row == nil {
		return
	}
	applyBoardGap(row, expected, byEx)
	applyBoardCVD(row, expected, byEx, now)
	applyBoardPrint(row, expected, byEx, now)
	applyBoardOrderCount(row, expected, byEx, now)
	applyBoardLiqHeat(row, expected, byEx, liqCover)
	applyBoardBid(row, expected, byEx, now)
}

func boardVenueRow(row *DerivativesCoin, ex Exchange) *DerivativesVenueRow {
	if row == nil {
		return nil
	}
	for i := range row.Venues {
		if row.Venues[i].Exchange == string(ex) {
			return &row.Venues[i]
		}
	}
	return nil
}

func applyBoardGap(row *DerivativesCoin, expected []Exchange, byEx map[Exchange]DerivativesVenueInput) {
	var gaps, weights []float64
	var scope []string
	allPublishersFresh := true
	publishers := 0
	for _, ex := range expected {
		v := byEx[ex]
		status := VenueReadMissing
		var gap float64
		switch {
		case v.TopUnsupported:
			status = VenueReadUnsupported
		case v.TopLongOK:
			status = boardLongStatus(row, ex)
			if status == VenueReadFresh {
				gap = (v.TopLongShare - v.LongShare) * 100
			}
		}
		if cell := boardVenueRow(row, ex); cell != nil {
			cell.GapStatus = status
			if status == VenueReadFresh {
				cell.GapPct = FormatSignedPct(gap)
			}
		}
		if status == VenueReadUnsupported {
			continue
		}
		publishers++
		if status != VenueReadFresh {
			allPublishersFresh = false
			continue
		}
		gaps = append(gaps, gap)
		weights = append(weights, v.OIValue)
		scope = append(scope, string(ex))
	}
	if publishers == 0 || !allPublishersFresh || len(gaps) != publishers {
		return
	}
	row.GapDefinite = true
	row.GapScope = strings.Join(scope, ",")
	row.GapPct = FormatSignedPct(boardWeightedMean(gaps, weights, row.OpenInterestDefinite))
}

func boardLongStatus(row *DerivativesCoin, ex Exchange) string {
	if cell := boardVenueRow(row, ex); cell != nil && cell.LongStatus != "" {
		return cell.LongStatus
	}
	return VenueReadMissing
}

func boardWeightedMean(values, weights []float64, useWeight bool) float64 {
	if len(values) == 0 {
		return 0
	}
	var weighted, weight, sum float64
	for i, v := range values {
		sum += v
		if useWeight && i < len(weights) && weights[i] > 0 {
			weighted += v * weights[i]
			weight += weights[i]
		}
	}
	if weight > 0 {
		return weighted / weight
	}
	return sum / float64(len(values))
}

func applyBoardCVD(row *DerivativesCoin, expected []Exchange, byEx map[Exchange]DerivativesVenueInput, now time.Time) {
	samples := make([]VenueSample, 0, len(expected))
	for _, ex := range expected {
		v := byEx[ex]
		if !v.CVDDayOK {
			continue
		}
		samples = append(samples, VenueSample{Exchange: ex, OK: true, At: v.CVDAt})
	}
	cover := CoverVenues(expected, samples, now, VenueTapeMaxAge)
	status := indexVenueReads(cover)
	var sum float64
	for _, ex := range expected {
		v := byEx[ex]
		st := status[ex]
		if cell := boardVenueRow(row, ex); cell != nil {
			cell.CVDStatus = st
			if st == VenueReadFresh {
				cell.CVDDay = FormatSignedQty(v.CVDDay)
			}
		}
		if st == VenueReadFresh {
			sum += v.CVDDay
		}
	}
	if !cover.Complete {
		return
	}
	row.CVDDayDefinite = true
	row.CVDDay = FormatSignedQty(sum)
}

func applyBoardPrint(row *DerivativesCoin, expected []Exchange, byEx map[Exchange]DerivativesVenueInput, now time.Time) {
	samples := make([]VenueSample, 0, len(expected))
	for _, ex := range expected {
		v := byEx[ex]
		if !v.PrintOK || v.PrintCount <= 0 || v.PrintPriorCount <= 0 {
			continue
		}
		samples = append(samples, VenueSample{Exchange: ex, OK: true, At: v.PrintAt})
	}
	cover := CoverVenues(expected, samples, now, VenueTapeMaxAge)
	status := indexVenueReads(cover)
	var notional, prior float64
	var count, priorCount int
	for _, ex := range expected {
		v := byEx[ex]
		st := status[ex]
		if cell := boardVenueRow(row, ex); cell != nil {
			cell.PrintStatus = st
			if st == VenueReadFresh {
				cell.PrintChangePct = FormatSignedPct(boardPrintChange(v.PrintNotional, v.PrintCount, v.PrintPriorNotional, v.PrintPriorCount))
			}
		}
		if st == VenueReadFresh {
			notional += v.PrintNotional
			prior += v.PrintPriorNotional
			count += v.PrintCount
			priorCount += v.PrintPriorCount
		}
	}
	if !cover.Complete || count <= 0 || priorCount <= 0 {
		return
	}
	change := boardPrintChange(notional, count, prior, priorCount)
	row.PrintDefinite = true
	row.PrintChangePct = FormatSignedPct(change)
	row.PrintRead = boardPrintRead(change)
}

func boardPrintChange(notional float64, count int, priorNotional float64, priorCount int) float64 {
	if count <= 0 || priorCount <= 0 || priorNotional <= 0 {
		return 0
	}
	avg := notional / float64(count)
	prior := priorNotional / float64(priorCount)
	if prior <= 0 {
		return 0
	}
	return (avg/prior - 1) * 100
}

func applyBoardOrderCount(row *DerivativesCoin, expected []Exchange, byEx map[Exchange]DerivativesVenueInput, now time.Time) {
	samples := make([]VenueSample, 0, len(expected))
	for _, ex := range expected {
		v := byEx[ex]
		if !v.OrderOK || v.OrderBuy+v.OrderSell <= 0 || v.OrderAt.IsZero() {
			continue
		}
		samples = append(samples, VenueSample{Exchange: ex, OK: true, At: v.OrderAt})
	}
	cover := CoverVenues(expected, samples, now, VenueTapeMaxAge)
	status := indexVenueReads(cover)
	var buy, sell int
	for _, ex := range expected {
		v := byEx[ex]
		st := status[ex]
		if cell := boardVenueRow(row, ex); cell != nil {
			cell.OrderStatus = st
			if st == VenueReadFresh {
				ratio := orderCountRatio(v.OrderBuy, v.OrderSell)
				cell.OrderBuy = v.OrderBuy
				cell.OrderSell = v.OrderSell
				cell.OrderRatio = fmt.Sprintf("%+.2f", ratio)
				cell.OrderLean = orderCountLean(ratio, v.OrderBuy+v.OrderSell)
			}
		}
		if st == VenueReadFresh {
			buy += v.OrderBuy
			sell += v.OrderSell
		}
	}
	if !cover.Complete || buy+sell <= 0 {
		return
	}
	ratio := orderCountRatio(buy, sell)
	row.OrderDefinite = true
	row.OrderBuy = buy
	row.OrderSell = sell
	row.OrderRatio = fmt.Sprintf("%+.2f", ratio)
	row.OrderLean = orderCountLean(ratio, buy+sell)
}

func orderCountRatio(buy, sell int) float64 {
	total := buy + sell
	if total <= 0 {
		return 0
	}
	return float64(buy-sell) / float64(total)
}

// AttachBoardUnusual copies each coin's farthest standout onto the row and
// names that venue. Other venues keep their own standout. A typical coin stays blank.
func AttachBoardUnusual(board *DerivativesBoard, hits []UnusualScanHit) {
	if board == nil || len(hits) == 0 {
		return
	}
	type pick struct {
		hit UnusualScanHit
		ok  bool
	}
	bestByVenue := map[string]pick{}
	for _, hit := range hits {
		symbol := NormalizeLiquidationSymbol(hit.Symbol)
		ex := strings.ToLower(strings.TrimSpace(hit.Exchange))
		if symbol == "" || ex == "" || !unusualStandout(hit.Band) {
			continue
		}
		hit.Symbol = symbol
		hit.Exchange = ex
		key := symbol + "|" + ex
		cur := bestByVenue[key]
		if !cur.ok || hit.Distance > cur.hit.Distance {
			bestByVenue[key] = pick{hit: hit, ok: true}
		}
	}
	for i := range board.Coins {
		row := &board.Coins[i]
		symbol := NormalizeLiquidationSymbol(row.Symbol)
		var best pick
		for j := range row.Venues {
			cell := &row.Venues[j]
			chosen, ok := bestByVenue[symbol+"|"+strings.ToLower(cell.Exchange)]
			if !ok || !chosen.ok {
				continue
			}
			cell.UnusualMetric = chosen.hit.Metric
			cell.UnusualBand = chosen.hit.Band
			cell.UnusualDistance = formatFixed(chosen.hit.Distance, 2)
			if !best.ok || chosen.hit.Distance > best.hit.Distance {
				best = chosen
			}
		}
		if !best.ok {
			continue
		}
		row.UnusualDefinite = true
		row.UnusualMetric = best.hit.Metric
		row.UnusualBand = best.hit.Band
		row.UnusualDistance = formatFixed(best.hit.Distance, 2)
		row.UnusualPercentile = formatFixed(best.hit.Percentile, 1)
		row.UnusualVenue = best.hit.Exchange
	}
}

func boardPrintRead(changePct float64) string {
	ratio := 1 + changePct/100
	switch {
	case ratio >= printSizeJumpRatio:
		return PrintSizeKindJumped
	case ratio > 0 && ratio <= printSizeFadeRatio:
		return PrintSizeKindFaded
	case ratio >= printSizeVolRatio:
		return PrintSizeReadLarger
	case ratio > 0 && ratio <= 1/printSizeVolRatio:
		return PrintSizeReadSmaller
	default:
		return PrintSizeReadStable
	}
}

func applyBoardLiqHeat(row *DerivativesCoin, expected []Exchange, byEx map[Exchange]DerivativesVenueInput, liqCover MarketCoverage) {
	feed := indexVenueReads(liqCover)
	definite := liqCover.Complete && len(expected) > 0
	var hottest Exchange
	var hotGrade string
	var hotRatio float64
	picked := false
	same := true
	var first string
	for _, ex := range expected {
		v := byEx[ex]
		status := feed[ex]
		if status == "" {
			status = VenueReadMissing
		}
		if !v.LiqHeatOK {
			status = VenueReadMissing
			definite = false
		} else if status != VenueReadFresh {
			definite = false
		}
		if cell := boardVenueRow(row, ex); cell != nil {
			cell.LiqHeatStatus = status
			if status == VenueReadFresh && v.LiqHeatOK {
				cell.LiqGrade = v.LiqGrade
			}
		}
		if !v.LiqHeatOK || feed[ex] != VenueReadFresh {
			continue
		}
		if first == "" {
			first = v.LiqGrade
		} else if v.LiqGrade != first {
			same = false
		}
		if !picked || cascadeGradeRank(v.LiqGrade) > cascadeGradeRank(hotGrade) || (v.LiqGrade == hotGrade && v.LiqRatio > hotRatio) {
			picked = true
			hottest = ex
			hotGrade = v.LiqGrade
			hotRatio = v.LiqRatio
		}
	}
	if !definite || !picked {
		return
	}
	row.LiqHeatDefinite = true
	row.LiqHeat = hotGrade
	row.LiqHeatRatio = formatFixed(hotRatio, 2)
	if !same {
		row.LiqHeatVenue = string(hottest)
	}
}

func applyBoardBid(row *DerivativesCoin, expected []Exchange, byEx map[Exchange]DerivativesVenueInput, now time.Time) {
	samples := make([]VenueSample, 0, len(expected))
	for _, ex := range expected {
		v := byEx[ex]
		if !v.BookOK || v.BidNotional+v.AskNotional <= 0 {
			continue
		}
		samples = append(samples, VenueSample{Exchange: ex, OK: true, At: v.BookAt})
	}
	cover := CoverVenues(expected, samples, now, VenueBookMaxAge)
	status := indexVenueReads(cover)
	var bid, ask float64
	for _, ex := range expected {
		v := byEx[ex]
		st := status[ex]
		if cell := boardVenueRow(row, ex); cell != nil {
			cell.BookStatus = st
			if st == VenueReadFresh {
				cell.BidPct = formatFixed(v.BidNotional/(v.BidNotional+v.AskNotional)*100, 2)
			}
		}
		if st == VenueReadFresh {
			bid += v.BidNotional
			ask += v.AskNotional
		}
	}
	if !cover.Complete || bid+ask <= 0 {
		return
	}
	row.BidDefinite = true
	row.BidPct = formatFixed(bid/(bid+ask)*100, 2)
}

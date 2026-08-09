package domain

import "math"

// TradingCost is paper taker fee and slippage for one venue (fractions, not percent).
// Example: FeeRate 0.001 = 0.10% of fill notional.
type TradingCost struct {
	FeeRate      float64
	SlippageRate float64
}

// Paper defaults match public spot taker-style rates (not paid tiers).
const (
	BinanceFeeRate       = 0.001  // 0.10%
	BinanceSlippageRate  = 0.0005 // 0.05%
	CoinbaseFeeRate      = 0.006  // 0.60%
	CoinbaseSlippageRate = 0.0008 // 0.08%
	BybitFeeRate         = 0.001  // 0.10%
	BybitSlippageRate    = 0.0005 // 0.05%
)

// TradingCostFor returns paper fee/slippage for an exchange.
func TradingCostFor(ex Exchange) TradingCost {
	switch ParseExchange(string(ex)) {
	case ExchangeCoinbase:
		return TradingCost{FeeRate: CoinbaseFeeRate, SlippageRate: CoinbaseSlippageRate}
	case ExchangeBybit:
		return TradingCost{FeeRate: BybitFeeRate, SlippageRate: BybitSlippageRate}
	default:
		return TradingCost{FeeRate: BinanceFeeRate, SlippageRate: BinanceSlippageRate}
	}
}

// ZeroTradingCosts is used by unit tests that assert pre-fee cash numbers.
func ZeroTradingCosts(Exchange) TradingCost { return TradingCost{} }

// ApplySlippage moves last against the trader: buys pay more, sells receive less.
func ApplySlippage(last float64, side TradeSide, slipRate float64) float64 {
	if last <= 0 || math.IsNaN(last) || math.IsInf(last, 0) {
		return last
	}
	if slipRate < 0 || math.IsNaN(slipRate) || math.IsInf(slipRate, 0) {
		slipRate = 0
	}
	switch side {
	case TradeSideBuy:
		return last * (1 + slipRate)
	case TradeSideSell:
		out := last * (1 - slipRate)
		if out < 0 {
			return 0
		}
		return out
	default:
		return last
	}
}

// FeeAmount is qty * fillPrice * feeRate (quote currency).
func FeeAmount(qty, fillPrice, feeRate float64) float64 {
	if qty <= 0 || fillPrice <= 0 || feeRate <= 0 {
		return 0
	}
	return qty * fillPrice * feeRate
}

// BuyCashDebit is cash spent on a buy including fee: qty * fill * (1+fee).
func BuyCashDebit(qty, fillPrice, feeRate float64) float64 {
	if qty <= 0 || fillPrice <= 0 {
		return 0
	}
	if feeRate < 0 {
		feeRate = 0
	}
	return qty * fillPrice * (1 + feeRate)
}

// SellCashCredit is cash received on a sell after fee: qty * fill * (1-fee).
func SellCashCredit(qty, fillPrice, feeRate float64) float64 {
	if qty <= 0 || fillPrice <= 0 {
		return 0
	}
	if feeRate < 0 {
		feeRate = 0
	}
	out := qty * fillPrice * (1 - feeRate)
	if out < 0 {
		return 0
	}
	return out
}

// BuyUnitCost is the tax-lot unit cost including the buy fee.
func BuyUnitCost(fillPrice, feeRate float64) float64 {
	if fillPrice <= 0 {
		return 0
	}
	if feeRate < 0 {
		feeRate = 0
	}
	return fillPrice * (1 + feeRate)
}

// NetSellPrice is fill after subtracting the sell fee (used for realized PnL).
func NetSellPrice(fillPrice, feeRate float64) float64 {
	if fillPrice <= 0 {
		return 0
	}
	if feeRate < 0 {
		feeRate = 0
	}
	out := fillPrice * (1 - feeRate)
	if out < 0 {
		return 0
	}
	return out
}

// MarginOpenTradeSide is the spot-like side of opening a margin position.
func MarginOpenTradeSide(side MarginSide) TradeSide {
	if side == MarginShort {
		return TradeSideSell
	}
	return TradeSideBuy
}

// MarginCloseTradeSide is the spot-like side of closing a margin position.
func MarginCloseTradeSide(side MarginSide) TradeSide {
	if side == MarginShort {
		return TradeSideBuy
	}
	return TradeSideSell
}

// EffectiveMarginEntry folds open fee into the stored entry (same idea as tax-lot cost).
func EffectiveMarginEntry(side MarginSide, fillPrice, feeRate float64) float64 {
	if side == MarginShort {
		return NetSellPrice(fillPrice, feeRate)
	}
	return BuyUnitCost(fillPrice, feeRate)
}

// EffectiveMarginExit folds close fee into the exit price for realized PnL.
func EffectiveMarginExit(side MarginSide, fillPrice, feeRate float64) float64 {
	if side == MarginShort {
		return BuyUnitCost(fillPrice, feeRate)
	}
	return NetSellPrice(fillPrice, feeRate)
}

// RawFillFromEffectiveEntry inverts EffectiveMarginEntry (zero fee → identity).
func RawFillFromEffectiveEntry(side MarginSide, effective, feeRate float64) float64 {
	if effective <= 0 || math.IsNaN(effective) || math.IsInf(effective, 0) {
		return effective
	}
	if feeRate <= 0 || math.IsNaN(feeRate) || math.IsInf(feeRate, 0) {
		return effective
	}
	if side == MarginShort {
		den := 1 - feeRate
		if den <= 0 {
			return effective
		}
		return effective / den
	}
	return effective / (1 + feeRate)
}

// PaperTradingCostView is the public per-exchange paper fee/slippage snapshot.
type PaperTradingCostView struct {
	Exchange     Exchange
	FeeRate      float64
	SlippageRate float64
	FeePct       float64
	SlippagePct  float64
}

// PaperTradingCostViewFor is the published paper cost card for one venue.
func PaperTradingCostViewFor(ex Exchange) PaperTradingCostView {
	parsed := ParseExchange(string(ex))
	c := TradingCostFor(parsed)
	return PaperTradingCostView{
		Exchange:     parsed,
		FeeRate:      c.FeeRate,
		SlippageRate: c.SlippageRate,
		FeePct:       c.FeeRate * 100,
		SlippagePct:  c.SlippageRate * 100,
	}
}

// AllPaperTradingCosts returns binance, coinbase, and bybit paper rates.
func AllPaperTradingCosts() []PaperTradingCostView {
	return []PaperTradingCostView{
		PaperTradingCostViewFor(ExchangeBinance),
		PaperTradingCostViewFor(ExchangeCoinbase),
		PaperTradingCostViewFor(ExchangeBybit),
	}
}

// PaperTradingCostsNote is returned with the public rates endpoint.
const PaperTradingCostsNote = "Paper taker fee and adverse slippage on every fill (market, pending, recurring, margin). Buy cash and tax-lot cost include the fee; sell proceeds and realized PnL are after the fee. Pending buy reservations cover worst-case slip + fee. Simulated only — not real money."

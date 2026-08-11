package bybit

import (
	"testing"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

func TestParseAllLiquidation_LongFromBuy(t *testing.T) {
	raw := []byte(`{"topic":"allLiquidation.BTCUSDT","type":"snapshot","ts":1,"data":[{"T":1700000000000,"s":"BTCUSDT","S":"Buy","v":"0.5","p":"64000"}]}`)
	got := ParseAllLiquidation(raw)
	if len(got) != 1 || got[0].Side != domain.LiquidationSideLong || got[0].Exchange != domain.ExchangeBybit {
		t.Fatalf("%+v", got)
	}
	if got[0].Notional < 31999 || got[0].Notional > 32001 {
		t.Fatalf("notional %v", got[0].Notional)
	}
}

func TestParseAllLiquidation_IgnorePong(t *testing.T) {
	if got := ParseAllLiquidation([]byte(`{"op":"pong"}`)); len(got) != 0 {
		t.Fatalf("%+v", got)
	}
}

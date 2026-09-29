package market

import "testing"

func TestFlowSamplePlan_SeedsFastLiquidSlow(t *testing.T) {
	svc := &Service{}
	svc.SetFlowLiquid([]string{"BTCUSDT", "PEPEUSDT"})
	var solSlow, btcSlow bool
	var sawBTC, sawSOL bool
	for _, item := range svc.FlowSamplePlan() {
		switch item.Symbol {
		case "BTCUSDT":
			sawBTC = true
			btcSlow = item.Slow
		case "PEPEUSDT":
			sawSOL = true
			solSlow = item.Slow
		}
	}
	if !sawBTC || !sawSOL {
		t.Fatalf("btc=%v sol=%v", sawBTC, sawSOL)
	}
	if btcSlow || !solSlow {
		t.Fatalf("btc slow=%v sol slow=%v", btcSlow, solSlow)
	}
}

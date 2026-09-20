package domain

import (
	"testing"
	"time"
)

func TestBlendMarketPressure_UserDisagreement(t *testing.T) {
	got := BlendMarketPressure(MarketPressureInputs{
		Book: 0.40, BookOK: true,
		Flow: -0.35, FlowOK: true,
		AbsKind: AbsorptionKindBid, AbsScore: 70, AbsOK: true, AbsGrade: AbsorptionGradeStrong,
	})
	if got.Value <= 0 {
		t.Fatalf("absorption should pull net buy, got %v", got.Value)
	}
	if got.Value < 0.25 || got.Value > 0.55 {
		t.Fatalf("value %v", got.Value)
	}
	if got.Lean != MarketPressureLeanBuy {
		t.Fatalf("lean %s", got.Lean)
	}
	if got.Agreement != MarketPressureAgreeOpposite {
		t.Fatalf("agreement %s", got.Agreement)
	}
	if got.Headline == "" || got.ValueText[0] != '+' {
		t.Fatalf("copy %+v", got)
	}
	var abs MarketPressureComponent
	for _, c := range got.Components {
		if c.ID == MarketPressureCompAbs {
			abs = c
		}
	}
	if !abs.Used || abs.Weight < 0.45 {
		t.Fatalf("abs weight %+v", abs)
	}
}

func TestBlendMarketPressure_AllBuy(t *testing.T) {
	got := BlendMarketPressure(MarketPressureInputs{
		Book: 0.50, BookOK: true,
		Flow: 0.40, FlowOK: true,
		AbsKind: AbsorptionKindBid, AbsScore: 60, AbsOK: true,
	})
	if got.Lean != MarketPressureLeanBuy || got.Agreement != MarketPressureAgreeSame {
		t.Fatalf("%+v", got)
	}
	if got.Value < 0.4 {
		t.Fatalf("value %v", got.Value)
	}
}

func TestBlendMarketPressure_BookOnly(t *testing.T) {
	got := BlendMarketPressure(MarketPressureInputs{Book: -0.50, BookOK: true})
	if got.Lean != MarketPressureLeanSell || got.Value > -0.4 {
		t.Fatalf("%+v", got)
	}
	if got.Agreement != MarketPressureAgreeMixed {
		t.Fatalf("one signal agreement %s", got.Agreement)
	}
}

func TestBlendMarketPressure_Empty(t *testing.T) {
	got := BlendMarketPressure(MarketPressureInputs{})
	if got.Value != 0 || got.Headline == "" {
		t.Fatalf("%+v", got)
	}
}

func TestBuildMarketPressure_HourChange(t *testing.T) {
	cur := MarketPressureInputs{Book: 0.40, BookOK: true, Flow: 0.20, FlowOK: true}
	past := MarketPressureInputs{Book: -0.30, BookOK: true, Flow: -0.20, FlowOK: true}
	got := BuildMarketPressure("btcusdt", "all", parseTestTime(), cur, past, parseTestTime().Add(-time.Hour), true)
	if got.Symbol != "BTCUSDT" || got.Current.Lean != MarketPressureLeanBuy {
		t.Fatalf("%+v", got)
	}
	if len(got.Windows) != 1 || got.Windows[0].Kind != MarketPressureFlipped {
		t.Fatalf("windows %+v", got.Windows)
	}
	if got.Change1h == "" || got.Change1h[0] != '+' {
		t.Fatalf("change %s", got.Change1h)
	}
}

func TestCVDFlowShare(t *testing.T) {
	v, ok := CVDFlowShare(CVDWindowStat{BuyNotional: 80, SellNotional: 20, CVDChange: 60})
	if !ok || v < 0.59 || v > 0.61 {
		t.Fatalf("%v %v", v, ok)
	}
}

func parseTestTime() time.Time {
	return time.Date(2026, 9, 20, 16, 0, 0, 0, time.UTC)
}

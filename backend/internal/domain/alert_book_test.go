package domain

import "testing"

func TestNormalizeAlertKind(t *testing.T) {
	k, ok := NormalizeAlertKind("")
	if !ok || k != AlertKindPrice {
		t.Fatalf("%v %v", k, ok)
	}
	k, ok = NormalizeAlertKind("imbalance")
	if !ok || k != AlertKindImbalance {
		t.Fatalf("%v %v", k, ok)
	}
	if _, ok := NormalizeAlertKind("rsi"); ok {
		t.Fatal("want invalid")
	}
}

func TestValidateAlertSpec(t *testing.T) {
	if err := ValidateAlertSpec(AlertKindPrice, "above", 100, 0); err != nil {
		t.Fatal(err)
	}
	if err := ValidateAlertSpec(AlertKindImbalance, "below", 0.2, 2); err != nil {
		t.Fatal(err)
	}
	if err := ValidateAlertSpec(AlertKindWall, "bid", 0, 2); err != nil {
		t.Fatal(err)
	}
	if err := ValidateAlertSpec(AlertKindImbalance, "above", 0.01, 2); err == nil {
		t.Fatal("tiny threshold")
	}
	if err := ValidateAlertSpec(AlertKindWall, "above", 0, 2); err == nil {
		t.Fatal("wall cond")
	}
}

func TestBookAlertObservation_Imbalance(t *testing.T) {
	a := PriceAlert{Kind: AlertKindImbalance, Condition: AlertAbove, TargetPrice: 0.2}
	met, m := BookAlertObservation(a, OrderBookAnalysis{Imbalance: 0.19})
	if met {
		t.Fatalf("under %v", m)
	}
	met, m = BookAlertObservation(a, OrderBookAnalysis{Imbalance: 0.25})
	if !met || m != 0.25 {
		t.Fatalf("%v %v", met, m)
	}
	a.Condition = AlertBelow
	met, _ = BookAlertObservation(a, OrderBookAnalysis{Imbalance: -0.21})
	if !met {
		t.Fatal("sell imbalance")
	}
}

func TestBookAlertObservation_Wall(t *testing.T) {
	an := OrderBookAnalysis{Walls: []OrderBookWall{
		{Side: "bid", Share: 0.4},
		{Side: "ask", Share: 0.12},
	}}
	a := PriceAlert{Kind: AlertKindWall, Condition: AlertWallBid, TargetPrice: 0.3}
	met, m := BookAlertObservation(a, an)
	if !met || m != 0.4 {
		t.Fatalf("%v %v", met, m)
	}
	a.TargetPrice = 0.5
	if met, _ = BookAlertObservation(a, an); met {
		t.Fatal("share too small")
	}
	a = PriceAlert{Kind: AlertKindWall, Condition: AlertWallAsk, TargetPrice: 0}
	if met, _ = BookAlertObservation(a, an); !met {
		t.Fatal("any ask wall")
	}
	a.Condition = AlertWallAny
	if met, _ = BookAlertObservation(a, an); !met {
		t.Fatal("any")
	}
	if met, _ = BookAlertObservation(a, OrderBookAnalysis{}); met {
		t.Fatal("empty")
	}
}

func TestEvaluateBookAlert_RepeatingNoRetrigger(t *testing.T) {
	a := PriceAlert{
		Kind: AlertKindImbalance, Condition: AlertAbove, TargetPrice: 0.2,
		Mode: AlertModeRepeating, Status: AlertStatusActive, Armed: true,
	}
	hot := OrderBookAnalysis{Imbalance: 0.3}
	ev, metric := EvaluateBookAlert(a, hot)
	if !ev.Fire || ev.NewArmed || metric != 0.3 {
		t.Fatalf("%+v %v", ev, metric)
	}
	a.Armed = false
	ev, _ = EvaluateBookAlert(a, hot)
	if ev.Fire {
		t.Fatal("still hot must not re-fire")
	}
	ev, _ = EvaluateBookAlert(a, OrderBookAnalysis{Imbalance: 0.01})
	if ev.Fire || !ev.NewArmed {
		t.Fatalf("re-arm %+v", ev)
	}
	a.Armed = true
	ev, _ = EvaluateBookAlert(a, hot)
	if !ev.Fire {
		t.Fatal("second appearance")
	}
}

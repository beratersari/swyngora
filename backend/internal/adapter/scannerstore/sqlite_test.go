package scannerstore

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

func TestSQLite_RulesAndDedupeResults(t *testing.T) {
	path := filepath.Join(t.TempDir(), "scan.db")
	s, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	now := time.Now().UTC()
	rule := domain.ScannerRule{
		ID: "r1", ClientID: "c1", Type: domain.ScannerRuleRSI, Interval: "1h", Enabled: true,
		RSIPeriod: 14, RSICondition: domain.AlertBelow, RSIThreshold: 30,
		CreatedAt: now, UpdatedAt: now,
	}
	if _, err := s.CreateRule(ctx, rule); err != nil {
		t.Fatal(err)
	}
	got, err := s.GetRule(ctx, "c1", "r1")
	if err != nil || got.RSIThreshold != 30 {
		t.Fatalf("%+v %v", got, err)
	}
	en, err := s.ListEnabledRules(ctx)
	if err != nil || len(en) != 1 {
		t.Fatalf("%+v %v", en, err)
	}
	res := domain.ScannerResult{
		ID: "x1", ClientID: "c1", RuleID: "r1", Exchange: domain.ExchangeBinance, Symbol: "BTCUSDT",
		RuleType: domain.ScannerRuleRSI, Interval: "1h", MarketDataKey: "2024-01-01T00:00:00Z",
		MatchedAt: now, Summary: "test", Metrics: map[string]float64{"rsi": 25},
	}
	ins, ok, err := s.InsertResult(ctx, res)
	if err != nil || !ok || ins == nil {
		t.Fatalf("insert %v %v %v", ins, ok, err)
	}
	// duplicate same market data
	_, ok, err = s.InsertResult(ctx, res)
	if err != nil || ok {
		t.Fatalf("dup want false: ok=%v err=%v", ok, err)
	}
	// different bar ok
	res.ID = "x2"
	res.MarketDataKey = "2024-01-01T01:00:00Z"
	_, ok, err = s.InsertResult(ctx, res)
	if err != nil || !ok {
		t.Fatalf("second bar: ok=%v err=%v", ok, err)
	}
	list, err := s.ListResults(ctx, "c1", 10, 0)
	if err != nil || len(list) != 2 {
		t.Fatalf("%+v %v", list, err)
	}
	_ = s.Close()
	s2, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer s2.Close()
	list, err = s2.ListResults(ctx, "c1", 10, 0)
	if err != nil || len(list) != 2 {
		t.Fatalf("persist %+v %v", list, err)
	}
}

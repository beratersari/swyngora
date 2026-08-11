package domain

import (
	"testing"
	"time"
)

func TestParseFuturesMetric(t *testing.T) {
	got, err := ParseFuturesMetric("oi")
	if err != nil || got != FuturesMetricOpenInterest {
		t.Fatalf("%s %v", got, err)
	}
	if _, err := ParseFuturesMetric("nope"); err == nil {
		t.Fatal("expected error")
	}
}

func TestTruncateToBucket(t *testing.T) {
	in := time.Date(2026, 8, 11, 16, 17, 42, 0, time.UTC)
	got := TruncateToBucket(in, 5*time.Minute)
	want := time.Date(2026, 8, 11, 16, 15, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Fatalf("got %s want %s", got, want)
	}
}

func TestNormalizeFuturesSymbols(t *testing.T) {
	got := NormalizeFuturesSymbols([]string{"btc-usd", "BTCUSDT", " ethusdt ", ""})
	if len(got) != 2 || got[0] != "BTCUSDT" || got[1] != "ETHUSDT" {
		t.Fatalf("%v", got)
	}
}

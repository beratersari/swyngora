package domain

import (
	"testing"
	"time"
)

func TestPrintTape_DedupAndCover(t *testing.T) {
	now := time.Date(2026, 10, 3, 12, 0, 0, 0, time.UTC)
	tape := NewPrintTape()
	tape.now = func() time.Time { return now }
	p := TakerPrint{
		Exchange: ExchangeBinance, Symbol: "BTCUSDT", Side: TakerSideBuy,
		Price: 100, Quantity: 2, Notional: 200, Time: now.Add(-time.Minute),
	}
	if !tape.Record(p) {
		t.Fatal("first record")
	}
	if tape.Record(p) {
		t.Fatal("duplicate should be ignored")
	}
	got := tape.Prints(ExchangeBinance, "BTCUSDT")
	if len(got) != 1 || got[0].Notional != 200 {
		t.Fatalf("%+v", got)
	}
	if !tape.Covers(ExchangeBinance, "BTCUSDT", now.Add(-time.Minute)) {
		t.Fatal("expected cover")
	}
	if tape.Covers(ExchangeBinance, "BTCUSDT", now.Add(-30*time.Minute)) {
		t.Fatal("tape does not reach 30m")
	}
}

func TestPrintTape_DropsStale(t *testing.T) {
	now := time.Date(2026, 10, 3, 12, 0, 0, 0, time.UTC)
	tape := NewPrintTape()
	tape.now = func() time.Time { return now }
	old := TakerPrint{
		Exchange: ExchangeOKX, Symbol: "ETHUSDT", Side: TakerSideSell,
		Price: 10, Quantity: 1, Notional: 10, Time: now.Add(-2 * time.Hour),
	}
	if !tape.Record(old) {
		t.Fatal("within book retention should count as new")
	}
	if len(tape.Prints(ExchangeOKX, "ETHUSDT")) != 0 {
		t.Fatal("older than the print retain should not stay on the tape")
	}
}

func TestRecordMeasuredPrint_SkipsDuplicateOnBook(t *testing.T) {
	now := time.Now().UTC()
	book := NewTakerBook()
	tape := NewPrintTape()
	p := TakerPrint{
		Exchange: ExchangeBybit, Symbol: "BTCUSDT", Side: TakerSideBuy,
		Price: 50, Quantity: 2, Notional: 100, Time: now.Add(-time.Second),
	}
	RecordMeasuredPrint(book, tape, p)
	RecordMeasuredPrint(book, tape, p)
	got := book.Buckets(ExchangeBybit, "BTCUSDT")
	if len(got) != 1 || got[0].BuyNotional != 100 {
		t.Fatalf("%+v", got)
	}
}

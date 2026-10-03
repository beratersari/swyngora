package okx

import (
	"strconv"
	"testing"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

func TestTradeHub_RecordsContractNotional(t *testing.T) {
	book := domain.NewTakerBook()
	tape := domain.NewPrintTape()
	h := NewTradeHub(TradeHubOptions{
		Book: book,
		Tape: tape,
		ContractValue: func(inst string) float64 {
			if inst == "BTC-USDT-SWAP" {
				return 0.01
			}
			return 0
		},
	})
	ts := time.Now().UTC().Add(-time.Second).UnixMilli()
	msg := []byte(`{"arg":{"channel":"trades","instId":"BTC-USDT-SWAP"},"data":[{"instId":"BTC-USDT-SWAP","side":"sell","px":"64000","sz":"2","ts":"` + strconv.FormatInt(ts, 10) + `"}]}`)
	for _, p := range h.parseTrades(msg) {
		domain.RecordMeasuredPrint(book, tape, p)
	}
	for _, p := range h.parseTrades(msg) {
		domain.RecordMeasuredPrint(book, tape, p)
	}
	got := book.Buckets(domain.ExchangeOKX, "BTCUSDT")
	if len(got) != 1 || got[0].SellNotional != 1280 { // 2 contracts * 0.01 * 64000
		t.Fatalf("%+v", got)
	}
	if len(tape.Prints(domain.ExchangeOKX, "BTCUSDT")) != 1 {
		t.Fatal("duplicate print stored")
	}
}

func TestTradePrint_SkipsUnknownContract(t *testing.T) {
	if _, ok := tradePrint("BTC-USDT-SWAP", "buy", "100", "1", "1710000000000", 0); ok {
		t.Fatal("missing ctVal should not invent a size")
	}
}

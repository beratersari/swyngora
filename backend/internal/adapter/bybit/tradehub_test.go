package bybit

import (
	"fmt"
	"testing"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

func TestTradeHubHandle_RecordsBuy(t *testing.T) {
	book := domain.NewTakerBook()
	h := NewTradeHub(TradeHubOptions{Book: book})
	ts := time.Now().UTC().UnixMilli()
	h.handle([]byte(fmt.Sprintf(
		`{"topic":"publicTrade.BTCUSDT","data":[{"T":%d,"s":"BTCUSDT","S":"Buy","v":"0.5","p":"64000"}]}`,
		ts,
	)))
	got := book.Snapshot(domain.ExchangeBybit, "BTCUSDT")
	if len(got.Windows) == 0 || got.Windows[0].BuyNotional < 31999 {
		t.Fatalf("%+v", got)
	}
}

func TestTradeHubHandle_IgnorePong(t *testing.T) {
	book := domain.NewTakerBook()
	h := NewTradeHub(TradeHubOptions{Book: book})
	h.handle([]byte(`{"op":"pong"}`))
	got := book.Snapshot(domain.ExchangeBybit, "BTCUSDT")
	if got.Windows != nil {
		for _, w := range got.Windows {
			if w.BuyNotional != 0 || w.SellNotional != 0 {
				t.Fatalf("%+v", got)
			}
		}
	}
}

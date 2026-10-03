package binance

import (
	"strconv"
	"testing"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

func TestParseAggTrade_MakerIsTakerSell(t *testing.T) {
	ts := time.Now().UTC().Add(-time.Second).UnixMilli()
	msg := []byte(`{"stream":"btcusdt@aggTrade","data":{"e":"aggTrade","s":"BTCUSDT","p":"64000","q":"0.5","T":` + strconv.FormatInt(ts, 10) + `,"m":true}}`)
	p, ok := parseAggTrade(msg)
	if !ok || p.Side != domain.TakerSideSell || p.Notional != 32000 {
		t.Fatalf("%v %+v", ok, p)
	}
	book := domain.NewTakerBook()
	tape := domain.NewPrintTape()
	domain.RecordMeasuredPrint(book, tape, p)
	domain.RecordMeasuredPrint(book, tape, p)
	got := book.Buckets(domain.ExchangeBinance, "BTCUSDT")
	if len(got) != 1 || got[0].SellNotional != 32000 || got[0].BuyNotional != 0 {
		t.Fatalf("%+v", got)
	}
}

func TestParseAggTrade_TakerBuy(t *testing.T) {
	ts := time.Now().UTC().Add(-time.Second).UnixMilli()
	p, ok := parseAggTrade([]byte(`{"e":"aggTrade","s":"ETHUSDT","p":"2000","q":"1.5","T":` + strconv.FormatInt(ts, 10) + `,"m":false}`))
	if !ok || p.Side != domain.TakerSideBuy || p.Notional != 3000 {
		t.Fatalf("%v %+v", ok, p)
	}
}

func TestParseAggTrade_IgnoresSubscribeAck(t *testing.T) {
	if _, ok := parseAggTrade([]byte(`{"result":null,"id":1}`)); ok {
		t.Fatal("ack is not a trade")
	}
}

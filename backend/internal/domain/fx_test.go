package domain

import (
	"math"
	"testing"
)

func TestConvertFx_TRYToUSDAndBack(t *testing.T) {
	rates := map[string]float64{FxTRY: 40, FxEUR: 0.9}
	got, err := ConvertFx(400, FxTRY, FxBaseUSD, rates)
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(got-10) > 1e-9 {
		t.Fatalf("400 TRY -> USD got %v want 10", got)
	}
	back, err := ConvertFx(got, FxBaseUSD, FxTRY, rates)
	if err != nil || math.Abs(back-400) > 1e-9 {
		t.Fatalf("round trip %v err=%v", back, err)
	}
}

func TestConvertFx_USDTAliasesUSD(t *testing.T) {
	rates := map[string]float64{FxTRY: 34}
	got, err := ConvertFx(1, FxUSDT, FxTRY, rates)
	if err != nil || math.Abs(got-34) > 1e-9 {
		t.Fatalf("1 USDT~USD -> TRY got %v err=%v", got, err)
	}
}

func TestQuoteForVenue(t *testing.T) {
	if QuoteForVenue(ExchangeBist) != FxTRY {
		t.Fatal("bist")
	}
	if QuoteForVenue(ExchangeNasdaq) != FxBaseUSD {
		t.Fatal("nasdaq")
	}
	if QuoteForVenue(ExchangeBinance) != FxUSDT {
		t.Fatal("binance")
	}
	if QuoteForMarketCap(ExchangeBist) != FxTRY || QuoteForMarketCap(ExchangeBinance) != FxBaseUSD {
		t.Fatal("mcap quote")
	}
}

func TestDisplayFxMeta(t *testing.T) {
	t.Parallel()
	vq := DisplayVenueQuotes()
	if vq["bist"] != FxTRY || vq["binance"] != FxUSDT {
		t.Fatalf("venue quotes=%v", vq)
	}
	mq := DisplayMarketCapQuotes()
	if mq["bist"] != FxTRY || mq["nasdaq"] != FxBaseUSD {
		t.Fatalf("mcap quotes=%v", mq)
	}
	if DisplayFxAliases()["USDT"] != FxBaseUSD {
		t.Fatal("alias")
	}
}

package domain

import "testing"

func TestCMCLogoURL(t *testing.T) {
	if CMCLogoURL(1) != "https://s2.coinmarketcap.com/static/img/coins/64x64/1.png" {
		t.Fatalf("%s", CMCLogoURL(1))
	}
	if CMCLogoURL(0) != "" {
		t.Fatal("zero id")
	}
}

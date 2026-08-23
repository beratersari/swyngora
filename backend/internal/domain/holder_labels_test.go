package domain

import "testing"

func TestHolderLabel(t *testing.T) {
	if got := HolderLabel("34xp4vRoCGJym3xR7yCVPFHoCNxv4Twseo"); got != "Binance" {
		t.Fatalf("binance=%q", got)
	}
	if got := HolderLabel("bc1qgdjqv0av3q56jvd82tkdjpy7gdp9ut8tlqmgrpmv24sq90ecnvqqjwvw97"); got != "Bitfinex" {
		t.Fatalf("bitfinex=%q", got)
	}
	if got := HolderLabel("BC1QGDJQV0AV3Q56JVD82TKDJPY7GDP9UT8TLQMGRPMV24SQ90ECNVQQJWVW97"); got != "Bitfinex" {
		t.Fatalf("bech32 case=%q", got)
	}
	if got := HolderLabel("1A1zP1eP5QGefi2DMPTfTL5SLmv7DivfNa"); got != "Satoshi (genesis)" {
		t.Fatalf("genesis=%q", got)
	}
	if got := HolderLabel("1a1zp1ep5qgefi2dmptftl5slmv7divfna"); got != "" {
		t.Fatalf("base58 must stay case-sensitive: %q", got)
	}
	if got := HolderLabel("0x75e89d5979E4f6Fba9F97c104c2F0AFB3F1dcB88"); got != "MEXC" {
		t.Fatalf("mexc=%q", got)
	}
	if got := HolderLabel("bc1qa5wkgaew2dkv56kfvj49j0av5nml45x9ek9hz6"); got != "Silk Road FBI" {
		t.Fatalf("silkroad=%q", got)
	}
	if got := HolderLabel("bc1qjasf9z3h7w3jspkhtgatgpyvvzgpa2wwd2lr0eh5tx44reyn2k7sfc27a4"); got != "Tether" {
		t.Fatalf("tether=%q", got)
	}
	if got := HolderLabel("1Ay8vMC7R1UbyCCZRVULMV7iQpHSAbguJP"); got != "Upbit" {
		t.Fatalf("upbit=%q", got)
	}
	if got := HolderLabel("3M219KR5vEneNb47ewrPfWyb5jQ2DjxRP6"); got != "Binance" {
		t.Fatalf("binance2=%q", got)
	}
	if got := HolderLabel("bc1q7ydrtdn8z62xhslqyqtyt38mm4e2c4h3mxjkug"); got != "UK government" {
		t.Fatalf("uk=%q", got)
	}
	if got := HolderLabel("162bzZT2hJfv5Gm3ZmWfWfHJjCtMD6rHhw"); got != "Gate.io" {
		t.Fatalf("gate=%q", got)
	}
	if got := HolderLabel("0xf89d7b9c864f589bbF53a82105107622B35EaA40"); got != "Bybit" {
		t.Fatalf("bybit=%q", got)
	}
	if got := HolderLabel("unknown-wallet-xyz"); got != "" {
		t.Fatalf("unknown=%q", got)
	}
}

func TestAnnotateHolderLabels(t *testing.T) {
	AnnotateHolderLabels(nil)
	in := &AssetHolders{
		TopHolders: []AssetHolder{
			{Address: "34xp4vRoCGJym3xR7yCVPFHoCNxv4Twseo"},
			{Address: "mystery", Label: "Keep me"},
			{Address: "1FeexV6bAHb8ybZjqQMjJrcCrHGW9sb6uF"},
		},
	}
	AnnotateHolderLabels(in)
	if in.TopHolders[0].Label != "Binance" {
		t.Fatalf("row0=%q", in.TopHolders[0].Label)
	}
	if in.TopHolders[1].Label != "Keep me" {
		t.Fatalf("preset overwritten: %q", in.TopHolders[1].Label)
	}
	if in.TopHolders[2].Label != "Mt. Gox hack" {
		t.Fatalf("row2=%q", in.TopHolders[2].Label)
	}
}

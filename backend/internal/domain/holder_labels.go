package domain

import "strings"

// holderLabels maps a normalized address to a public attribution.
// Sources: BitInfoCharts wallet tags, ChainQuery rich-list (non-speculative),
// Etherscan nametags, well-known historical addresses. Not identity proof,
// not clustering (no Patoshi / suspected-Satoshi miner set).
// Speculative single-source tags (unconfirmed Kraken / Deribit / Coincheck) are omitted.
var holderLabels = map[string]string{
	// --- Bitcoin: exchanges ---
	"34xp4vRoCGJym3xR7yCVPFHoCNxv4Twseo":                             "Binance",
	"3M219KR5vEneNb47ewrPfWyb5jQ2DjxRP6":                             "Binance",
	"3M219KT5vCn87s7feN7E2qeEPnTWUDmCnK":                             "Binance",
	"3LYJfcfHPXYJreMsASk2jkn69LWEYKzexb":                             "Binance",
	"1P5ZEDWTKTFGxQjZphgWPQUpe554WKDfHQ":                             "Binance",
	"1NDyJtNTjmwk5xPNhjgAMu4HDHigtobu1s":                             "Binance",
	"3LQUu4v9z6KNch71j7kbj8GPeAGUo1FW6a":                             "Binance",
	"3FHNBLobJnbCTFTVakh5TXmEneyf5PT61B":                             "Binance",
	"34HpHYiyQwg69gFmCq2BGHjF1DZnZnBeBP":                             "Binance",
	"1PJiGp2yDLvUgqeBsuZVCBADArNsk6XEiw":                             "Binance",
	"bc1qm34lsc65zpw79lxes69zkqmk6ee3ewf0j77s3h":                     "Binance",
	"bc1qx9t2l3pyny2spqpqlye8svce70nppwtaxwdrp4":                     "Binance Pool",
	"1Q8QR5k32hexiMQnRgkJ6fmmjn5fMWhdv9":                             "Binance Pool",
	"bc1qgdjqv0av3q56jvd82tkdjpy7gdp9ut8tlqmgrpmv24sq90ecnvqqjwvw97": "Bitfinex",
	"3JZq4atUahhuA9rLhXLMhhTo133J9rF97j":                             "Bitfinex",
	"bc1ql49ydapnjafl5t2cp9zqpjwe6pdgmxy98859v2":                     "Robinhood",
	"1Ay8vMC7R1UbyCCZRVULMV7iQpHSAbguJP":                             "Upbit",
	"3MgEAFWu1HKSnZ5ZsC8qf61ZW18xrP5pgd":                             "OKX",
	"3FM9vDYsN2iuMPKWjAcqgyahdwdrUxhbJ3":                             "OKX",
	"1CY7fykRLWXeSbKB885Kr4KjQxmDdvW923":                             "OKX",
	"bc1qa2eu6p5rl9255e3xz7fcgm6snn4wl5kdfh7zpt05qp5fad9dmsys0qjg0e": "Bybit",
	"bc1qr4dl5wa7kl8yu792dceg9z5knl2gkn220lk7a9":                     "Crypto.com",
	"162bzZT2hJfv5Gm3ZmWfWfHJjCtMD6rHhw":                             "Gate.io",
	"bc1qx2x5cqhymfcnjtg902ky6u5t5htmt7fvqztdsm028hkrvxcl4t2sjtpd9l": "Bitbank",
	"bc1qjasf9z3h7w3jspkhtgatgpyvvzgpa2wwd2lr0eh5tx44reyn2k7sfc27a4": "Tether",

	// --- Bitcoin: seizures, hacks, history ---
	"1FeexV6bAHb8ybZjqQMjJrcCrHGW9sb6uF":         "Mt. Gox hack",
	"bc1qazcm763858nkj2dj986etajv6wquslv8uxwczt": "Bitfinex hack recovery",
	"bc1qkmk4v2xn29yge68fq6zh7gvfdqrvpq3v3p3y0s": "Bitfinex hack recovery",
	"bc1qa5wkgaew2dkv56kfvj49j0av5nml45x9ek9hz6": "Silk Road FBI",
	"1F1tAaz5x1HUXrCNLbtMDqcw6o5GNn4xqX":         "Silk Road",
	"bc1q7ydrtdn8z62xhslqyqtyt38mm4e2c4h3mxjkug": "UK government",
	"bc1q4vxn43l44h30nkluqfxd9eckf45vr2awz38lwa": "UK government",
	"12ib7dApVFvg82TXKycWBNpN8kFyiAN1dr":         "PlusToken",
	"1A1zP1eP5QGefi2DMPTfTL5SLmv7DivfNa":         "Satoshi (genesis)",
	"12cbQLTFMXRnSzktFkuoG3eHoMeFtpTu3S":         "Hal Finney",
	"17SkEw2md5avVNyYgj6RiXuQKNwkXaxFyQ":         "Bitcoin Pizza",

	// --- Ethereum / EVM: exchanges (Etherscan nametags; same 0x form on BSC) ---
	"0x28c6c06298d514db089934071355e5743bf21d60": "Binance",
	"0x21a31ee1afc51d94c2efccaa2092ad1028285549": "Binance",
	"0xdfd5293d8e347dfe59e90efd55b2956a1343963d": "Binance",
	"0xf977814e90da44bfa03b6295a0616a897441acec": "Binance",
	"0x564286362092d8e7936f0549571a803b203aaced": "Binance",
	"0x0681d8db095565fe8a346fa0277bffde9c0edbbf": "Binance",
	"0x56eddb7aa87536c09ccc2793473599fd21a8b17f": "Binance",
	"0x9696f59e4d72e237be84ffd425dcad154bf96976": "Binance",
	"0x4e9ce36e442e55ecd9025b9a6e0d88485d628a67": "Binance",
	"0xbe0eb53f46cd790cd13851d5eff43d12404d33e8": "Binance",
	"0x3f5ce5fbfe3e9af3971dd833d26ba9b5c936f0be": "Binance",
	"0x47ac0fb4f2d84898e4d9e7b4dab3c24507a6d503": "Binance",
	"0x5a52e96bacdabb82fd05763e25335261b270efcb": "Binance",
	"0x4976a4a02f38326660d17bf34b431dc6e2eb2327": "Binance",
	"0x8894e0a0c962cb723c1976a4421c95949be2d4e3": "Binance",
	"0x742d35cc6634c0532925a3b844bc454e4438f44e": "Bitfinex",
	"0x876eabf441b2ee5b5b0554fd502a8e0600950cfa": "Bitfinex",
	"0x75e89d5979e4f6fba9f97c104c2f0afb3f1dcb88": "MEXC",
	"0x3cc936b795a188f0e246cbb2d74c5bd190aecf18": "MEXC",
	"0xa9d1e08c7793af67e9d92fe308d5697fb81d3e43": "Coinbase",
	"0x71660c4005ba85c37ccec55d0c4493e66fe775d3": "Coinbase",
	"0x503828976d22510aad0201ac7ec88293211d23da": "Coinbase",
	"0xddfabcdc4d8ffc6d5beaf154f18b778f892a0740": "Coinbase",
	"0xa090e606e30bd747d4e6245a1517ebe430f0057e": "Coinbase",
	"0xb5d85cbf7cb3ee0d56b3bb207d5fc4b82f43f511": "Coinbase",
	"0xeb2629a2734e272bcc07bda959863f316f4bd4cf": "Coinbase",
	"0x2910543af39aba0cd09dbb2d50200b3e800a63d2": "Kraken",
	"0x267be1c1d684f78cb4f6a176c4911b741e4ffdc0": "Kraken",
	"0xae2d4617c862309a3d75a0ffb358c7a5009c673f": "Kraken",
	"0x0a869d79a7052c7f1b55a8ebabbea3420f0d1e13": "Kraken",
	"0xe853c56864a2ebe4576a807d26fdc4a0ada51919": "Kraken",
	"0xda9dfa130df4de4673b89022ee50ff26f6ea73cf": "Kraken",
	"0x6cc5f688a315f3dc28a7781717a9a798a59fda7b": "OKX",
	"0x236f9f97e0e62388479bf9e5ba4889e46b0273c3": "OKX",
	"0x5041ed759dd4afc3a72b8192c143f72f4724081a": "OKX",
	"0xf89d7b9c864f589bbf53a82105107622b35eaa40": "Bybit",
	"0x1ab4973a48dc892cd9971ece8e01dcc7688f8f23": "Bybit",
	"0xee5b5b923ffce93a870b3104b7ca09c3db80047a": "Bybit",
	"0x2b5634c42055806a59e9107ed44d43c426e58258": "KuCoin",
	"0xf16e9b0d03470827a95cdfd0cb8a8a3b46969b91": "KuCoin",
	"0xd6216fc19db775df9774a6e33526131da7d19a2c": "KuCoin",
	"0x0d0707963952f2fba59dd06f2b425ace40b492fe": "Gate.io",
	"0x1c4b70a3968436b9a0a9cf5205c787eb81bb558c": "Gate.io",
	"0xd793281182a0e3e023116004778f45c29fc14f19": "Gate.io",
	"0xc882b111a75c0c657fc507c04fbfcd2cc984f071": "Gate.io",
	"0x5c985e89dde482efe97ea9f1950ad149eb73829b": "HTX",
	"0x46705dfff24256421a05d056c29e81bdc09723b8": "HTX",
	"0xeee28d484628d41a82d01e21d12e2e78d69920da": "HTX",
	"0xab5c66752a9e8167967685f1450532fb96d5d24f": "HTX",
	"0x6262998ced04146fa42253a5c0af90ca02dfd2a3": "Crypto.com",
	"0x46340b20830761efd32832a74d7169b29feb9758": "Crypto.com",
	"0xcffad3200574698b78f32232aa9d63eabd290703": "Crypto.com",
	"0xd24400ae8bfebb18ca49be86258a3c749cf46853": "Gemini",
	"0x6fc82a5fe25a5cdb58bc74600a40a69c065263f8": "Gemini",
	"0x61edcdf5bb737adffe5043706e7c5bb1f1a56eea": "Gemini",
	"0x00bdb5699745f5b860228c8f939abf1b9ae374ed": "Bitstamp",
	"0x1522900b6dafac587d499a862861c0869be6e428": "Bitstamp",
	"0x32be343b94f860124dc4fee278fdcbd38c102d88": "Poloniex",
	"0xb794f5ea0ba39494ce839613fffba74279579268": "Poloniex",
	"0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2": "WETH",
	"0x2260fac5e5542a773aa44fbcfedf7c193bc2c599": "WBTC",
	"0xae7ab96520de3a18e5e111b5eaab095312d7fe84": "Lido stETH",
}

// NormalizeHolderAddress trims and folds case for EVM and bech32.
// Base58 Bitcoin addresses stay case-sensitive.
func NormalizeHolderAddress(address string) string {
	raw := strings.TrimSpace(address)
	if raw == "" {
		return ""
	}
	lower := strings.ToLower(raw)
	if strings.HasPrefix(lower, "0x") || strings.HasPrefix(lower, "bc1") || strings.HasPrefix(lower, "tb1") {
		return lower
	}
	return raw
}

// HolderLabel returns a public attribution for address, or empty when unknown.
func HolderLabel(address string) string {
	return holderLabels[NormalizeHolderAddress(address)]
}

// AnnotateHolderLabels fills Label on each top wallet when the address is known.
// Existing non-empty labels are left as-is.
func AnnotateHolderLabels(in *AssetHolders) {
	if in == nil {
		return
	}
	for i := range in.TopHolders {
		if strings.TrimSpace(in.TopHolders[i].Label) != "" {
			continue
		}
		in.TopHolders[i].Label = HolderLabel(in.TopHolders[i].Address)
	}
}

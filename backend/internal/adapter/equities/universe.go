package equities

// Liquid Nasdaq names (Yahoo symbols). Quote currency is USD.
// Curated large/mid caps so the markets table stays useful without a paid vendor.
var nasdaqUniverse = []string{
	"AAPL", "MSFT", "NVDA", "AMZN", "GOOGL", "GOOG", "META", "TSLA", "AVGO", "COST",
	"NFLX", "AMD", "ADBE", "PEP", "CSCO", "TMUS", "INTC", "QCOM", "TXN", "AMGN",
	"INTU", "ISRG", "CMCSA", "AMAT", "HON", "SBUX", "BKNG", "VRTX", "GILD", "ADP",
	"PANW", "ADI", "LRCX", "MU", "MDLZ", "REGN", "KLAC", "SNPS", "CDNS", "ASML",
	"MELI", "CRWD", "MAR", "ORLY", "CTAS", "FTNT", "DASH", "ABNB", "PYPL", "MRVL",
	"CSX", "NXPI", "ADSK", "PCAR", "WDAY", "ROST", "AEP", "CPRT", "ODFL", "PAYX",
	"KDP", "FAST", "KHC", "GEHC", "VRSK", "EXC", "CTSH", "BKR", "XEL", "EA",
	"IDXX", "CCEP", "FANG", "CSGP", "ON", "BIIB", "ANSS", "DXCM", "CDW", "TTWO",
	"TEAM", "DDOG", "ZS", "GFS", "ARM", "SMCI", "MDB", "OKTA", "HOOD", "COIN",
	"PLTR", "SHOP", "SNOW", "NET", "MSTR", "ROKU", "DKNG", "RIVN",
}

// Liquid Borsa Istanbul names (Yahoo root; we append .IS). Quote currency is TRY.
var bistUniverse = []string{
	"THYAO", "GARAN", "AKBNK", "YKBNK", "ISCTR", "SAHOL", "KCHOL", "TUPRS", "SISE", "EREGL",
	"BIMAS", "MGROS", "AEFES", "CCOLA", "TOASO", "FROTO", "DOAS", "ASELS", "TCELL", "TTKOM",
	"TAVHL", "PGSUS", "SASA", "KRDMD", "PETKM", "TTRAK", "ARCLK", "VESTL",
	"ENKAI", "ENJSA", "AKSEN", "HEKTS", "SOKM", "MAVI", "ULKER",
	"EKGYO", "OYAKC", "CIMSA", "OTKAR", "LOGO",
	"KOZAL", "KOZAA", "GUBRF", "ALARK", "HALKB", "VAKBN", "TSKB",
}

func uniqueUpper(in []string) []string {
	seen := make(map[string]struct{}, len(in))
	out := make([]string, 0, len(in))
	for _, s := range in {
		if s == "" {
			continue
		}
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	return out
}

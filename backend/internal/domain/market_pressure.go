package domain

import (
	"fmt"
	"math"
	"strings"
	"time"
)

const (
	MarketPressureLeanBuy   = "buy"
	MarketPressureLeanSell  = "sell"
	MarketPressureLeanMixed = "mixed"

	MarketPressureAgreeSame     = "same"
	MarketPressureAgreeMixed    = "mixed"
	MarketPressureAgreeOpposite = "opposite"

	MarketPressureMoreBuy  = "more_buy"
	MarketPressureMoreSell = "more_sell"
	MarketPressureFlipped  = "flipped"
	MarketPressureStable   = "stable"

	MarketPressureCompBook = "book"
	MarketPressureCompFlow = "flow"
	MarketPressureCompAbs  = "absorption"

	marketPressureEven          = 0.08
	marketPressureChangeThresh  = 0.08
	marketPressureAbsBoostScore = 40
)

// MarketPressureInputs is one timestamp's book / CVD / absorption read.
type MarketPressureInputs struct {
	Book       float64 // order-book imbalance (bid−ask)/(bid+ask)
	BookOK     bool
	Flow       float64 // aggressive (buy−sell)/(buy+sell)
	FlowOK     bool
	AbsKind    string // bid | ask | ""
	AbsScore   int    // 0–100
	AbsOK      bool
	AbsGrade   string
	AbsSummary string
}

// MarketPressureComponent is one signed input to the blend.
type MarketPressureComponent struct {
	ID     string
	Label  string
	Value  float64
	Weight float64
	Used   bool
	Lean   string
	Detail string
}

// MarketPressureReading is the blended −1…+1 score.
type MarketPressureReading struct {
	Value      float64
	ValueText  string
	Lean       string
	Agreement  string
	Components []MarketPressureComponent
	Headline   string
	Detail     string
}

// MarketPressureWindow is live score vs a lookback.
type MarketPressureWindow struct {
	Window     string
	Value      float64
	ValueText  string
	PastValue  float64
	PastText   string
	Change     float64
	ChangeText string
	Lean       string
	Kind       string
	Complete   bool
	SampleTime time.Time
}

// MarketPressureReport is GET /api/v1/market/pressure.
type MarketPressureReport struct {
	Symbol   string
	Exchange string
	AsOf     time.Time
	Current  MarketPressureReading
	Change1h string
	Windows  []MarketPressureWindow
	Note     string
}

// BlendMarketPressure combines resting book, aggressive CVD, and absorption into −1…+1.
// +1 is full buy pressure, −1 full sell. When the book looks bid-heavy but CVD is
// selling, absorption (bids taking that flow) is weighted more so the net is not a wash.
func BlendMarketPressure(in MarketPressureInputs) MarketPressureReading {
	book := clampUnit(in.Book)
	flow := clampUnit(in.Flow)
	abs := absorptionPressureValue(in.AbsKind, in.AbsScore)
	bookW, flowW, absW := 0.34, 0.33, 0.33
	if !in.BookOK {
		bookW = 0
	}
	if !in.FlowOK {
		flowW = 0
	}
	if !in.AbsOK || in.AbsKind == "" || in.AbsScore <= 0 {
		absW = 0
		in.AbsOK = false
	}
	if in.BookOK && in.FlowOK && in.AbsOK && in.AbsScore >= marketPressureAbsBoostScore &&
		signsDisagree(book, flow) {
		bookW, flowW, absW = 0.30, 0.20, 0.50
	}
	sumW := bookW + flowW + absW
	out := MarketPressureReading{Lean: MarketPressureLeanMixed, Agreement: MarketPressureAgreeMixed}
	if sumW <= 0 {
		out.Headline = "Not enough book, CVD, or absorption to score pressure."
		return out
	}
	bookW, flowW, absW = bookW/sumW, flowW/sumW, absW/sumW
	val := bookW*book + flowW*flow + absW*abs
	val = clampUnit(val)
	out.Value = val
	out.ValueText = formatPressureValue(val)
	out.Lean = pressureLean(val)
	out.Components = []MarketPressureComponent{
		pressureComponent(MarketPressureCompBook, "Order book", book, bookW, in.BookOK, bookDetail(in.BookOK, book)),
		pressureComponent(MarketPressureCompFlow, "CVD", flow, flowW, in.FlowOK, flowDetail(in.FlowOK, flow)),
		pressureComponent(MarketPressureCompAbs, "Absorption", abs, absW, in.AbsOK, absDetail(in)),
	}
	out.Agreement = pressureAgreement(out.Components)
	out.Headline, out.Detail = explainMarketPressure(out, in)
	return out
}

// BuildMarketPressure fills current plus the 1h change.
func BuildMarketPressure(symbol, exchange string, now time.Time, current, past MarketPressureInputs, pastAt time.Time, pastComplete bool) *MarketPressureReport {
	symbol = NormalizeLiquidationSymbol(symbol)
	if exchange == "" {
		exchange = "all"
	}
	now = now.UTC()
	cur := BlendMarketPressure(current)
	out := &MarketPressureReport{
		Symbol:   symbol,
		Exchange: exchange,
		AsOf:     now,
		Current:  cur,
		Note:     "Net market pressure from −1 (sell) to +1 (buy) blends the resting order book, aggressive CVD, and absorption. When those disagree, absorption is weighted more. change1h is this score minus the score ~1h ago. Informational only — not financial advice.",
	}
	win := MarketPressureWindow{
		Window:    SnapshotWindow1h,
		Value:     cur.Value,
		ValueText: cur.ValueText,
		Lean:      cur.Lean,
		Kind:      MarketPressureStable,
		Complete:  pastComplete,
	}
	if past.BookOK || past.FlowOK || past.AbsOK {
		prev := BlendMarketPressure(past)
		win.PastValue = prev.Value
		win.PastText = prev.ValueText
		win.Change = clampUnit(cur.Value - prev.Value)
		win.ChangeText = formatPressureValue(win.Change)
		win.Kind = marketPressureChangeKind(prev.Value, cur.Value)
		win.Complete = pastComplete
		if !pastAt.IsZero() {
			win.SampleTime = pastAt.UTC()
		}
		out.Change1h = win.ChangeText
	}
	out.Windows = []MarketPressureWindow{win}
	if out.Change1h != "" && win.Kind != "" && win.Kind != MarketPressureStable {
		out.Current.Headline = strings.TrimSpace(out.Current.Headline + " " + marketPressureChangeLine(win))
	}
	return out
}

func absorptionPressureValue(kind string, score int) float64 {
	if score <= 0 {
		return 0
	}
	v := float64(score) / 100
	if v > 1 {
		v = 1
	}
	switch kind {
	case AbsorptionKindBid:
		return v
	case AbsorptionKindAsk:
		return -v
	default:
		return 0
	}
}

func pressureComponent(id, label string, value, weight float64, used bool, detail string) MarketPressureComponent {
	c := MarketPressureComponent{
		ID: id, Label: label, Value: value, Weight: weight, Used: used, Detail: detail,
		Lean: MarketPressureLeanMixed,
	}
	if used {
		c.Lean = pressureLean(value)
	}
	return c
}

func pressureLean(v float64) string {
	switch {
	case v >= marketPressureEven:
		return MarketPressureLeanBuy
	case v <= -marketPressureEven:
		return MarketPressureLeanSell
	default:
		return MarketPressureLeanMixed
	}
}

func signsDisagree(a, b float64) bool {
	return pressureLean(a) == MarketPressureLeanBuy && pressureLean(b) == MarketPressureLeanSell ||
		pressureLean(a) == MarketPressureLeanSell && pressureLean(b) == MarketPressureLeanBuy
}

func pressureAgreement(comps []MarketPressureComponent) string {
	var buy, sell int
	for _, c := range comps {
		if !c.Used {
			continue
		}
		switch c.Lean {
		case MarketPressureLeanBuy:
			buy++
		case MarketPressureLeanSell:
			sell++
		}
	}
	switch {
	case buy > 0 && sell > 0:
		return MarketPressureAgreeOpposite
	case buy >= 2 || sell >= 2:
		return MarketPressureAgreeSame
	default:
		return MarketPressureAgreeMixed
	}
}

func explainMarketPressure(out MarketPressureReading, in MarketPressureInputs) (headline, detail string) {
	book, flow, abs := componentByID(out.Components, MarketPressureCompBook),
		componentByID(out.Components, MarketPressureCompFlow),
		componentByID(out.Components, MarketPressureCompAbs)
	switch {
	case out.Agreement == MarketPressureAgreeOpposite && abs.Used && signsDisagree(book.Value, flow.Value):
		if abs.Lean == MarketPressureLeanBuy {
			headline = fmt.Sprintf("The order book looks bid-heavy while CVD is selling, but bids are absorbing that flow. Net pressure %s.", out.ValueText)
		} else {
			headline = fmt.Sprintf("The order book looks offer-heavy while CVD is buying, but asks are absorbing that flow. Net pressure %s.", out.ValueText)
		}
	case out.Agreement == MarketPressureAgreeOpposite:
		headline = fmt.Sprintf("Book, CVD, and absorption disagree. Net pressure %s.", out.ValueText)
	case out.Lean == MarketPressureLeanBuy:
		headline = fmt.Sprintf("Net buy pressure %s.", out.ValueText)
	case out.Lean == MarketPressureLeanSell:
		headline = fmt.Sprintf("Net sell pressure %s.", out.ValueText)
	default:
		headline = fmt.Sprintf("Mixed pressure %s.", out.ValueText)
	}
	parts := make([]string, 0, 3)
	if book.Used {
		parts = append(parts, book.Detail)
	}
	if flow.Used {
		parts = append(parts, flow.Detail)
	}
	if abs.Used {
		parts = append(parts, abs.Detail)
		if in.AbsSummary != "" {
			parts = append(parts, in.AbsSummary)
		}
	}
	detail = strings.Join(parts, " ")
	return headline, detail
}

func componentByID(comps []MarketPressureComponent, id string) MarketPressureComponent {
	for _, c := range comps {
		if c.ID == id {
			return c
		}
	}
	return MarketPressureComponent{}
}

func bookDetail(ok bool, v float64) string {
	if !ok {
		return "No live order book."
	}
	switch pressureLean(v) {
	case MarketPressureLeanBuy:
		return fmt.Sprintf("Resting book is bid-heavy (imbalance %s).", formatPressureValue(v))
	case MarketPressureLeanSell:
		return fmt.Sprintf("Resting book is offer-heavy (imbalance %s).", formatPressureValue(v))
	default:
		return fmt.Sprintf("Resting book is balanced (imbalance %s).", formatPressureValue(v))
	}
}

func flowDetail(ok bool, v float64) string {
	if !ok {
		return "No CVD window."
	}
	switch pressureLean(v) {
	case MarketPressureLeanBuy:
		return fmt.Sprintf("CVD is buying (flow %s).", formatPressureValue(v))
	case MarketPressureLeanSell:
		return fmt.Sprintf("CVD is selling (flow %s).", formatPressureValue(v))
	default:
		return fmt.Sprintf("CVD is mixed (flow %s).", formatPressureValue(v))
	}
}

func absDetail(in MarketPressureInputs) string {
	if !in.AbsOK || in.AbsKind == "" {
		return "No absorption."
	}
	grade := in.AbsGrade
	if grade == "" {
		grade = AbsorptionGrade(in.AbsScore)
	}
	if in.AbsKind == AbsorptionKindBid {
		return fmt.Sprintf("Bids absorbing market sells (%s, score %d).", grade, in.AbsScore)
	}
	return fmt.Sprintf("Asks absorbing market buys (%s, score %d).", grade, in.AbsScore)
}

func marketPressureChangeKind(past, now float64) string {
	pastL, nowL := pressureLean(past), pressureLean(now)
	if pastL != nowL && pastL != MarketPressureLeanMixed && nowL != MarketPressureLeanMixed {
		return MarketPressureFlipped
	}
	d := now - past
	switch {
	case d > marketPressureChangeThresh:
		return MarketPressureMoreBuy
	case d < -marketPressureChangeThresh:
		return MarketPressureMoreSell
	default:
		return MarketPressureStable
	}
}

func marketPressureChangeLine(w MarketPressureWindow) string {
	switch w.Kind {
	case MarketPressureFlipped:
		return fmt.Sprintf("Over 1h the score flipped (%s → %s).", w.PastText, w.ValueText)
	case MarketPressureMoreBuy:
		return fmt.Sprintf("Over 1h pressure moved %s toward buy.", w.ChangeText)
	case MarketPressureMoreSell:
		return fmt.Sprintf("Over 1h pressure moved %s toward sell.", w.ChangeText)
	default:
		return ""
	}
}

func clampUnit(v float64) float64 {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return 0
	}
	if v > 1 {
		return 1
	}
	if v < -1 {
		return -1
	}
	return v
}

// FormatMarketPressureValue is the signed −1…+1 text used on the wire.
func FormatMarketPressureValue(v float64) string {
	return formatPressureValue(v)
}

func formatPressureValue(v float64) string {
	if math.Abs(v) < 0.005 {
		return "0"
	}
	s := formatFixed(v, 2)
	if v > 0 && !strings.HasPrefix(s, "-") {
		return "+" + s
	}
	return s
}

// CVDFlowShare is aggressive (buy−sell)/(buy+sell) for one CVD window.
func CVDFlowShare(w CVDWindowStat) (float64, bool) {
	tot := w.BuyNotional + w.SellNotional
	if tot <= 0 {
		return 0, false
	}
	return clampUnit(w.CVDChange / tot), true
}

// CVDFlowShareInRange sums 5m buy/sell inside [from, to].
func CVDFlowShareInRange(points []CVDPoint, from, to time.Time) (float64, bool) {
	var buy, sell float64
	for _, p := range points {
		if p.Time.Before(from) || p.Time.After(to) {
			continue
		}
		buy += p.BuyNotional
		sell += p.SellNotional
	}
	tot := buy + sell
	if tot <= 0 {
		return 0, false
	}
	return clampUnit((buy - sell) / tot), true
}

// AbsorptionSignedFromWindow reads kind/score from a named absorption window.
func AbsorptionSignedFromWindow(windows []AbsorptionWindowStat, id string) (kind string, score int, ok bool) {
	for _, w := range windows {
		if w.Window == id && w.Kind != "" && w.Score > 0 {
			return w.Kind, w.Score, true
		}
	}
	return "", 0, false
}

// AbsorptionInRange classifies absorption from 5m points inside [from, to].
func AbsorptionInRange(points []AbsorptionPoint, from, to time.Time, flatPct float64) (kind string, score int, ok bool) {
	var buy, sell, pxFrom, pxTo float64
	var gotFrom, gotTo bool
	for _, p := range points {
		if p.Time.Before(from) || p.Time.After(to) {
			continue
		}
		buy += p.BuyNotional
		sell += p.SellNotional
		if p.Price <= 0 {
			continue
		}
		if !gotFrom {
			pxFrom, gotFrom = p.Price, true
		}
		pxTo, gotTo = p.Price, true
	}
	if buy+sell <= 0 {
		return "", 0, false
	}
	chg := 0.0
	if gotFrom && gotTo && pxFrom > 0 {
		chg = (pxTo - pxFrom) / pxFrom * 100
	}
	if flatPct <= 0 {
		flatPct = 0.35
	}
	pt := ClassifyAbsorption(buy, sell, chg, flatPct)
	if pt.Kind == "" || pt.Score <= 0 {
		return "", 0, false
	}
	return pt.Kind, pt.Score, true
}

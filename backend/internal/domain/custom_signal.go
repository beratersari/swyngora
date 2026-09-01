package domain

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

// Custom signal mix (Signals desk). Same swing-pattern ids as the engine,
// plus optional hunt_up from the liquidation-hunt combined lean.
const (
	CustomFactorEMACrossover   = "ema_crossover"
	CustomFactorEMATrend       = "ema_trend_bias"
	CustomFactorSupertrend     = "supertrend"
	CustomFactorRSIRecovery    = "rsi_recovery"
	CustomFactorMACD           = "macd_momentum"
	CustomFactorADX            = "adx_trend_strength"
	CustomFactorAccumulation   = "accumulation"
	CustomFactorVolumeBreakout = "volume_breakout"
	CustomFactorHuntUp         = "hunt_up"

	CustomFactorUsed     = "used"
	CustomFactorMissing  = "missing"
	CustomFactorDisabled = "disabled"

	CustomSignalMixDefault = "default"
	CustomSignalMixCustom  = "custom"

	MaxCustomSignalProfilesPerClient = 20
	MaxCustomSignalProfileNameLen    = 80
)

// CustomSignalFactorIDs is the mix in display order (swing first, hunt last).
var CustomSignalFactorIDs = []string{
	CustomFactorEMACrossover, CustomFactorEMATrend, CustomFactorSupertrend,
	CustomFactorRSIRecovery, CustomFactorMACD, CustomFactorADX,
	CustomFactorAccumulation, CustomFactorVolumeBreakout, CustomFactorHuntUp,
}

// CustomSignalSwingFactorIDs are the engine pattern names (no hunt).
var CustomSignalSwingFactorIDs = []string{
	CustomFactorEMACrossover, CustomFactorEMATrend, CustomFactorSupertrend,
	CustomFactorRSIRecovery, CustomFactorMACD, CustomFactorADX,
	CustomFactorAccumulation, CustomFactorVolumeBreakout,
}

// CustomSignalQueryKeys maps factor id → query/MCP parameter name.
var CustomSignalQueryKeys = map[string]string{
	CustomFactorEMACrossover:   "weightEmaCrossover",
	CustomFactorEMATrend:       "weightEmaTrend",
	CustomFactorSupertrend:     "weightSupertrend",
	CustomFactorRSIRecovery:    "weightRsiRecovery",
	CustomFactorMACD:           "weightMacd",
	CustomFactorADX:            "weightAdx",
	CustomFactorAccumulation:   "weightAccumulation",
	CustomFactorVolumeBreakout: "weightVolumeBreakout",
	CustomFactorHuntUp:         "weightHuntUp",
}

var customSignalFactorLabel = map[string]string{
	CustomFactorEMACrossover:   "EMA 9/21 cross",
	CustomFactorEMATrend:       "EMA 50/200 pullback",
	CustomFactorSupertrend:     "SuperTrend",
	CustomFactorRSIRecovery:    "RSI recovery",
	CustomFactorMACD:           "MACD histogram",
	CustomFactorADX:            "ADX trend strength",
	CustomFactorAccumulation:   "BB squeeze / dry-up",
	CustomFactorVolumeBreakout: "Volume breakout",
	CustomFactorHuntUp:         "Liquidation hunt (up)",
}

// defaultCustomSignalPct matches swingWeights (sums to 100). Hunt is off.
var defaultCustomSignalPct = map[string]float64{
	CustomFactorEMACrossover:   16,
	CustomFactorEMATrend:       12,
	CustomFactorSupertrend:     13,
	CustomFactorRSIRecovery:    13,
	CustomFactorMACD:           12,
	CustomFactorADX:            12,
	CustomFactorAccumulation:   10,
	CustomFactorVolumeBreakout: 12,
	CustomFactorHuntUp:         0,
}

// CustomSignalWeights is the requested mix. Percents stay as given.
type CustomSignalWeights struct {
	Source string
	Pct    map[string]float64
}

// CustomSignalProfile is a saved named mix for one client.
type CustomSignalProfile struct {
	ID        string
	ClientID  string
	Name      string
	Weights   CustomSignalWeights
	CreatedAt time.Time
	UpdatedAt time.Time
}

// CustomSignalProfilePort persists named custom mixes.
type CustomSignalProfilePort interface {
	CreateProfile(ctx context.Context, p CustomSignalProfile) (*CustomSignalProfile, error)
	GetProfile(ctx context.Context, clientID, id string) (*CustomSignalProfile, error)
	UpdateProfile(ctx context.Context, p CustomSignalProfile) (*CustomSignalProfile, error)
	ListProfiles(ctx context.Context, clientID string) ([]CustomSignalProfile, error)
	DeleteProfile(ctx context.Context, clientID, id string) error
	CountProfiles(ctx context.Context, clientID string) (int, error)
	PurgeProfiles(ctx context.Context, clientID string) error
}

// CustomSignalCatalogRow describes one mixable factor.
type CustomSignalCatalogRow struct {
	ID         string  `json:"id"`
	Label      string  `json:"label"`
	Group      string  `json:"group"`
	DefaultPct float64 `json:"defaultPct"`
}

// CustomSignalRaw is one factor's 0–100 input before the mix.
type CustomSignalRaw struct {
	Score  float64
	Status string
	Detail string
}

// CustomSignalWeightRow is one factor in the requested or used mix.
type CustomSignalWeightRow struct {
	ID        string  `json:"id"`
	Label     string  `json:"label"`
	WeightPct float64 `json:"weightPct"`
	Status    string  `json:"status"`
	Detail    string  `json:"detail,omitempty"`
}

// CustomSignalMix is how the custom score was built.
type CustomSignalMix struct {
	Source         string                  `json:"source"`
	Requested      []CustomSignalWeightRow `json:"requested"`
	Used           []CustomSignalWeightRow `json:"used"`
	RequestedTotal float64                 `json:"requestedTotal"`
	UsedTotal      float64                 `json:"usedTotal"`
	Missing        []string                `json:"missing"`
	Disabled       []string                `json:"disabled"`
	Note           string                  `json:"note"`
}

// CustomSignalFactor is one scored driver of the custom mix.
type CustomSignalFactor struct {
	ID           string  `json:"id"`
	Label        string  `json:"label"`
	Score        float64 `json:"score"`
	RequestedPct float64 `json:"requestedPct"`
	SharePct     float64 `json:"sharePct"`
	Effect       float64 `json:"effect"`
	Status       string  `json:"status"`
	Detail       string  `json:"detail,omitempty"`
}

// CustomSignalScore is the 0–100 custom mix for one symbol.
type CustomSignalScore struct {
	Score   float64              `json:"score"`
	Grade   string               `json:"grade"`
	Factors []CustomSignalFactor `json:"factors"`
	Mix     CustomSignalMix      `json:"scoreMix"`
}

// CustomSignalSnapshot is one mix's score (default or applied).
type CustomSignalSnapshot struct {
	Source  string                      `json:"source"`
	Score   float64                     `json:"score"`
	Grade   string                      `json:"grade"`
	Factors []CustomSignalFactorCompare `json:"factors"`
}

// CustomSignalFactorCompare is one factor under default vs applied weights.
type CustomSignalFactorCompare struct {
	ID            string  `json:"id"`
	Label         string  `json:"label"`
	Status        string  `json:"status"`
	Score         float64 `json:"score"`
	DefaultPct    float64 `json:"defaultPct"`
	AppliedPct    float64 `json:"appliedPct"`
	DefaultEffect float64 `json:"defaultEffect"`
	AppliedEffect float64 `json:"appliedEffect"`
	DeltaEffect   float64 `json:"deltaEffect"`
	Detail        string  `json:"detail,omitempty"`
}

// CustomSignalLargestChange is the factor that moved the score the most.
type CustomSignalLargestChange struct {
	ID            string  `json:"id"`
	Label         string  `json:"label"`
	Status        string  `json:"status"`
	Score         float64 `json:"score"`
	DefaultPct    float64 `json:"defaultPct"`
	AppliedPct    float64 `json:"appliedPct"`
	DefaultEffect float64 `json:"defaultEffect"`
	AppliedEffect float64 `json:"appliedEffect"`
	DeltaEffect   float64 `json:"deltaEffect"`
	Summary       string  `json:"summary"`
}

// CustomSignalDelta is applied minus default.
type CustomSignalDelta struct {
	Score         float64                     `json:"score"`
	Factors       []CustomSignalFactorCompare `json:"factors"`
	LargestChange *CustomSignalLargestChange  `json:"largestChange,omitempty"`
}

// CustomSignalCompare is default vs applied mix for the desk and for AI.
type CustomSignalCompare struct {
	Default CustomSignalSnapshot `json:"default"`
	Applied CustomSignalSnapshot `json:"applied"`
	Delta   CustomSignalDelta    `json:"delta"`
	Note    string               `json:"note"`
}

// CustomSignalItem is one watchlist row after the custom mix.
type CustomSignalItem struct {
	Exchange   Exchange            `json:"exchange"`
	Symbol     string              `json:"symbol"`
	Interval   string              `json:"interval"`
	Price      float64             `json:"price"`
	BarTime    time.Time           `json:"barTime"`
	Engine     *SwingDecision      `json:"engine,omitempty"`
	Custom     CustomSignalScore   `json:"custom"`
	Compare    CustomSignalCompare `json:"scoreCompare"`
	HuntDetail string              `json:"huntDetail,omitempty"`
}

// CustomSignalFactorLabel is the desk label for a factor id.
func CustomSignalFactorLabel(id string) string {
	if s := customSignalFactorLabel[id]; s != "" {
		return s
	}
	return id
}

// CustomSignalFactorGroup is "swing" or "hunt".
func CustomSignalFactorGroup(id string) string {
	if id == CustomFactorHuntUp {
		return "hunt"
	}
	return "swing"
}

// CustomSignalCatalog lists mixable factors and built-in percents.
func CustomSignalCatalog() []CustomSignalCatalogRow {
	out := make([]CustomSignalCatalogRow, 0, len(CustomSignalFactorIDs))
	for _, id := range CustomSignalFactorIDs {
		out = append(out, CustomSignalCatalogRow{
			ID: id, Label: CustomSignalFactorLabel(id),
			Group: CustomSignalFactorGroup(id), DefaultPct: defaultCustomSignalPct[id],
		})
	}
	return out
}

// DefaultCustomSignalWeights is the built-in swing mix (hunt off).
func DefaultCustomSignalWeights() CustomSignalWeights {
	pct := make(map[string]float64, len(defaultCustomSignalPct))
	for id, v := range defaultCustomSignalPct {
		pct[id] = v
	}
	return CustomSignalWeights{Source: CustomSignalMixDefault, Pct: pct}
}

// ParseCustomSignalWeights reads weightEmaCrossover… query/MCP fields.
// No weight params → defaults. Any present → custom; omitted keys are 0.
// Custom mix must sum to 100; it is never renormalized.
func ParseCustomSignalWeights(get func(string) string) (CustomSignalWeights, error) {
	any := false
	raw := make(map[string]string, len(CustomSignalFactorIDs))
	for _, id := range CustomSignalFactorIDs {
		v := strings.TrimSpace(get(CustomSignalQueryKeys[id]))
		raw[id] = v
		if v != "" {
			any = true
		}
	}
	if !any {
		return DefaultCustomSignalWeights(), nil
	}
	return parseCustomSignalPctMap(raw, true)
}

// ParseCustomSignalWeightMap builds a custom mix from a JSON object of factor id → percent.
// Omitted keys are 0. Must sum to 100.
func ParseCustomSignalWeightMap(in map[string]float64) (CustomSignalWeights, error) {
	raw := make(map[string]string, len(CustomSignalFactorIDs))
	for _, id := range CustomSignalFactorIDs {
		if in == nil {
			raw[id] = ""
			continue
		}
		if v, ok := in[id]; ok {
			raw[id] = strconv.FormatFloat(v, 'f', -1, 64)
		}
	}
	return parseCustomSignalPctMap(raw, true)
}

func parseCustomSignalPctMap(raw map[string]string, requireSum100 bool) (CustomSignalWeights, error) {
	out := CustomSignalWeights{Source: CustomSignalMixCustom, Pct: make(map[string]float64, len(CustomSignalFactorIDs))}
	var sum float64
	for _, id := range CustomSignalFactorIDs {
		if strings.TrimSpace(raw[id]) == "" {
			out.Pct[id] = 0
			continue
		}
		n, err := strconv.ParseFloat(raw[id], 64)
		if err != nil || math.IsNaN(n) || math.IsInf(n, 0) {
			return CustomSignalWeights{}, fmt.Errorf("%w: %s is not a number", ErrInvalidArgument, CustomSignalQueryKeys[id])
		}
		if n < 0 {
			return CustomSignalWeights{}, fmt.Errorf("%w: %s cannot be negative", ErrInvalidArgument, CustomSignalQueryKeys[id])
		}
		out.Pct[id] = n
		sum += n
	}
	if requireSum100 && math.Abs(sum-100) > 0.05 {
		return CustomSignalWeights{}, fmt.Errorf("%w: custom signal weights sum to %.2f, not 100 — not applied", ErrInvalidArgument, sum)
	}
	if sum <= 0 {
		return CustomSignalWeights{}, fmt.Errorf("%w: enable at least one factor", ErrInvalidArgument)
	}
	return out, nil
}

func (w CustomSignalWeights) pctOf(id string) float64 {
	if w.Pct == nil {
		return defaultCustomSignalPct[id]
	}
	return w.Pct[id]
}

// NeedsHunt reports whether the mix assigns weight to liquidation hunt.
func (w CustomSignalWeights) NeedsHunt() bool {
	return w.pctOf(CustomFactorHuntUp) > 0
}

// NormalizeCustomSignalProfileName trims and checks length.
func NormalizeCustomSignalProfileName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", fmt.Errorf("%w: name is required", ErrInvalidArgument)
	}
	if utf8.RuneCountInString(name) > MaxCustomSignalProfileNameLen {
		return "", fmt.Errorf("%w: name must be at most %d characters", ErrInvalidArgument, MaxCustomSignalProfileNameLen)
	}
	return name, nil
}

// BuildCustomSignalRaws maps engine patterns + optional hunt into mix inputs.
// A swing pattern that did not fire is used at score 0 (not missing).
// Hunt is missing when the report is nil or not usable.
func BuildCustomSignalRaws(patterns []SwingPattern, hunt *CustomSignalRaw) map[string]CustomSignalRaw {
	byName := map[string]SwingPattern{}
	for _, p := range patterns {
		if prev, ok := byName[p.Name]; !ok || p.Score > prev.Score {
			byName[p.Name] = p
		}
	}
	out := make(map[string]CustomSignalRaw, len(CustomSignalFactorIDs))
	for _, id := range CustomSignalSwingFactorIDs {
		if p, ok := byName[id]; ok {
			out[id] = CustomSignalRaw{Score: p.Score, Status: CustomFactorUsed, Detail: p.Description}
			continue
		}
		out[id] = CustomSignalRaw{Score: 0, Status: CustomFactorUsed, Detail: "pattern not present"}
	}
	if hunt != nil && hunt.Status != "" {
		out[CustomFactorHuntUp] = *hunt
	} else {
		out[CustomFactorHuntUp] = CustomSignalRaw{
			Score: 50, Status: CustomFactorMissing, Detail: "hunt not loaded",
		}
	}
	return out
}

// HuntUpRaw maps a hunt report onto the long-side custom factor.
func HuntUpRaw(report *HuntReport) CustomSignalRaw {
	if report == nil || report.Bias == nil {
		return CustomSignalRaw{Score: 50, Status: CustomFactorMissing, Detail: "hunt unavailable"}
	}
	if !report.Bias.Coverage.Usable {
		detail := "hunt not usable"
		if report.Bias.Summary != "" {
			detail = report.Bias.Summary
		}
		return CustomSignalRaw{Score: 50, Status: CustomFactorMissing, Detail: detail}
	}
	return CustomSignalRaw{
		Score:  report.Bias.UpScore,
		Status: CustomFactorUsed,
		Detail: fmt.Sprintf("up %.1f / down %.1f · %s", report.Bias.UpScore, report.Bias.DownScore, report.Bias.Lean),
	}
}

// ScoreCustomSignal mixes raw factor scores. Requested percents stay as given.
// Missing or disabled factors keep their share and add 0 (neutral 50).
// Score = 50 + sum(effect), clamped 0–100.
func ScoreCustomSignal(raws map[string]CustomSignalRaw, w CustomSignalWeights) CustomSignalScore {
	if w.Pct == nil {
		w = DefaultCustomSignalWeights()
	}
	factors := make([]CustomSignalFactor, 0, len(CustomSignalFactorIDs))
	var rawScore float64 = 50
	used := make([]CustomSignalWeightRow, 0, len(CustomSignalFactorIDs))
	requested := make([]CustomSignalWeightRow, 0, len(CustomSignalFactorIDs))
	var missing, disabled []string
	var reqTotal, usedTotal float64
	for _, id := range CustomSignalFactorIDs {
		pct := w.pctOf(id)
		src := raws[id]
		row := CustomSignalWeightRow{
			ID: id, Label: CustomSignalFactorLabel(id), WeightPct: round1(pct),
		}
		reqTotal += pct
		f := CustomSignalFactor{
			ID: id, Label: CustomSignalFactorLabel(id),
			Score: round1(src.Score), RequestedPct: round1(pct),
			SharePct: round1(pct), Detail: src.Detail,
		}
		switch {
		case pct <= 0:
			f.Status = CustomFactorDisabled
			row.Status = CustomFactorDisabled
			disabled = append(disabled, id)
		case src.Status == CustomFactorMissing || src.Status == "":
			f.Status = CustomFactorMissing
			row.Status = CustomFactorMissing
			row.Detail = src.Detail
			missing = append(missing, id)
		default:
			effect := (src.Score - 50) * (pct / 100)
			rawScore += effect
			f.Effect = round1(effect)
			f.Status = CustomFactorUsed
			row.Status = CustomFactorUsed
			row.Detail = src.Detail
			used = append(used, row)
			usedTotal += pct
		}
		requested = append(requested, row)
		factors = append(factors, f)
	}
	score := clampScore(rawScore)
	src := w.Source
	if src == "" {
		src = CustomSignalMixCustom
	}
	note := "Requested percents stay as given. A missing factor keeps its share and adds 0 (neutral 50)."
	return CustomSignalScore{
		Score:   score,
		Grade:   customSignalGrade(score),
		Factors: factors,
		Mix: CustomSignalMix{
			Source: src, Requested: requested, Used: used,
			RequestedTotal: round1(reqTotal), UsedTotal: round1(usedTotal),
			Missing: missing, Disabled: disabled, Note: note,
		},
	}
}

// CompareCustomSignal is default swing mix vs the applied mix on the same raws.
func CompareCustomSignal(raws map[string]CustomSignalRaw, applied CustomSignalWeights) CustomSignalCompare {
	defW := DefaultCustomSignalWeights()
	def := ScoreCustomSignal(raws, defW)
	app := ScoreCustomSignal(raws, applied)
	factors := make([]CustomSignalFactorCompare, 0, len(CustomSignalFactorIDs))
	var largest *CustomSignalLargestChange
	for i, id := range CustomSignalFactorIDs {
		df, af := def.Factors[i], app.Factors[i]
		row := CustomSignalFactorCompare{
			ID: id, Label: af.Label, Status: af.Status, Score: af.Score,
			DefaultPct: df.RequestedPct, AppliedPct: af.RequestedPct,
			DefaultEffect: df.Effect, AppliedEffect: af.Effect,
			DeltaEffect: round1(af.Effect - df.Effect), Detail: af.Detail,
		}
		factors = append(factors, row)
		if math.Abs(row.DeltaEffect) < 0.05 {
			continue
		}
		if largest == nil || math.Abs(row.DeltaEffect) > math.Abs(largest.DeltaEffect) {
			cp := row
			largest = &CustomSignalLargestChange{
				ID: cp.ID, Label: cp.Label, Status: cp.Status, Score: cp.Score,
				DefaultPct: cp.DefaultPct, AppliedPct: cp.AppliedPct,
				DefaultEffect: cp.DefaultEffect, AppliedEffect: cp.AppliedEffect,
				DeltaEffect: cp.DeltaEffect,
				Summary:     fmt.Sprintf("%s moved the score %+.1f vs the built-in mix", cp.Label, cp.DeltaEffect),
			}
		}
	}
	return CustomSignalCompare{
		Default: CustomSignalSnapshot{
			Source: CustomSignalMixDefault, Score: def.Score, Grade: def.Grade, Factors: factors,
		},
		Applied: CustomSignalSnapshot{
			Source: app.Mix.Source, Score: app.Score, Grade: app.Grade, Factors: factors,
		},
		Delta: CustomSignalDelta{
			Score: round1(app.Score - def.Score), Factors: factors, LargestChange: largest,
		},
		Note: "Default is the built-in swing mix (hunt off). Applied uses the selected weights. Missing factors add 0.",
	}
}

func customSignalGrade(score float64) string {
	if score >= 78 {
		return "A"
	}
	if score >= 62 {
		return "B"
	}
	return "C"
}

// SortCustomSignalItems puts higher custom score first.
func SortCustomSignalItems(in []CustomSignalItem) {
	sort.SliceStable(in, func(i, j int) bool {
		if in[i].Custom.Score != in[j].Custom.Score {
			return in[i].Custom.Score > in[j].Custom.Score
		}
		return in[i].Symbol < in[j].Symbol
	})
}

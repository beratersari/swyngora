package domain

// Emergency stop kinds. Each one is one step the stop tried.
const (
	EmergencyCancelOrder   = "cancel_order"
	EmergencyCancelSpread  = "cancel_spread"
	EmergencyPausePlan     = "pause_recurring"
	EmergencyCloseSpot     = "close_spot"
	EmergencyCloseMargin   = "close_margin"
	EmergencyCancelPending = "cancel_pending"

	EmergencyOK     = "ok"
	EmergencyFailed = "failed"
)

// EmergencyStop is the switch for one paper book.
// LossPct is today's trading-loss cap. 5 means 5 percent.
type EmergencyStop struct {
	ClientID  string
	Enabled   bool
	LossPct   float64
	Tripped   bool
	RunID     string
	UpdatedAt string
}

// EmergencyAction is one cancel, pause, or close.
// TargetID is the order, plan, position, or exchange|symbol.
// Status is ok or failed. Detail is set when the step failed.
type EmergencyAction struct {
	Kind     string `json:"kind"`
	TargetID string `json:"targetId"`
	Exchange string `json:"exchange,omitempty"`
	Symbol   string `json:"symbol,omitempty"`
	Status   string `json:"status"`
	Detail   string `json:"detail,omitempty"`
}

// EmergencyStopRun is one trip past the loss line, including later retries.
type EmergencyStopRun struct {
	ID          string
	ClientID    string
	CreatedAt   string
	UpdatedAt   string
	DailyPnLPct float64
	LossPct     float64
	Actions     []EmergencyAction
}

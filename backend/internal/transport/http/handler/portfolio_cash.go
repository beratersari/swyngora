package handler

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
	"gitlab.com/trace-analysis/swyngora/backend/internal/service/portfolio"
)

type cashMoveBody struct {
	ClientID    string  `json:"clientId"`
	PortfolioID string  `json:"portfolioId"`
	Amount      float64 `json:"amount"`
	Note        string  `json:"note"`
}

type cashMovementDTO struct {
	ID                        string  `json:"id"`
	Kind                      string  `json:"kind"`
	Amount                    float64 `json:"amount"`
	CashAfter                 float64 `json:"cashAfter"`
	NetDepositsAfter          float64 `json:"netDepositsAfter"`
	Note                      string  `json:"note,omitempty"`
	CounterpartyPortfolioID   string  `json:"counterpartyPortfolioId,omitempty"`
	CounterpartyPortfolioName string  `json:"counterpartyPortfolioName,omitempty"`
	PeerMovementID            string  `json:"peerMovementId,omitempty"`
	CreatedAt                 string  `json:"createdAt"`
}

func cashMovementToDTO(m *domain.CashMovement) cashMovementDTO {
	return cashMovementDTO{
		ID: m.ID, Kind: string(m.Kind), Amount: m.Amount,
		CashAfter: m.CashAfter, NetDepositsAfter: m.NetDepositsAfter, Note: m.Note,
		CounterpartyPortfolioID: m.CounterpartyPortfolioID, CounterpartyPortfolioName: m.CounterpartyPortfolioName,
		PeerMovementID: m.PeerMovementID,
		CreatedAt:      m.CreatedAt.UTC().Format(time.RFC3339Nano),
	}
}

// Deposit handles POST /api/v1/portfolio/deposits
func (h *PortfolioHandler) Deposit(w http.ResponseWriter, r *http.Request) {
	h.cashMove(w, r, domain.CashMovementDeposit)
}

// Withdraw handles POST /api/v1/portfolio/withdrawals
func (h *PortfolioHandler) Withdraw(w http.ResponseWriter, r *http.Request) {
	h.cashMove(w, r, domain.CashMovementWithdrawal)
}

func (h *PortfolioHandler) cashMove(w http.ResponseWriter, r *http.Request, kind domain.CashMovementKind) {
	var body cashMoveBody
	if err := decodeJSON(r, &body, DefaultMaxJSONBody); err != nil {
		writeError(w, fmt.Errorf("%w: invalid JSON body", domain.ErrInvalidArgument))
		return
	}
	clientID := body.ClientID
	if clientID == "" {
		clientID = clientIDFrom(r)
	}
	in := portfolio.CashMoveInput{ClientID: clientID, PortfolioID: coalescePortfolioID(r, body.PortfolioID), Amount: body.Amount, Note: body.Note}
	var (
		m   *domain.CashMovement
		v   *domain.PortfolioView
		err error
	)
	if kind == domain.CashMovementDeposit {
		m, v, err = h.svc.Deposit(r.Context(), in)
	} else {
		m, v, err = h.svc.Withdraw(r.Context(), in)
	}
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"movement":  cashMovementToDTO(m),
		"portfolio": portfolioViewDTO(v),
	})
}

// ListCashMovements handles GET /api/v1/portfolio/cash-movements
func (h *PortfolioHandler) ListCashMovements(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	limit, _ := strconv.Atoi(q.Get("limit"))
	offset, _ := strconv.Atoi(q.Get("offset"))
	list, total, err := h.svc.ListCashMovements(r.Context(), clientIDFrom(r), limit, offset, portfolioIDFrom(r))
	if err != nil {
		writeError(w, err)
		return
	}
	items := make([]cashMovementDTO, 0, len(list))
	for i := range list {
		items = append(items, cashMovementToDTO(&list[i]))
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"clientId":   clientIDFrom(r),
		"movements":  items,
		"count":      len(items),
		"total":      total,
		"limit":      limit,
		"offset":     offset,
	})
}

type transferBody struct {
	ClientID        string  `json:"clientId"`
	FromPortfolioID string  `json:"fromPortfolioId"`
	ToPortfolioID   string  `json:"toPortfolioId"`
	Amount          float64 `json:"amount"`
	Note            string  `json:"note"`
}

// Transfer handles POST /api/v1/portfolio/transfers (owner only).
func (h *PortfolioHandler) Transfer(w http.ResponseWriter, r *http.Request) {
	var body transferBody
	if err := decodeJSON(r, &body, DefaultMaxJSONBody); err != nil {
		writeError(w, fmt.Errorf("%w: invalid JSON body", domain.ErrInvalidArgument))
		return
	}
	clientID := body.ClientID
	if clientID == "" {
		clientID = clientIDFrom(r)
	}
	fromID := strings.TrimSpace(body.FromPortfolioID)
	if fromID == "" {
		fromID = portfolioIDFrom(r)
	}
	out, in, fromV, toV, err := h.svc.Transfer(r.Context(), portfolio.TransferInput{
		ClientID: clientID, FromPortfolioID: fromID, ToPortfolioID: body.ToPortfolioID,
		Amount: body.Amount, Note: body.Note,
	})
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"from": map[string]any{"movement": cashMovementToDTO(out), "portfolio": portfolioViewDTO(fromV)},
		"to":   map[string]any{"movement": cashMovementToDTO(in), "portfolio": portfolioViewDTO(toV)},
		"note": "Internal transfer between your paper portfolios. Not a deposit or withdrawal. Simulated only.",
	})
}

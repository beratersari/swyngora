package domain

import (
	"fmt"
	"strings"
)

// PortfolioSnapshot is a full paper-book backup (balance, positions, history, orders, lots, margin, shares).
type PortfolioSnapshot struct {
	Book            Portfolio
	Positions       []Position
	Trades          []Trade
	OpenOrders      []PendingOrder
	Lots            []TaxLot
	LotFills        []TaxLotFill
	RecurringPlans  []RecurringBuyPlan
	RecurringRuns   []RecurringBuyRun
	MarginPositions []MarginPosition
	MarginOrders    []MarginOrder
	MarginTrades    []MarginTrade
	Shares          []PortfolioShare
}

// RemapPortfolioSnapshot rewrites ownership to importerClientID.
// The file's main book (id == fileOwnerClientID) becomes importerClientID so the first-book convention holds.
func RemapPortfolioSnapshot(snap PortfolioSnapshot, fileOwnerClientID, importerClientID string) PortfolioSnapshot {
	fileOwnerClientID = strings.TrimSpace(fileOwnerClientID)
	importerClientID = strings.TrimSpace(importerClientID)
	oldBook := strings.TrimSpace(snap.Book.ID)
	if oldBook == "" {
		oldBook = strings.TrimSpace(snap.Book.ClientID)
	}
	newBook := oldBook
	if oldBook == "" || oldBook == fileOwnerClientID {
		newBook = importerClientID
	}
	if newBook == "" {
		newBook = importerClientID
	}
	return rebindPortfolioSnapshotBook(snap, oldBook, newBook, importerClientID)
}

// MustRemintImportedBookID is true for every extra book (id != owner). Extra
// ids must not occupy another tenant's first-book primary key (id == clientId),
// including unused UUID-shaped client ids that are not in portfolios yet.
// The importer's remapped main book is kept.
func MustRemintImportedBookID(bookID, ownerClientID string) bool {
	bookID = strings.TrimSpace(bookID)
	ownerClientID = strings.TrimSpace(ownerClientID)
	if bookID == "" || bookID == ownerClientID {
		return false
	}
	return true
}

// RebindPortfolioSnapshotBookID rewrites Book.ID and every child row keyed by the old book id.
func RebindPortfolioSnapshotBookID(snap PortfolioSnapshot, newBook string) PortfolioSnapshot {
	oldBook := strings.TrimSpace(snap.Book.ID)
	if oldBook == "" {
		oldBook = strings.TrimSpace(snap.Book.ClientID)
	}
	newBook = strings.TrimSpace(newBook)
	if newBook == "" || newBook == oldBook {
		return snap
	}
	owner := strings.TrimSpace(snap.Book.ClientID)
	return rebindPortfolioSnapshotBook(snap, oldBook, newBook, owner)
}

func rebindPortfolioSnapshotBook(snap PortfolioSnapshot, oldBook, newBook, ownerClientID string) PortfolioSnapshot {
	mapID := func(id string) string {
		if strings.TrimSpace(id) == oldBook {
			return newBook
		}
		return id
	}
	snap.Book.ID = newBook
	if ownerClientID != "" {
		snap.Book.ClientID = ownerClientID
	}
	for i := range snap.Positions {
		snap.Positions[i].ClientID = newBook
	}
	for i := range snap.Trades {
		snap.Trades[i].ClientID = newBook
	}
	for i := range snap.OpenOrders {
		snap.OpenOrders[i].ClientID = newBook
	}
	for i := range snap.Lots {
		snap.Lots[i].ClientID = newBook
	}
	for i := range snap.RecurringPlans {
		snap.RecurringPlans[i].ClientID = newBook
	}
	for i := range snap.RecurringRuns {
		snap.RecurringRuns[i].ClientID = newBook
	}
	for i := range snap.MarginPositions {
		snap.MarginPositions[i].ClientID = newBook
	}
	for i := range snap.MarginOrders {
		snap.MarginOrders[i].ClientID = newBook
	}
	for i := range snap.MarginTrades {
		snap.MarginTrades[i].ClientID = newBook
	}
	for i := range snap.Shares {
		snap.Shares[i].PortfolioID = mapID(snap.Shares[i].PortfolioID)
		if ownerClientID != "" {
			snap.Shares[i].OwnerClientID = ownerClientID
		}
	}
	return snap
}

// RekeyPortfolioSnapshot assigns new ids to trades, lots, orders, plans, and margin
// rows so two clients can share one database without primary-key collisions.
// Book id is left unchanged (already remapped for tenancy). newID must return unique ids.
func RekeyPortfolioSnapshot(snap PortfolioSnapshot, newID func() string) PortfolioSnapshot {
	if newID == nil {
		return snap
	}
	ids := map[string]string{}
	mapID := func(old string) string {
		old = strings.TrimSpace(old)
		if old == "" {
			return ""
		}
		if v, ok := ids[old]; ok {
			return v
		}
		n := newID()
		ids[old] = n
		return n
	}
	for i := range snap.Trades {
		snap.Trades[i].ID = mapID(snap.Trades[i].ID)
		snap.Trades[i].PendingOrderID = mapID(snap.Trades[i].PendingOrderID)
	}
	for i := range snap.OpenOrders {
		snap.OpenOrders[i].ID = mapID(snap.OpenOrders[i].ID)
		snap.OpenOrders[i].OCOGroupID = mapID(snap.OpenOrders[i].OCOGroupID)
		snap.OpenOrders[i].OCOPeerID = mapID(snap.OpenOrders[i].OCOPeerID)
		snap.OpenOrders[i].BracketID = mapID(snap.OpenOrders[i].BracketID)
		snap.OpenOrders[i].FillTradeID = mapID(snap.OpenOrders[i].FillTradeID)
	}
	for i := range snap.Lots {
		snap.Lots[i].ID = mapID(snap.Lots[i].ID)
		snap.Lots[i].SourceTradeID = mapID(snap.Lots[i].SourceTradeID)
	}
	for i := range snap.LotFills {
		snap.LotFills[i].ID = mapID(snap.LotFills[i].ID)
		snap.LotFills[i].TradeID = mapID(snap.LotFills[i].TradeID)
		snap.LotFills[i].LotID = mapID(snap.LotFills[i].LotID)
	}
	for i := range snap.RecurringPlans {
		snap.RecurringPlans[i].ID = mapID(snap.RecurringPlans[i].ID)
	}
	for i := range snap.RecurringRuns {
		snap.RecurringRuns[i].ID = mapID(snap.RecurringRuns[i].ID)
		snap.RecurringRuns[i].PlanID = mapID(snap.RecurringRuns[i].PlanID)
		snap.RecurringRuns[i].TradeID = mapID(snap.RecurringRuns[i].TradeID)
	}
	for i := range snap.MarginPositions {
		snap.MarginPositions[i].ID = mapID(snap.MarginPositions[i].ID)
	}
	for i := range snap.MarginOrders {
		snap.MarginOrders[i].ID = mapID(snap.MarginOrders[i].ID)
		snap.MarginOrders[i].PositionID = mapID(snap.MarginOrders[i].PositionID)
	}
	for i := range snap.MarginTrades {
		snap.MarginTrades[i].ID = mapID(snap.MarginTrades[i].ID)
		snap.MarginTrades[i].PositionID = mapID(snap.MarginTrades[i].PositionID)
	}
	return snap
}

// ValidatePortfolioSnapshot checks a remapped snapshot can be imported.
func ValidatePortfolioSnapshot(snap PortfolioSnapshot) error {
	if strings.TrimSpace(snap.Book.ID) == "" || strings.TrimSpace(snap.Book.ClientID) == "" {
		return fmt.Errorf("%w: portfolio id and owner are required", ErrInvalidArgument)
	}
	if _, err := ValidatePortfolioName(snap.Book.Name); err != nil {
		return err
	}
	cur := strings.ToUpper(strings.TrimSpace(snap.Book.Currency))
	if cur == "" {
		return fmt.Errorf("%w: currency is required", ErrInvalidArgument)
	}
	if snap.Book.StartingBalance < 0 || snap.Book.CashBalance < 0 {
		return fmt.Errorf("%w: balances cannot be negative", ErrInvalidArgument)
	}
	return nil
}

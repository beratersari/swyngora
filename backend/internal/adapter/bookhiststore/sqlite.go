package bookhiststore

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/adapter/sqliteutil"
	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"

	_ "modernc.org/sqlite"
)

// SQLite persists compact order-book samples.
type SQLite struct {
	db   *sql.DB
	mu   sync.Mutex
	path string
}

type payload struct {
	GroupSize float64                   `json:"g"`
	Bids      []domain.BookHistoryLevel `json:"b"`
	Asks      []domain.BookHistoryLevel `json:"a"`
	Walls     []domain.BookHistoryWall  `json:"w"`
}

// Open opens or creates the order-book history database.
func Open(path string) (*SQLite, error) {
	path = filepath.Clean(path)
	if path == "" || path == "." {
		return nil, fmt.Errorf("orderbook history sqlite path is required")
	}
	if dir := filepath.Dir(path); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("create orderbook history db dir: %w", err)
		}
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	dsn := "file:" + filepath.ToSlash(abs) + "?_pragma=busy_timeout(5000)&_pragma=foreign_keys(1)"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(4)
	db.SetMaxIdleConns(2)
	if _, err := db.Exec(`PRAGMA journal_mode=WAL`); err != nil {
		_ = db.Close()
		return nil, err
	}
	s := &SQLite{db: db, path: abs}
	if err := s.migrate(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return s, nil
}

// Path returns the absolute database path.
func (s *SQLite) Path() string { return s.path }

// Close closes the database.
func (s *SQLite) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

func (s *SQLite) migrate() error {
	const schema = `
CREATE TABLE IF NOT EXISTS book_snapshots (
	exchange        TEXT NOT NULL,
	symbol          TEXT NOT NULL,
	sampled_at_ms   INTEGER NOT NULL,
	mid             REAL NOT NULL DEFAULT 0,
	best_bid        REAL NOT NULL DEFAULT 0,
	best_ask        REAL NOT NULL DEFAULT 0,
	spread          REAL NOT NULL DEFAULT 0,
	spread_pct      REAL NOT NULL DEFAULT 0,
	group_size      REAL NOT NULL DEFAULT 0,
	bid_notional    REAL NOT NULL DEFAULT 0,
	ask_notional    REAL NOT NULL DEFAULT 0,
	imbalance       REAL NOT NULL DEFAULT 0,
	pressure        TEXT NOT NULL DEFAULT '',
	bid_walls       INTEGER NOT NULL DEFAULT 0,
	ask_walls       INTEGER NOT NULL DEFAULT 0,
	live            INTEGER NOT NULL DEFAULT 0,
	payload         TEXT NOT NULL DEFAULT '',
	PRIMARY KEY (exchange, symbol, sampled_at_ms)
);
CREATE INDEX IF NOT EXISTS idx_book_snapshots_lookup
	ON book_snapshots(exchange, symbol, sampled_at_ms);
CREATE TABLE IF NOT EXISTS book_flashes (
	exchange       TEXT NOT NULL,
	symbol         TEXT NOT NULL,
	side           TEXT NOT NULL,
	price          REAL NOT NULL,
	quantity       REAL NOT NULL DEFAULT 0,
	notional       REAL NOT NULL DEFAULT 0,
	seen_from_ms   INTEGER NOT NULL,
	seen_until_ms  INTEGER NOT NULL,
	PRIMARY KEY (exchange, symbol, side, price, seen_from_ms)
);
CREATE INDEX IF NOT EXISTS idx_book_flashes_lookup
	ON book_flashes(exchange, symbol, seen_from_ms, seen_until_ms);
CREATE TABLE IF NOT EXISTS book_moves (
	exchange       TEXT NOT NULL,
	symbol         TEXT NOT NULL,
	side           TEXT NOT NULL,
	price          REAL NOT NULL,
	from_notional  REAL NOT NULL DEFAULT 0,
	to_notional    REAL NOT NULL DEFAULT 0,
	peak_notional  REAL NOT NULL DEFAULT 0,
	change         TEXT NOT NULL DEFAULT '',
	seen_from_ms   INTEGER NOT NULL,
	seen_until_ms  INTEGER NOT NULL,
	PRIMARY KEY (exchange, symbol, side, price, seen_from_ms)
);
CREATE INDEX IF NOT EXISTS idx_book_moves_lookup
	ON book_moves(exchange, symbol, seen_from_ms, seen_until_ms);
`
	if _, err := s.db.Exec(schema); err != nil {
		return fmt.Errorf("book history schema: %w", err)
	}
	return sqliteutil.SetUserVersion(s.db, 1)
}

// InsertSnapshot stores one sample. Duplicate keys are ignored.
func (s *SQLite) InsertSnapshot(ctx context.Context, rec domain.BookHistorySnapshot) (bool, error) {
	if s == nil || s.db == nil {
		return false, fmt.Errorf("book history store is nil")
	}
	if rec.Exchange == "" || rec.Symbol == "" || rec.SampledAt.IsZero() {
		return false, fmt.Errorf("%w: incomplete book snapshot", domain.ErrInvalidArgument)
	}
	rec.Symbol = domain.NormalizeSymbol(rec.Exchange, rec.Symbol)
	blob, err := json.Marshal(payload{GroupSize: rec.GroupSize, Bids: rec.Bids, Asks: rec.Asks, Walls: rec.Walls})
	if err != nil {
		return false, err
	}
	live := 0
	if rec.Live {
		live = 1
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	res, err := s.db.ExecContext(ctx, `
INSERT OR IGNORE INTO book_snapshots(
	exchange, symbol, sampled_at_ms, mid, best_bid, best_ask, spread, spread_pct,
	group_size, bid_notional, ask_notional, imbalance, pressure, bid_walls, ask_walls, live, payload
) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		string(rec.Exchange), rec.Symbol, rec.SampledAt.UTC().UnixMilli(),
		rec.Mid, rec.BestBid, rec.BestAsk, rec.Spread, rec.SpreadPct, rec.GroupSize,
		rec.BidNotional, rec.AskNotional, rec.Imbalance, rec.Pressure,
		rec.BidWalls, rec.AskWalls, live, string(blob),
	)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

// EarliestBook is the oldest stored ladder for one venue and symbol.
func (s *SQLite) EarliestBook(ctx context.Context, exchange domain.Exchange, symbol string) (time.Time, bool, error) {
	if s == nil || s.db == nil {
		return time.Time{}, false, fmt.Errorf("book history store is nil")
	}
	symbol = domain.NormalizeSymbol(exchange, symbol)
	if exchange == "" || symbol == "" {
		return time.Time{}, false, fmt.Errorf("%w: incomplete book query", domain.ErrInvalidArgument)
	}
	var ms sql.NullInt64
	err := s.db.QueryRowContext(ctx, `
SELECT MIN(sampled_at_ms) FROM book_snapshots
WHERE exchange = ? AND symbol = ?`, string(exchange), symbol).Scan(&ms)
	if err != nil {
		return time.Time{}, false, err
	}
	if !ms.Valid || ms.Int64 <= 0 {
		return time.Time{}, false, nil
	}
	return time.UnixMilli(ms.Int64).UTC(), true, nil
}

// NearestAt returns the sample at or before at, else the next one after.
func (s *SQLite) NearestAt(ctx context.Context, exchange, symbol string, at time.Time) (*domain.BookHistorySnapshot, error) {
	if s == nil || s.db == nil {
		return nil, fmt.Errorf("book history store is nil")
	}
	ex := domain.ParseExchange(exchange)
	if ex == "" {
		ex = domain.DefaultExchange
	}
	symbol = domain.NormalizeSymbol(ex, symbol)
	if symbol == "" || at.IsZero() {
		return nil, fmt.Errorf("%w: symbol and time are required", domain.ErrInvalidArgument)
	}
	ms := at.UTC().UnixMilli()
	row := s.scanOne(ctx, `
SELECT exchange, symbol, sampled_at_ms, mid, best_bid, best_ask, spread, spread_pct,
	group_size, bid_notional, ask_notional, imbalance, pressure, bid_walls, ask_walls, live, payload
FROM book_snapshots
WHERE exchange = ? AND symbol = ? AND sampled_at_ms <= ?
ORDER BY sampled_at_ms DESC LIMIT 1`, string(ex), symbol, ms)
	if row != nil {
		return row, nil
	}
	return s.scanOne(ctx, `
SELECT exchange, symbol, sampled_at_ms, mid, best_bid, best_ask, spread, spread_pct,
	group_size, bid_notional, ask_notional, imbalance, pressure, bid_walls, ask_walls, live, payload
FROM book_snapshots
WHERE exchange = ? AND symbol = ? AND sampled_at_ms > ?
ORDER BY sampled_at_ms ASC LIMIT 1`, string(ex), symbol, ms), nil
}

// ListSnapshots returns newest-first samples in an optional window.
func (s *SQLite) ListSnapshots(ctx context.Context, q domain.BookHistoryQuery) ([]domain.BookHistorySnapshot, error) {
	if s == nil || s.db == nil {
		return nil, fmt.Errorf("book history store is nil")
	}
	ex := domain.ParseExchange(q.Exchange)
	if ex == "" {
		ex = domain.DefaultExchange
	}
	symbol := domain.NormalizeSymbol(ex, q.Symbol)
	if symbol == "" {
		return nil, fmt.Errorf("%w: symbol is required", domain.ErrInvalidArgument)
	}
	limit := domain.ParseBookHistoryLimit(q.Limit)
	args := []any{string(ex), symbol}
	where := `exchange = ? AND symbol = ?`
	if !q.From.IsZero() {
		where += ` AND sampled_at_ms >= ?`
		args = append(args, q.From.UTC().UnixMilli())
	}
	if !q.To.IsZero() {
		where += ` AND sampled_at_ms <= ?`
		args = append(args, q.To.UTC().UnixMilli())
	}
	args = append(args, limit)
	rows, err := s.db.QueryContext(ctx, `
SELECT exchange, symbol, sampled_at_ms, mid, best_bid, best_ask, spread, spread_pct,
	group_size, bid_notional, ask_notional, imbalance, pressure, bid_walls, ask_walls, live, payload
FROM book_snapshots WHERE `+where+` ORDER BY sampled_at_ms DESC LIMIT ?`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]domain.BookHistorySnapshot, 0, limit)
	for rows.Next() {
		rec, err := scanBookRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, rec)
	}
	return out, rows.Err()
}

// PurgeOlderThan deletes samples before cutoff.
func (s *SQLite) PurgeOlderThan(ctx context.Context, cutoff time.Time) (int, error) {
	if s == nil || s.db == nil || cutoff.IsZero() {
		return 0, nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	res, err := s.db.ExecContext(ctx, `DELETE FROM book_snapshots WHERE sampled_at_ms < ?`, cutoff.UTC().UnixMilli())
	if err != nil {
		return 0, err
	}
	n, _ := res.RowsAffected()
	res2, err := s.db.ExecContext(ctx, `DELETE FROM book_flashes WHERE seen_until_ms < ?`, cutoff.UTC().UnixMilli())
	if err != nil {
		return int(n), err
	}
	n2, _ := res2.RowsAffected()
	res3, err := s.db.ExecContext(ctx, `DELETE FROM book_moves WHERE seen_until_ms < ?`, cutoff.UTC().UnixMilli())
	if err != nil {
		return int(n + n2), err
	}
	n3, _ := res3.RowsAffected()
	return int(n + n2 + n3), nil
}

// InsertFlash stores one brief wall. Duplicate keys are ignored.
func (s *SQLite) InsertFlash(ctx context.Context, rec domain.BookHistoryFlash) (bool, error) {
	if s == nil || s.db == nil {
		return false, fmt.Errorf("book history store is nil")
	}
	if rec.Exchange == "" || rec.Symbol == "" || rec.SeenFrom.IsZero() || rec.Price <= 0 {
		return false, fmt.Errorf("%w: incomplete book flash", domain.ErrInvalidArgument)
	}
	rec.Symbol = domain.NormalizeSymbol(rec.Exchange, rec.Symbol)
	s.mu.Lock()
	defer s.mu.Unlock()
	res, err := s.db.ExecContext(ctx, `
INSERT OR IGNORE INTO book_flashes(
	exchange, symbol, side, price, quantity, notional, seen_from_ms, seen_until_ms
) VALUES (?,?,?,?,?,?,?,?)`,
		string(rec.Exchange), rec.Symbol, rec.Side, rec.Price, rec.Quantity, rec.Notional,
		rec.SeenFrom.UTC().UnixMilli(), rec.SeenUntil.UTC().UnixMilli(),
	)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

// ListFlashes returns brief walls that overlap [from, to).
func (s *SQLite) ListFlashes(ctx context.Context, exchange, symbol string, from, to time.Time) ([]domain.BookHistoryFlash, error) {
	if s == nil || s.db == nil {
		return nil, fmt.Errorf("book history store is nil")
	}
	ex := domain.ParseExchange(exchange)
	if ex == "" {
		ex = domain.DefaultExchange
	}
	symbol = domain.NormalizeSymbol(ex, symbol)
	if symbol == "" || from.IsZero() || to.IsZero() {
		return nil, fmt.Errorf("%w: symbol and window are required", domain.ErrInvalidArgument)
	}
	rows, err := s.db.QueryContext(ctx, `
SELECT side, price, quantity, notional, seen_from_ms, seen_until_ms
FROM book_flashes
WHERE exchange = ? AND symbol = ? AND seen_until_ms >= ? AND seen_from_ms < ?
ORDER BY seen_from_ms ASC`, string(ex), symbol, from.UTC().UnixMilli(), to.UTC().UnixMilli())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.BookHistoryFlash
	for rows.Next() {
		var side string
		var price, qty, notional float64
		var fromMs, untilMs int64
		if err := rows.Scan(&side, &price, &qty, &notional, &fromMs, &untilMs); err != nil {
			return nil, err
		}
		seenFrom := time.UnixMilli(fromMs).UTC()
		seenUntil := time.UnixMilli(untilMs).UTC()
		out = append(out, domain.BookHistoryFlash{
			Exchange: ex, Symbol: symbol, Side: side,
			Price: price, Quantity: qty, Notional: notional,
			SeenFrom: seenFrom, SeenUntil: seenUntil,
			Duration: seenUntil.Sub(seenFrom), Brief: true,
		})
	}
	return out, rows.Err()
}

// InsertMove stores one between-minute size change. Duplicate keys are ignored.
func (s *SQLite) InsertMove(ctx context.Context, rec domain.BookHistoryMove) (bool, error) {
	if s == nil || s.db == nil {
		return false, fmt.Errorf("book history store is nil")
	}
	if rec.Exchange == "" || rec.Symbol == "" || rec.SeenFrom.IsZero() || rec.Price <= 0 {
		return false, fmt.Errorf("%w: incomplete book move", domain.ErrInvalidArgument)
	}
	rec.Symbol = domain.NormalizeSymbol(rec.Exchange, rec.Symbol)
	s.mu.Lock()
	defer s.mu.Unlock()
	res, err := s.db.ExecContext(ctx, `
INSERT OR IGNORE INTO book_moves(
	exchange, symbol, side, price, from_notional, to_notional, peak_notional, change, seen_from_ms, seen_until_ms
) VALUES (?,?,?,?,?,?,?,?,?,?)`,
		string(rec.Exchange), rec.Symbol, rec.Side, rec.Price, rec.FromNotional, rec.ToNotional, rec.PeakNotional, rec.Change,
		rec.SeenFrom.UTC().UnixMilli(), rec.SeenUntil.UTC().UnixMilli(),
	)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

// ListMoves returns size changes that overlap [from, to).
func (s *SQLite) ListMoves(ctx context.Context, exchange, symbol string, from, to time.Time) ([]domain.BookHistoryMove, error) {
	if s == nil || s.db == nil {
		return nil, fmt.Errorf("book history store is nil")
	}
	ex := domain.ParseExchange(exchange)
	if ex == "" {
		ex = domain.DefaultExchange
	}
	symbol = domain.NormalizeSymbol(ex, symbol)
	if symbol == "" || from.IsZero() || to.IsZero() {
		return nil, fmt.Errorf("%w: symbol and window are required", domain.ErrInvalidArgument)
	}
	rows, err := s.db.QueryContext(ctx, `
SELECT side, price, from_notional, to_notional, peak_notional, change, seen_from_ms, seen_until_ms
FROM book_moves
WHERE exchange = ? AND symbol = ? AND seen_until_ms >= ? AND seen_from_ms < ?
ORDER BY seen_from_ms ASC`, string(ex), symbol, from.UTC().UnixMilli(), to.UTC().UnixMilli())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.BookHistoryMove
	for rows.Next() {
		var side, change string
		var price, fromN, toN, peak float64
		var fromMs, untilMs int64
		if err := rows.Scan(&side, &price, &fromN, &toN, &peak, &change, &fromMs, &untilMs); err != nil {
			return nil, err
		}
		seenFrom := time.UnixMilli(fromMs).UTC()
		seenUntil := time.UnixMilli(untilMs).UTC()
		out = append(out, domain.BookHistoryMove{
			Exchange: ex, Symbol: symbol, Side: side, Price: price,
			FromNotional: fromN, ToNotional: toN, PeakNotional: peak, Change: change,
			SeenFrom: seenFrom, SeenUntil: seenUntil, Duration: seenUntil.Sub(seenFrom),
		})
	}
	return out, rows.Err()
}

type rowScanner interface {
	Scan(dest ...any) error
}

func (s *SQLite) scanOne(ctx context.Context, q string, args ...any) *domain.BookHistorySnapshot {
	row := s.db.QueryRowContext(ctx, q, args...)
	rec, err := scanBookRow(row)
	if err != nil {
		return nil
	}
	return &rec
}

func scanBookRow(sc rowScanner) (domain.BookHistorySnapshot, error) {
	var (
		ex, symbol, pressure, blob string
		ms                         int64
		live                       int
		rec                        domain.BookHistorySnapshot
	)
	err := sc.Scan(
		&ex, &symbol, &ms, &rec.Mid, &rec.BestBid, &rec.BestAsk, &rec.Spread, &rec.SpreadPct,
		&rec.GroupSize, &rec.BidNotional, &rec.AskNotional, &rec.Imbalance, &pressure,
		&rec.BidWalls, &rec.AskWalls, &live, &blob,
	)
	if err != nil {
		return domain.BookHistorySnapshot{}, err
	}
	rec.Exchange = domain.Exchange(ex)
	rec.Symbol = symbol
	rec.SampledAt = time.UnixMilli(ms).UTC()
	rec.Pressure = pressure
	rec.Live = live == 1
	rec.Complete = true
	if blob != "" {
		var p payload
		if err := json.Unmarshal([]byte(blob), &p); err == nil {
			if rec.GroupSize <= 0 {
				rec.GroupSize = p.GroupSize
			}
			rec.Bids, rec.Asks, rec.Walls = p.Bids, p.Asks, p.Walls
		}
	}
	return rec, nil
}

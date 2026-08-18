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
	return int(n), nil
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

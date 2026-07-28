package watchliststore

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"

	_ "modernc.org/sqlite"
)

// SQLite is a file-backed watchlist store implementing domain.WatchlistPort.
// Safe for concurrent use. Data survives process restarts when using a file path.
type SQLite struct {
	db         *sql.DB
	mu         sync.Mutex // serializes mutations (max-items / max-clients checks + writes)
	maxClients int
	maxItems   int
	path       string
}

// OpenSQLite opens (or creates) a SQLite database at path and migrates schema.
// Parent directories are created when needed. path must be non-empty.
func OpenSQLite(path string) (*SQLite, error) {
	return OpenSQLiteWithMaxClients(path, DefaultMaxClients)
}

// OpenSQLiteWithMaxClients is like OpenSQLite with an explicit client cap (0 = unlimited).
func OpenSQLiteWithMaxClients(path string, maxClients int) (*SQLite, error) {
	path = filepath.Clean(path)
	if path == "" || path == "." {
		return nil, fmt.Errorf("watchlist sqlite path is required")
	}
	if dir := filepath.Dir(path); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("create watchlist db dir: %w", err)
		}
	}

	// Absolute path so reopening after chdir still hits the same file.
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("watchlist sqlite abs path: %w", err)
	}

	// modernc URI form; enable FK + busy wait. WAL is set via PRAGMA after open.
	dsn := "file:" + filepath.ToSlash(abs) + "?_pragma=busy_timeout(5000)&_pragma=foreign_keys(1)"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open watchlist sqlite: %w", err)
	}
	// Bound connections; mutations are also serialized by mu.
	db.SetMaxOpenConns(4)
	db.SetMaxIdleConns(2)
	db.SetConnMaxLifetime(0)

	if _, err := db.Exec(`PRAGMA journal_mode=WAL`); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("watchlist sqlite wal: %w", err)
	}
	if _, err := db.Exec(`PRAGMA foreign_keys=ON`); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("watchlist sqlite foreign_keys: %w", err)
	}

	s := &SQLite{
		db:         db,
		maxClients: maxClients,
		maxItems:   domain.MaxWatchlistItems,
		path:       abs,
	}
	if err := s.migrate(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return s, nil
}

func (s *SQLite) migrate() error {
	const schema = `
CREATE TABLE IF NOT EXISTS watchlist_meta (
	client_id  TEXT PRIMARY KEY NOT NULL,
	updated_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS watchlist_items (
	client_id TEXT NOT NULL,
	exchange  TEXT NOT NULL,
	symbol    TEXT NOT NULL,
	note      TEXT NOT NULL DEFAULT '',
	added_at  TEXT NOT NULL,
	PRIMARY KEY (client_id, exchange, symbol),
	FOREIGN KEY (client_id) REFERENCES watchlist_meta(client_id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_watchlist_items_client ON watchlist_items(client_id);
`
	if _, err := s.db.Exec(schema); err != nil {
		return fmt.Errorf("watchlist sqlite migrate: %w", err)
	}
	return nil
}

// Path returns the absolute database file path.
func (s *SQLite) Path() string { return s.path }

// Close releases the database handle.
func (s *SQLite) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

// Get returns a copy of the watchlist (empty items if unknown client).
func (s *SQLite) Get(ctx context.Context, clientID string) (*domain.Watchlist, error) {
	items, updated, ok, err := s.load(ctx, s.db, clientID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return &domain.Watchlist{
			ClientID: clientID,
			Items:    []domain.WatchlistItem{},
			Updated:  time.Now().UTC(),
		}, nil
	}
	return &domain.Watchlist{
		ClientID: clientID,
		Items:    items,
		Updated:  updated,
	}, nil
}

// Set replaces the list. Rejects when len(items) > maxItems.
func (s *SQLite) Set(ctx context.Context, clientID string, items []domain.WatchlistItem) (*domain.Watchlist, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.maxItems > 0 && len(items) > s.maxItems {
		return nil, fmt.Errorf("%w: watchlist max %d items", domain.ErrInvalidArgument, s.maxItems)
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("watchlist sqlite begin: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if err := s.ensureClientTx(ctx, tx, clientID); err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	// Meta first so FK on items succeeds for brand-new clients.
	if err := upsertMeta(ctx, tx, clientID, now); err != nil {
		return nil, err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM watchlist_items WHERE client_id = ?`, clientID); err != nil {
		return nil, fmt.Errorf("watchlist sqlite clear items: %w", err)
	}
	for _, it := range items {
		if err := insertItem(ctx, tx, clientID, it); err != nil {
			return nil, err
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("watchlist sqlite commit: %w", err)
	}

	outItems := append([]domain.WatchlistItem(nil), items...)
	return &domain.Watchlist{ClientID: clientID, Items: outItems, Updated: now}, nil
}

// Add upserts one item. Enforces maxItems under the same lock (no TOCTOU).
func (s *SQLite) Add(ctx context.Context, clientID string, item domain.WatchlistItem) (*domain.Watchlist, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("watchlist sqlite begin: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if err := s.ensureClientTx(ctx, tx, clientID); err != nil {
		return nil, err
	}

	items, _, _, err := s.load(ctx, tx, clientID)
	if err != nil {
		return nil, err
	}

	found := false
	for i, it := range items {
		if it.Exchange == item.Exchange && it.Symbol == item.Symbol {
			if item.AddedAt.IsZero() {
				item.AddedAt = it.AddedAt
			}
			items[i] = item
			found = true
			break
		}
	}
	if !found {
		if s.maxItems > 0 && len(items) >= s.maxItems {
			return nil, fmt.Errorf("%w: watchlist max %d items", domain.ErrInvalidArgument, s.maxItems)
		}
		if item.AddedAt.IsZero() {
			item.AddedAt = time.Now().UTC()
		}
		items = append(items, item)
	}

	now := time.Now().UTC()
	if err := upsertMeta(ctx, tx, clientID, now); err != nil {
		return nil, err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM watchlist_items WHERE client_id = ?`, clientID); err != nil {
		return nil, fmt.Errorf("watchlist sqlite clear items: %w", err)
	}
	for _, it := range items {
		if err := insertItem(ctx, tx, clientID, it); err != nil {
			return nil, err
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("watchlist sqlite commit: %w", err)
	}

	return &domain.Watchlist{
		ClientID: clientID,
		Items:    append([]domain.WatchlistItem(nil), items...),
		Updated:  now,
	}, nil
}

// Remove deletes one item.
func (s *SQLite) Remove(ctx context.Context, clientID string, exchange domain.Exchange, symbol string) (*domain.Watchlist, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("watchlist sqlite begin: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	items, _, ok, err := s.load(ctx, tx, clientID)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	if !ok {
		return &domain.Watchlist{ClientID: clientID, Items: []domain.WatchlistItem{}, Updated: now}, nil
	}

	next := make([]domain.WatchlistItem, 0, len(items))
	for _, it := range items {
		if it.Exchange == exchange && it.Symbol == symbol {
			continue
		}
		next = append(next, it)
	}

	if _, err := tx.ExecContext(ctx, `
		DELETE FROM watchlist_items WHERE client_id = ? AND exchange = ? AND symbol = ?
	`, clientID, string(exchange), symbol); err != nil {
		return nil, fmt.Errorf("watchlist sqlite delete item: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `
		UPDATE watchlist_meta SET updated_at = ? WHERE client_id = ?
	`, now.Format(time.RFC3339Nano), clientID); err != nil {
		return nil, fmt.Errorf("watchlist sqlite update meta: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("watchlist sqlite commit: %w", err)
	}

	return &domain.Watchlist{ClientID: clientID, Items: next, Updated: now}, nil
}

// queryer is satisfied by *sql.DB and *sql.Tx.
type queryer interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

func (s *SQLite) ensureClientTx(ctx context.Context, q queryer, clientID string) error {
	var exists int
	err := q.QueryRowContext(ctx, `SELECT 1 FROM watchlist_meta WHERE client_id = ?`, clientID).Scan(&exists)
	if err == nil {
		return nil
	}
	if err != sql.ErrNoRows {
		return fmt.Errorf("watchlist sqlite client lookup: %w", err)
	}
	if s.maxClients <= 0 {
		return nil
	}
	var n int
	if err := q.QueryRowContext(ctx, `SELECT COUNT(*) FROM watchlist_meta`).Scan(&n); err != nil {
		return fmt.Errorf("watchlist sqlite client count: %w", err)
	}
	if n >= s.maxClients {
		return fmt.Errorf("%w: watchlist client capacity reached", domain.ErrInvalidArgument)
	}
	return nil
}

func (s *SQLite) load(ctx context.Context, q queryer, clientID string) (items []domain.WatchlistItem, updated time.Time, ok bool, err error) {
	var updatedRaw string
	err = q.QueryRowContext(ctx, `SELECT updated_at FROM watchlist_meta WHERE client_id = ?`, clientID).Scan(&updatedRaw)
	if err == sql.ErrNoRows {
		return nil, time.Time{}, false, nil
	}
	if err != nil {
		return nil, time.Time{}, false, fmt.Errorf("watchlist sqlite load meta: %w", err)
	}
	updated, err = time.Parse(time.RFC3339Nano, updatedRaw)
	if err != nil {
		// Tolerate RFC3339 without nanos.
		updated, err = time.Parse(time.RFC3339, updatedRaw)
		if err != nil {
			updated = time.Now().UTC()
		}
	}

	rows, err := q.QueryContext(ctx, `
		SELECT exchange, symbol, note, added_at
		FROM watchlist_items
		WHERE client_id = ?
		ORDER BY added_at ASC, exchange ASC, symbol ASC
	`, clientID)
	if err != nil {
		return nil, time.Time{}, false, fmt.Errorf("watchlist sqlite load items: %w", err)
	}
	defer rows.Close()

	items = make([]domain.WatchlistItem, 0)
	for rows.Next() {
		var ex, sym, note, addedRaw string
		if err := rows.Scan(&ex, &sym, &note, &addedRaw); err != nil {
			return nil, time.Time{}, false, fmt.Errorf("watchlist sqlite scan item: %w", err)
		}
		added, perr := time.Parse(time.RFC3339Nano, addedRaw)
		if perr != nil {
			added, perr = time.Parse(time.RFC3339, addedRaw)
			if perr != nil {
				added = time.Time{}
			}
		}
		items = append(items, domain.WatchlistItem{
			Exchange: domain.Exchange(ex),
			Symbol:   sym,
			Note:     note,
			AddedAt:  added.UTC(),
		})
	}
	if err := rows.Err(); err != nil {
		return nil, time.Time{}, false, fmt.Errorf("watchlist sqlite rows: %w", err)
	}
	return items, updated.UTC(), true, nil
}

func upsertMeta(ctx context.Context, q queryer, clientID string, now time.Time) error {
	_, err := q.ExecContext(ctx, `
		INSERT INTO watchlist_meta (client_id, updated_at) VALUES (?, ?)
		ON CONFLICT(client_id) DO UPDATE SET updated_at = excluded.updated_at
	`, clientID, now.UTC().Format(time.RFC3339Nano))
	if err != nil {
		return fmt.Errorf("watchlist sqlite upsert meta: %w", err)
	}
	return nil
}

func insertItem(ctx context.Context, q queryer, clientID string, it domain.WatchlistItem) error {
	added := it.AddedAt
	if added.IsZero() {
		added = time.Now().UTC()
	}
	_, err := q.ExecContext(ctx, `
		INSERT INTO watchlist_items (client_id, exchange, symbol, note, added_at)
		VALUES (?, ?, ?, ?, ?)
	`, clientID, string(it.Exchange), it.Symbol, it.Note, added.UTC().Format(time.RFC3339Nano))
	if err != nil {
		return fmt.Errorf("watchlist sqlite insert item: %w", err)
	}
	return nil
}
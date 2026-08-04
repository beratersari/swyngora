package watchliststore

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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
	updated_at TEXT NOT NULL,
	version    INTEGER NOT NULL DEFAULT 0
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

CREATE TABLE IF NOT EXISTS watchlist_shares (
	owner_client_id   TEXT NOT NULL,
	grantee_client_id TEXT NOT NULL,
	role              TEXT NOT NULL,
	created_at        TEXT NOT NULL,
	updated_at        TEXT NOT NULL,
	PRIMARY KEY (owner_client_id, grantee_client_id)
);
CREATE INDEX IF NOT EXISTS idx_watchlist_shares_grantee ON watchlist_shares(grantee_client_id);
CREATE INDEX IF NOT EXISTS idx_watchlist_shares_owner ON watchlist_shares(owner_client_id);

CREATE TABLE IF NOT EXISTS watchlist_audit (
	id              TEXT PRIMARY KEY NOT NULL,
	owner_client_id TEXT NOT NULL,
	actor_client_id TEXT NOT NULL,
	action          TEXT NOT NULL,
	exchange        TEXT NOT NULL DEFAULT '',
	symbol          TEXT NOT NULL DEFAULT '',
	detail          TEXT NOT NULL DEFAULT '',
	created_at      TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_watchlist_audit_owner ON watchlist_audit(owner_client_id, created_at DESC);
`
	if _, err := s.db.Exec(schema); err != nil {
		return fmt.Errorf("watchlist sqlite migrate: %w", err)
	}
	// Older DBs created before version column: add it if missing.
	if !s.columnExists("watchlist_meta", "version") {
		if _, err := s.db.Exec(`ALTER TABLE watchlist_meta ADD COLUMN version INTEGER NOT NULL DEFAULT 0`); err != nil {
			return fmt.Errorf("watchlist sqlite migrate version column: %w", err)
		}
	}
	return nil
}

func (s *SQLite) columnExists(table, col string) bool {
	rows, err := s.db.Query(`PRAGMA table_info(` + table + `)`)
	if err != nil {
		return false
	}
	defer rows.Close()
	for rows.Next() {
		var cid int
		var name, ctype string
		var notnull, pk int
		var dflt any
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk); err != nil {
			return false
		}
		if name == col {
			return true
		}
	}
	return false
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
	items, updated, version, ok, err := s.load(ctx, s.db, clientID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return &domain.Watchlist{
			ClientID: clientID,
			Items:    []domain.WatchlistItem{},
			Updated:  time.Now().UTC(),
			Version:  0,
		}, nil
	}
	return &domain.Watchlist{
		ClientID: clientID,
		Items:    items,
		Updated:  updated,
		Version:  version,
	}, nil
}

func versionMismatch(clientID string, items []domain.WatchlistItem, updated time.Time, version int64) error {
	return &domain.WatchlistVersionMismatch{Current: &domain.Watchlist{
		ClientID: clientID, Items: append([]domain.WatchlistItem(nil), items...),
		Updated: updated, Version: version,
	}}
}

// Set replaces the list. Rejects when len(items) > maxItems.
func (s *SQLite) Set(ctx context.Context, clientID string, items []domain.WatchlistItem, expectedVersion int64) (*domain.Watchlist, error) {
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

	curItems, curUpdated, curVer, ok, err := s.load(ctx, tx, clientID)
	if err != nil {
		return nil, err
	}
	if !ok {
		curVer = 0
	}
	if expectedVersion >= 0 && curVer != expectedVersion {
		return nil, versionMismatch(clientID, curItems, curUpdated, curVer)
	}

	now := time.Now().UTC()
	newVer := curVer + 1
	if err := upsertMetaVersion(ctx, tx, clientID, now, newVer); err != nil {
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
	return &domain.Watchlist{ClientID: clientID, Items: outItems, Updated: now, Version: newVer}, nil
}

// Add upserts one item. Enforces maxItems under the same lock (no TOCTOU).
func (s *SQLite) Add(ctx context.Context, clientID string, item domain.WatchlistItem, expectedVersion int64) (*domain.Watchlist, error) {
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

	items, curUpdated, curVer, ok, err := s.load(ctx, tx, clientID)
	if err != nil {
		return nil, err
	}
	if !ok {
		curVer = 0
	}
	if expectedVersion >= 0 && curVer != expectedVersion {
		return nil, versionMismatch(clientID, items, curUpdated, curVer)
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
	newVer := curVer + 1
	if err := upsertMetaVersion(ctx, tx, clientID, now, newVer); err != nil {
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
		Version:  newVer,
	}, nil
}

// Remove deletes one item.
func (s *SQLite) Remove(ctx context.Context, clientID string, exchange domain.Exchange, symbol string, expectedVersion int64) (*domain.Watchlist, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("watchlist sqlite begin: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	items, curUpdated, curVer, ok, err := s.load(ctx, tx, clientID)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	if !ok {
		if expectedVersion >= 0 && expectedVersion != 0 {
			return nil, versionMismatch(clientID, nil, now, 0)
		}
		return &domain.Watchlist{ClientID: clientID, Items: []domain.WatchlistItem{}, Updated: now, Version: 0}, nil
	}
	if expectedVersion >= 0 && curVer != expectedVersion {
		return nil, versionMismatch(clientID, items, curUpdated, curVer)
	}

	next := make([]domain.WatchlistItem, 0, len(items))
	for _, it := range items {
		if it.Exchange == exchange && it.Symbol == symbol {
			continue
		}
		next = append(next, it)
	}

	newVer := curVer + 1
	if _, err := tx.ExecContext(ctx, `
		DELETE FROM watchlist_items WHERE client_id = ? AND exchange = ? AND symbol = ?
	`, clientID, string(exchange), symbol); err != nil {
		return nil, fmt.Errorf("watchlist sqlite delete item: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `
		UPDATE watchlist_meta SET updated_at = ?, version = ? WHERE client_id = ?
	`, now.Format(time.RFC3339Nano), newVer, clientID); err != nil {
		return nil, fmt.Errorf("watchlist sqlite update meta: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("watchlist sqlite commit: %w", err)
	}

	return &domain.Watchlist{ClientID: clientID, Items: next, Updated: now, Version: newVer}, nil
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

func (s *SQLite) load(ctx context.Context, q queryer, clientID string) (items []domain.WatchlistItem, updated time.Time, version int64, ok bool, err error) {
	var updatedRaw string
	err = q.QueryRowContext(ctx, `SELECT updated_at, version FROM watchlist_meta WHERE client_id = ?`, clientID).Scan(&updatedRaw, &version)
	if err == sql.ErrNoRows {
		return nil, time.Time{}, 0, false, nil
	}
	if err != nil {
		return nil, time.Time{}, 0, false, fmt.Errorf("watchlist sqlite load meta: %w", err)
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
		return nil, time.Time{}, 0, false, fmt.Errorf("watchlist sqlite load items: %w", err)
	}
	defer rows.Close()

	items = make([]domain.WatchlistItem, 0)
	for rows.Next() {
		var ex, sym, note, addedRaw string
		if err := rows.Scan(&ex, &sym, &note, &addedRaw); err != nil {
			return nil, time.Time{}, 0, false, fmt.Errorf("watchlist sqlite scan item: %w", err)
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
		return nil, time.Time{}, 0, false, fmt.Errorf("watchlist sqlite rows: %w", err)
	}
	return items, updated.UTC(), version, true, nil
}

func upsertMeta(ctx context.Context, q queryer, clientID string, now time.Time) error {
	// Preserve version on simple timestamp touch if any callers remain.
	_, err := q.ExecContext(ctx, `
		INSERT INTO watchlist_meta (client_id, updated_at, version) VALUES (?, ?, 0)
		ON CONFLICT(client_id) DO UPDATE SET updated_at = excluded.updated_at
	`, clientID, now.UTC().Format(time.RFC3339Nano))
	return err
}

func upsertMetaVersion(ctx context.Context, q queryer, clientID string, now time.Time, version int64) error {
	_, err := q.ExecContext(ctx, `
		INSERT INTO watchlist_meta (client_id, updated_at, version) VALUES (?, ?, ?)
		ON CONFLICT(client_id) DO UPDATE SET updated_at = excluded.updated_at, version = excluded.version
	`, clientID, now.UTC().Format(time.RFC3339Nano), version)
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

// PurgeClient deletes list, items, shares, and audit for clientID.
func (s *SQLite) PurgeClient(ctx context.Context, clientID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx, `DELETE FROM watchlist_items WHERE client_id = ?`, clientID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM watchlist_meta WHERE client_id = ?`, clientID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `
		DELETE FROM watchlist_shares WHERE owner_client_id = ? OR grantee_client_id = ?
	`, clientID, clientID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `
		DELETE FROM watchlist_audit WHERE owner_client_id = ? OR actor_client_id = ?
	`, clientID, clientID); err != nil {
		return err
	}
	return tx.Commit()
}

// CreateShare inserts a share; fails if the pair already exists.
func (s *SQLite) CreateShare(ctx context.Context, share domain.WatchlistShare) (*domain.WatchlistShare, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if share.CreatedAt.IsZero() {
		share.CreatedAt = time.Now().UTC()
	}
	if share.UpdatedAt.IsZero() {
		share.UpdatedAt = share.CreatedAt
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO watchlist_shares (owner_client_id, grantee_client_id, role, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?)
	`, share.OwnerClientID, share.GranteeClientID, string(share.Role),
		share.CreatedAt.UTC().Format(time.RFC3339Nano), share.UpdatedAt.UTC().Format(time.RFC3339Nano))
	if err != nil {
		if isUniqueConstraint(err) {
			return nil, fmt.Errorf("%w: watchlist already shared with this user", domain.ErrInvalidArgument)
		}
		return nil, fmt.Errorf("watchlist share create: %w", err)
	}
	cp := share
	return &cp, nil
}

func isUniqueConstraint(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "unique") || strings.Contains(msg, "constraint")
}

// UpdateShareRole updates role for an existing share.
func (s *SQLite) UpdateShareRole(ctx context.Context, ownerClientID, granteeClientID string, role domain.WatchlistShareRole, at time.Time) (*domain.WatchlistShare, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if at.IsZero() {
		at = time.Now().UTC()
	}
	res, err := s.db.ExecContext(ctx, `
		UPDATE watchlist_shares SET role = ?, updated_at = ?
		WHERE owner_client_id = ? AND grantee_client_id = ?
	`, string(role), at.UTC().Format(time.RFC3339Nano), ownerClientID, granteeClientID)
	if err != nil {
		return nil, err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return nil, domain.ErrNotFound
	}
	return s.GetShare(ctx, ownerClientID, granteeClientID)
}

// GetShare returns one share or ErrNotFound.
func (s *SQLite) GetShare(ctx context.Context, ownerClientID, granteeClientID string) (*domain.WatchlistShare, error) {
	var sh domain.WatchlistShare
	var role, cAt, uAt string
	err := s.db.QueryRowContext(ctx, `
		SELECT owner_client_id, grantee_client_id, role, created_at, updated_at
		FROM watchlist_shares WHERE owner_client_id = ? AND grantee_client_id = ?
	`, ownerClientID, granteeClientID).Scan(&sh.OwnerClientID, &sh.GranteeClientID, &role, &cAt, &uAt)
	if err == sql.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	sh.Role = domain.WatchlistShareRole(role)
	sh.CreatedAt = parseWLTime(cAt)
	sh.UpdatedAt = parseWLTime(uAt)
	return &sh, nil
}

// ListSharesByOwner lists shares granted by owner.
func (s *SQLite) ListSharesByOwner(ctx context.Context, ownerClientID string) ([]domain.WatchlistShare, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT owner_client_id, grantee_client_id, role, created_at, updated_at
		FROM watchlist_shares WHERE owner_client_id = ?
		ORDER BY created_at ASC
	`, ownerClientID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanShares(rows)
}

// ListSharesForGrantee lists lists shared with grantee.
func (s *SQLite) ListSharesForGrantee(ctx context.Context, granteeClientID string) ([]domain.WatchlistShare, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT owner_client_id, grantee_client_id, role, created_at, updated_at
		FROM watchlist_shares WHERE grantee_client_id = ?
		ORDER BY created_at ASC
	`, granteeClientID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanShares(rows)
}

// DeleteShare revokes access.
func (s *SQLite) DeleteShare(ctx context.Context, ownerClientID, granteeClientID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	res, err := s.db.ExecContext(ctx, `
		DELETE FROM watchlist_shares WHERE owner_client_id = ? AND grantee_client_id = ?
	`, ownerClientID, granteeClientID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return domain.ErrNotFound
	}
	return nil
}

// CountSharesByOwner counts shares for owner.
func (s *SQLite) CountSharesByOwner(ctx context.Context, ownerClientID string) (int, error) {
	var n int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM watchlist_shares WHERE owner_client_id = ?`, ownerClientID).Scan(&n)
	return n, err
}

// AppendAudit writes an audit event.
func (s *SQLite) AppendAudit(ctx context.Context, ev domain.WatchlistAuditEvent) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if ev.CreatedAt.IsZero() {
		ev.CreatedAt = time.Now().UTC()
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO watchlist_audit (id, owner_client_id, actor_client_id, action, exchange, symbol, detail, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, ev.ID, ev.OwnerClientID, ev.ActorClientID, string(ev.Action), ev.Exchange, ev.Symbol, ev.Detail,
		ev.CreatedAt.UTC().Format(time.RFC3339Nano))
	return err
}

// ListAudit returns newest-first audit events for an owner list.
func (s *SQLite) ListAudit(ctx context.Context, ownerClientID string, limit, offset int) ([]domain.WatchlistAuditEvent, error) {
	if limit <= 0 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, owner_client_id, actor_client_id, action, exchange, symbol, detail, created_at
		FROM watchlist_audit WHERE owner_client_id = ?
		ORDER BY created_at DESC LIMIT ? OFFSET ?
	`, ownerClientID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]domain.WatchlistAuditEvent, 0)
	for rows.Next() {
		var ev domain.WatchlistAuditEvent
		var action, cAt string
		if err := rows.Scan(&ev.ID, &ev.OwnerClientID, &ev.ActorClientID, &action, &ev.Exchange, &ev.Symbol, &ev.Detail, &cAt); err != nil {
			return nil, err
		}
		ev.Action = domain.WatchlistAuditAction(action)
		ev.CreatedAt = parseWLTime(cAt)
		out = append(out, ev)
	}
	return out, rows.Err()
}

func scanShares(rows *sql.Rows) ([]domain.WatchlistShare, error) {
	out := make([]domain.WatchlistShare, 0)
	for rows.Next() {
		var sh domain.WatchlistShare
		var role, cAt, uAt string
		if err := rows.Scan(&sh.OwnerClientID, &sh.GranteeClientID, &role, &cAt, &uAt); err != nil {
			return nil, err
		}
		sh.Role = domain.WatchlistShareRole(role)
		sh.CreatedAt = parseWLTime(cAt)
		sh.UpdatedAt = parseWLTime(uAt)
		out = append(out, sh)
	}
	return out, rows.Err()
}

func parseWLTime(raw string) time.Time {
	t, err := time.Parse(time.RFC3339Nano, raw)
	if err != nil {
		t, err = time.Parse(time.RFC3339, raw)
		if err != nil {
			return time.Time{}
		}
	}
	return t.UTC()
}
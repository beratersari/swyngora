package importstore

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"

	_ "modernc.org/sqlite"
)

// SQLite persists import job metadata.
type SQLite struct {
	db   *sql.DB
	mu   sync.Mutex
	path string
}

// Open opens or creates the import jobs database.
func Open(path string) (*SQLite, error) {
	path = filepath.Clean(path)
	if path == "" || path == "." {
		return nil, fmt.Errorf("import sqlite path is required")
	}
	if dir := filepath.Dir(path); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("create import db dir: %w", err)
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

func (s *SQLite) Path() string { return s.path }

func (s *SQLite) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

func (s *SQLite) migrate() error {
	const schema = `
CREATE TABLE IF NOT EXISTS import_jobs (
	id             TEXT PRIMARY KEY NOT NULL,
	client_id      TEXT NOT NULL,
	format         TEXT NOT NULL,
	mode           TEXT NOT NULL DEFAULT '',
	status         TEXT NOT NULL,
	progress_pct   REAL NOT NULL DEFAULT 0,
	stage          TEXT NOT NULL DEFAULT '',
	error_message  TEXT NOT NULL DEFAULT '',
	section_counts TEXT NOT NULL DEFAULT '{}',
	totals_json    TEXT NOT NULL DEFAULT '{}',
	added_counts   TEXT NOT NULL DEFAULT '{}',
	file_name      TEXT NOT NULL DEFAULT '',
	file_path      TEXT NOT NULL DEFAULT '',
	payload_path   TEXT NOT NULL DEFAULT '',
	byte_size      INTEGER NOT NULL DEFAULT 0,
	expires_at     TEXT,
	created_at     TEXT NOT NULL,
	started_at     TEXT,
	finished_at    TEXT
);
CREATE INDEX IF NOT EXISTS idx_import_jobs_client ON import_jobs(client_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_import_jobs_status ON import_jobs(status);
CREATE INDEX IF NOT EXISTS idx_import_jobs_expires ON import_jobs(expires_at);
`
	if _, err := s.db.Exec(schema); err != nil {
		return fmt.Errorf("import sqlite migrate: %w", err)
	}
	return nil
}

func nullTime(t *time.Time) any {
	if t == nil || t.IsZero() {
		return nil
	}
	return t.UTC().Format(time.RFC3339Nano)
}

func parseNullTime(ns sql.NullString) *time.Time {
	if !ns.Valid || ns.String == "" {
		return nil
	}
	t, err := time.Parse(time.RFC3339Nano, ns.String)
	if err != nil {
		t2, err2 := time.Parse(time.RFC3339, ns.String)
		if err2 != nil {
			return nil
		}
		t = t2
	}
	u := t.UTC()
	return &u
}

func parseTime(s string) time.Time {
	t, err := time.Parse(time.RFC3339Nano, s)
	if err != nil {
		t, _ = time.Parse(time.RFC3339, s)
	}
	return t.UTC()
}

func marshalCounts(m map[domain.ExportSection]domain.ImportSectionCount) string {
	if m == nil {
		return "{}"
	}
	// JSON keys as strings
	tmp := make(map[string]domain.ImportSectionCount, len(m))
	for k, v := range m {
		tmp[string(k)] = v
	}
	b, _ := json.Marshal(tmp)
	return string(b)
}

func unmarshalCounts(s string) map[domain.ExportSection]domain.ImportSectionCount {
	out := map[domain.ExportSection]domain.ImportSectionCount{}
	if s == "" || s == "{}" {
		return out
	}
	var tmp map[string]domain.ImportSectionCount
	if err := json.Unmarshal([]byte(s), &tmp); err != nil {
		return out
	}
	for k, v := range tmp {
		out[domain.ExportSection(k)] = v
	}
	return out
}

func marshalAdded(m map[domain.ExportSection]int) string {
	if m == nil {
		return "{}"
	}
	tmp := make(map[string]int, len(m))
	for k, v := range m {
		tmp[string(k)] = v
	}
	b, _ := json.Marshal(tmp)
	return string(b)
}

func unmarshalAdded(s string) map[domain.ExportSection]int {
	out := map[domain.ExportSection]int{}
	if s == "" || s == "{}" {
		return out
	}
	var tmp map[string]int
	if err := json.Unmarshal([]byte(s), &tmp); err != nil {
		return out
	}
	for k, v := range tmp {
		out[domain.ExportSection(k)] = v
	}
	return out
}

func marshalTotals(t domain.ImportPreviewTotals) string {
	b, _ := json.Marshal(t)
	return string(b)
}

func unmarshalTotals(s string) domain.ImportPreviewTotals {
	var t domain.ImportPreviewTotals
	_ = json.Unmarshal([]byte(s), &t)
	return t
}

const jobCols = `id, client_id, format, mode, status, progress_pct, stage, error_message,
	section_counts, totals_json, added_counts, file_name, file_path, payload_path, byte_size,
	expires_at, created_at, started_at, finished_at`

func scanJob(sc interface{ Scan(dest ...any) error }) (*domain.ImportJob, error) {
	var j domain.ImportJob
	var sectionsJSON, totalsJSON, addedJSON, created string
	var expires, started, finished sql.NullString
	err := sc.Scan(
		&j.ID, &j.ClientID, &j.Format, &j.Mode, &j.Status, &j.ProgressPct, &j.Stage, &j.ErrorMessage,
		&sectionsJSON, &totalsJSON, &addedJSON, &j.FileName, &j.FilePath, &j.PayloadPath, &j.ByteSize,
		&expires, &created, &started, &finished,
	)
	if err != nil {
		return nil, err
	}
	j.SectionCounts = unmarshalCounts(sectionsJSON)
	j.Totals = unmarshalTotals(totalsJSON)
	j.AddedCounts = unmarshalAdded(addedJSON)
	j.CreatedAt = parseTime(created)
	j.ExpiresAt = parseNullTime(expires)
	j.StartedAt = parseNullTime(started)
	j.FinishedAt = parseNullTime(finished)
	return &j, nil
}

func scanJobs(rows *sql.Rows) ([]domain.ImportJob, error) {
	var out []domain.ImportJob
	for rows.Next() {
		j, err := scanJob(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *j)
	}
	return out, rows.Err()
}

func (s *SQLite) Create(ctx context.Context, job domain.ImportJob) (*domain.ImportJob, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO import_jobs (`+jobCols+`)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, job.ID, job.ClientID, string(job.Format), string(job.Mode), string(job.Status),
		job.ProgressPct, job.Stage, job.ErrorMessage,
		marshalCounts(job.SectionCounts), marshalTotals(job.Totals), marshalAdded(job.AddedCounts),
		job.FileName, job.FilePath, job.PayloadPath, job.ByteSize,
		nullTime(job.ExpiresAt), job.CreatedAt.UTC().Format(time.RFC3339Nano),
		nullTime(job.StartedAt), nullTime(job.FinishedAt))
	if err != nil {
		return nil, err
	}
	return s.getByIDUnlocked(ctx, job.ID)
}

func (s *SQLite) getByIDUnlocked(ctx context.Context, id string) (*domain.ImportJob, error) {
	row := s.db.QueryRowContext(ctx, `SELECT `+jobCols+` FROM import_jobs WHERE id = ?`, id)
	j, err := scanJob(row)
	if err == sql.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	return j, err
}

func (s *SQLite) Get(ctx context.Context, clientID, id string) (*domain.ImportJob, error) {
	row := s.db.QueryRowContext(ctx, `SELECT `+jobCols+` FROM import_jobs WHERE id = ? AND client_id = ?`, id, clientID)
	j, err := scanJob(row)
	if err == sql.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	return j, err
}

func (s *SQLite) GetByID(ctx context.Context, id string) (*domain.ImportJob, error) {
	return s.getByIDUnlocked(ctx, id)
}

func (s *SQLite) ListByClient(ctx context.Context, clientID string, limit, offset int) ([]domain.ImportJob, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT `+jobCols+` FROM import_jobs WHERE client_id = ?
		ORDER BY created_at DESC LIMIT ? OFFSET ?
	`, clientID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanJobs(rows)
}

func (s *SQLite) CountByClient(ctx context.Context, clientID string) (int, error) {
	var n int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM import_jobs WHERE client_id = ?`, clientID).Scan(&n)
	return n, err
}

func (s *SQLite) FindActiveApply(ctx context.Context, clientID string) (*domain.ImportJob, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT `+jobCols+` FROM import_jobs
		WHERE client_id = ? AND status IN ('pending','running')
		ORDER BY created_at ASC LIMIT 1
	`, clientID)
	j, err := scanJob(row)
	if err == sql.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	return j, err
}

func (s *SQLite) ListPending(ctx context.Context) ([]domain.ImportJob, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT `+jobCols+` FROM import_jobs WHERE status = 'pending' ORDER BY created_at ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanJobs(rows)
}

func (s *SQLite) Claim(ctx context.Context, id string, at time.Time) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	res, err := s.db.ExecContext(ctx, `
		UPDATE import_jobs SET status = 'running', started_at = ?, stage = 'starting'
		WHERE id = ? AND status = 'pending'
	`, at.UTC().Format(time.RFC3339Nano), id)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

func (s *SQLite) UpdateProgress(ctx context.Context, id string, progressPct float64, stage string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	res, err := s.db.ExecContext(ctx, `
		UPDATE import_jobs SET progress_pct = ?, stage = ? WHERE id = ?
	`, progressPct, stage, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (s *SQLite) Confirm(ctx context.Context, clientID, id string, mode domain.ImportMode, at time.Time) (*domain.ImportJob, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	res, err := s.db.ExecContext(ctx, `
		UPDATE import_jobs SET mode = ?, status = 'pending', stage = 'queued'
		WHERE id = ? AND client_id = ? AND status = 'previewed'
	`, string(mode), id, clientID)
	if err != nil {
		return nil, err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		// distinguish not found vs wrong status
		j, err := s.Get(ctx, clientID, id)
		if err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("%w: import is not in previewed state (status=%s)", domain.ErrConflict, j.Status)
	}
	return s.Get(ctx, clientID, id)
}

func (s *SQLite) UpdatePreviewStats(ctx context.Context, id string, counts map[domain.ExportSection]domain.ImportSectionCount, totals domain.ImportPreviewTotals, payloadPath string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	res, err := s.db.ExecContext(ctx, `
		UPDATE import_jobs SET section_counts = ?, totals_json = ?, payload_path = ? WHERE id = ?
	`, marshalCounts(counts), marshalTotals(totals), payloadPath, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (s *SQLite) Finish(ctx context.Context, id string, status domain.ImportStatus, added map[domain.ExportSection]int, errMsg string, at time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	pct := 0.0
	stage := string(status)
	if status == domain.ImportCompleted {
		pct = 100
		stage = "done"
	}
	res, err := s.db.ExecContext(ctx, `
		UPDATE import_jobs SET status = ?, progress_pct = CASE WHEN ? = 100 THEN 100 ELSE progress_pct END,
			stage = ?, error_message = ?, added_counts = ?, finished_at = ?
		WHERE id = ?
	`, string(status), pct, stage, errMsg, marshalAdded(added), at.UTC().Format(time.RFC3339Nano), id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (s *SQLite) Cancel(ctx context.Context, clientID, id string, at time.Time) (*domain.ImportJob, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	res, err := s.db.ExecContext(ctx, `
		UPDATE import_jobs SET status = 'canceled', stage = 'canceled', finished_at = ?
		WHERE id = ? AND client_id = ? AND status IN ('previewed','pending','running')
	`, at.UTC().Format(time.RFC3339Nano), id, clientID)
	if err != nil {
		return nil, err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		j, err := s.Get(ctx, clientID, id)
		if err != nil {
			return nil, err
		}
		if j.IsTerminal() {
			return j, nil
		}
		return nil, domain.ErrNotFound
	}
	return s.Get(ctx, clientID, id)
}

func (s *SQLite) GetStatus(ctx context.Context, id string) (domain.ImportStatus, error) {
	var st string
	err := s.db.QueryRowContext(ctx, `SELECT status FROM import_jobs WHERE id = ?`, id).Scan(&st)
	if err == sql.ErrNoRows {
		return "", domain.ErrNotFound
	}
	return domain.ImportStatus(st), err
}

func (s *SQLite) ListExpired(ctx context.Context, now time.Time, limit int) ([]domain.ImportJob, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT `+jobCols+` FROM import_jobs
		WHERE expires_at IS NOT NULL AND expires_at <= ?
		ORDER BY expires_at ASC LIMIT ?
	`, now.UTC().Format(time.RFC3339Nano), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanJobs(rows)
}

func (s *SQLite) Delete(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	res, err := s.db.ExecContext(ctx, `DELETE FROM import_jobs WHERE id = ?`, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (s *SQLite) RequeueStuckRunning(ctx context.Context, before time.Time) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	res, err := s.db.ExecContext(ctx, `
		UPDATE import_jobs SET status = 'pending', started_at = NULL, stage = 'requeued'
		WHERE status = 'running' AND started_at IS NOT NULL AND started_at < ?
	`, before.UTC().Format(time.RFC3339Nano))
	if err != nil {
		return 0, err
	}
	n, _ := res.RowsAffected()
	return int(n), nil
}

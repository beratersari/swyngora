package exportstore

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/adapter/sqliteutil"
	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"

	_ "modernc.org/sqlite"
)

// SQLite persists export job metadata.
type SQLite struct {
	db   *sql.DB
	mu   sync.Mutex
	path string
}

// Open opens or creates the export jobs database.
func Open(path string) (*SQLite, error) {
	path = filepath.Clean(path)
	if path == "" || path == "." {
		return nil, fmt.Errorf("export sqlite path is required")
	}
	if dir := filepath.Dir(path); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("create export db dir: %w", err)
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

// Path returns the absolute DB path.
func (s *SQLite) Path() string { return s.path }

// Close releases the database.
func (s *SQLite) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

func (s *SQLite) migrate() error {
	const schema = `
CREATE TABLE IF NOT EXISTS export_jobs (
	id            TEXT PRIMARY KEY NOT NULL,
	client_id     TEXT NOT NULL,
	format        TEXT NOT NULL,
	sections      TEXT NOT NULL,
	status        TEXT NOT NULL,
	progress_pct  REAL NOT NULL DEFAULT 0,
	stage         TEXT NOT NULL DEFAULT '',
	error_message TEXT NOT NULL DEFAULT '',
	file_name     TEXT NOT NULL DEFAULT '',
	file_path     TEXT NOT NULL DEFAULT '',
	byte_size     INTEGER NOT NULL DEFAULT 0,
	expires_at    TEXT,
	created_at    TEXT NOT NULL,
	started_at    TEXT,
	finished_at   TEXT
);
CREATE INDEX IF NOT EXISTS idx_export_jobs_client ON export_jobs(client_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_export_jobs_status ON export_jobs(status);
CREATE INDEX IF NOT EXISTS idx_export_jobs_expires ON export_jobs(status, expires_at);
`
	if _, err := s.db.Exec(schema); err != nil {
		return fmt.Errorf("export sqlite migrate: %w", err)
	}
	return sqliteutil.SetUserVersion(s.db, 1)
}

func nullTime(t *time.Time) any {
	if t == nil || t.IsZero() {
		return nil
	}
	return t.UTC().Format(time.RFC3339Nano)
}

func parseNullTime(s sql.NullString) *time.Time {
	if !s.Valid || s.String == "" {
		return nil
	}
	t, err := time.Parse(time.RFC3339Nano, s.String)
	if err != nil {
		t2, err2 := time.Parse(time.RFC3339, s.String)
		if err2 != nil {
			return nil
		}
		t = t2
	}
	u := t.UTC()
	return &u
}

func scanJob(sc interface {
	Scan(dest ...any) error
}) (*domain.ExportJob, error) {
	var j domain.ExportJob
	var sections, created string
	var expires, started, finished sql.NullString
	err := sc.Scan(
		&j.ID, &j.ClientID, &j.Format, &sections, &j.Status, &j.ProgressPct, &j.Stage,
		&j.ErrorMessage, &j.FileName, &j.FilePath, &j.ByteSize, &expires, &created, &started, &finished,
	)
	if err != nil {
		return nil, err
	}
	j.Sections = domain.ParseSectionsCSV(sections)
	j.CreatedAt = parseTime(created)
	j.ExpiresAt = parseNullTime(expires)
	j.StartedAt = parseNullTime(started)
	j.FinishedAt = parseNullTime(finished)
	return &j, nil
}

func parseTime(s string) time.Time {
	t, err := time.Parse(time.RFC3339Nano, s)
	if err != nil {
		t, _ = time.Parse(time.RFC3339, s)
	}
	return t.UTC()
}

const jobCols = `id, client_id, format, sections, status, progress_pct, stage, error_message,
	file_name, file_path, byte_size, expires_at, created_at, started_at, finished_at`

// Create inserts a job.
func (s *SQLite) Create(ctx context.Context, job domain.ExportJob) (*domain.ExportJob, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO export_jobs (`+jobCols+`)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, job.ID, job.ClientID, string(job.Format), domain.SectionsCSV(job.Sections), string(job.Status),
		job.ProgressPct, job.Stage, job.ErrorMessage, job.FileName, job.FilePath, job.ByteSize,
		nullTime(job.ExpiresAt), job.CreatedAt.UTC().Format(time.RFC3339Nano),
		nullTime(job.StartedAt), nullTime(job.FinishedAt))
	if err != nil {
		return nil, err
	}
	return s.GetByID(ctx, job.ID)
}

// Get returns a job for client.
func (s *SQLite) Get(ctx context.Context, clientID, id string) (*domain.ExportJob, error) {
	row := s.db.QueryRowContext(ctx, `SELECT `+jobCols+` FROM export_jobs WHERE id = ? AND client_id = ?`, id, clientID)
	j, err := scanJob(row)
	if err == sql.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	return j, err
}

// GetByID returns a job by id.
func (s *SQLite) GetByID(ctx context.Context, id string) (*domain.ExportJob, error) {
	row := s.db.QueryRowContext(ctx, `SELECT `+jobCols+` FROM export_jobs WHERE id = ?`, id)
	j, err := scanJob(row)
	if err == sql.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	return j, err
}

// ListByClient lists newest-first.
func (s *SQLite) ListByClient(ctx context.Context, clientID string, limit, offset int) ([]domain.ExportJob, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT `+jobCols+` FROM export_jobs WHERE client_id = ?
		ORDER BY created_at DESC LIMIT ? OFFSET ?
	`, clientID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanJobs(rows)
}

func scanJobs(rows *sql.Rows) ([]domain.ExportJob, error) {
	var out []domain.ExportJob
	for rows.Next() {
		j, err := scanJob(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *j)
	}
	return out, rows.Err()
}

// CountByClient counts jobs.
func (s *SQLite) CountByClient(ctx context.Context, clientID string) (int, error) {
	var n int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM export_jobs WHERE client_id = ?`, clientID).Scan(&n)
	return n, err
}

// FindActive finds pending/running for client.
func (s *SQLite) FindActive(ctx context.Context, clientID string) (*domain.ExportJob, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT `+jobCols+` FROM export_jobs
		WHERE client_id = ? AND status IN ('pending','running')
		ORDER BY created_at ASC LIMIT 1
	`, clientID)
	j, err := scanJob(row)
	if err == sql.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	return j, err
}

// ListPending lists pending jobs.
func (s *SQLite) ListPending(ctx context.Context) ([]domain.ExportJob, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT `+jobCols+` FROM export_jobs WHERE status = 'pending' ORDER BY created_at ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanJobs(rows)
}

// Claim moves pending → running.
func (s *SQLite) Claim(ctx context.Context, id string, at time.Time) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	res, err := s.db.ExecContext(ctx, `
		UPDATE export_jobs SET status = 'running', started_at = ?, stage = 'starting'
		WHERE id = ? AND status = 'pending'
	`, at.UTC().Format(time.RFC3339Nano), id)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

// UpdateProgress updates progress.
func (s *SQLite) UpdateProgress(ctx context.Context, id string, progressPct float64, stage string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	res, err := s.db.ExecContext(ctx, `
		UPDATE export_jobs SET progress_pct = ?, stage = ? WHERE id = ?
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

// Finish sets terminal status.
func (s *SQLite) Finish(ctx context.Context, id string, status domain.ExportStatus, fileName, filePath string, byteSize int64, expiresAt *time.Time, errMsg string, at time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	pct := 0.0
	stage := string(status)
	if status == domain.ExportCompleted {
		pct = 100
		stage = "done"
	}
	res, err := s.db.ExecContext(ctx, `
		UPDATE export_jobs SET status = ?, progress_pct = CASE WHEN ? = 100 THEN 100 ELSE progress_pct END,
			stage = ?, error_message = ?, file_name = ?, file_path = ?, byte_size = ?,
			expires_at = ?, finished_at = ?
		WHERE id = ?
	`, string(status), pct, stage, errMsg, fileName, filePath, byteSize, nullTime(expiresAt),
		at.UTC().Format(time.RFC3339Nano), id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return domain.ErrNotFound
	}
	return nil
}

// Cancel cancels active job.
func (s *SQLite) Cancel(ctx context.Context, clientID, id string, at time.Time) (*domain.ExportJob, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	res, err := s.db.ExecContext(ctx, `
		UPDATE export_jobs SET status = 'canceled', stage = 'canceled', finished_at = ?
		WHERE id = ? AND client_id = ? AND status IN ('pending','running')
	`, at.UTC().Format(time.RFC3339Nano), id, clientID)
	if err != nil {
		return nil, err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		// Return current if terminal own job
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

// GetStatus returns status.
func (s *SQLite) GetStatus(ctx context.Context, id string) (domain.ExportStatus, error) {
	var st string
	err := s.db.QueryRowContext(ctx, `SELECT status FROM export_jobs WHERE id = ?`, id).Scan(&st)
	if err == sql.ErrNoRows {
		return "", domain.ErrNotFound
	}
	return domain.ExportStatus(st), err
}

// ListExpiredCompleted lists expired files.
func (s *SQLite) ListExpiredCompleted(ctx context.Context, now time.Time, limit int) ([]domain.ExportJob, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT `+jobCols+` FROM export_jobs
		WHERE status = 'completed' AND expires_at IS NOT NULL AND expires_at <= ?
		ORDER BY expires_at ASC LIMIT ?
	`, now.UTC().Format(time.RFC3339Nano), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanJobs(rows)
}

// Delete removes a job row.
func (s *SQLite) Delete(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	res, err := s.db.ExecContext(ctx, `DELETE FROM export_jobs WHERE id = ?`, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return domain.ErrNotFound
	}
	return nil
}

// RequeueStuckRunning resets stuck running jobs.
func (s *SQLite) RequeueStuckRunning(ctx context.Context, before time.Time) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	res, err := s.db.ExecContext(ctx, `
		UPDATE export_jobs SET status = 'pending', started_at = NULL, stage = 'requeued'
		WHERE status = 'running' AND started_at IS NOT NULL AND started_at < ?
	`, before.UTC().Format(time.RFC3339Nano))
	if err != nil {
		return 0, err
	}
	n, _ := res.RowsAffected()
	return int(n), nil
}

// PurgeClient deletes all export jobs for clientID and returns them for file cleanup.
func (s *SQLite) PurgeClient(ctx context.Context, clientID string) ([]domain.ExportJob, error) {
	list, err := s.ListByClient(ctx, clientID, 10_000, 0)
	if err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, err := s.db.ExecContext(ctx, `DELETE FROM export_jobs WHERE client_id = ?`, clientID); err != nil {
		return nil, err
	}
	return list, nil
}

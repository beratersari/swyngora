package alertstore

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

// SQLite is a file-backed price alert store implementing domain.PriceAlertPort.
type SQLite struct {
	db   *sql.DB
	mu   sync.Mutex
	path string
}

// Open opens (or creates) a SQLite database at path and migrates schema.
func Open(path string) (*SQLite, error) {
	path = filepath.Clean(path)
	if path == "" || path == "." {
		return nil, fmt.Errorf("alerts sqlite path is required")
	}
	if dir := filepath.Dir(path); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("create alerts db dir: %w", err)
		}
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("alerts sqlite abs path: %w", err)
	}
	dsn := "file:" + filepath.ToSlash(abs) + "?_pragma=busy_timeout(5000)&_pragma=foreign_keys(1)"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open alerts sqlite: %w", err)
	}
	db.SetMaxOpenConns(4)
	db.SetMaxIdleConns(2)
	db.SetConnMaxLifetime(0)
	if _, err := db.Exec(`PRAGMA journal_mode=WAL`); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("alerts sqlite wal: %w", err)
	}
	if _, err := db.Exec(`PRAGMA foreign_keys=ON`); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("alerts sqlite foreign_keys: %w", err)
	}
	s := &SQLite{db: db, path: abs}
	if err := s.migrate(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return s, nil
}

func (s *SQLite) migrate() error {
	const schema = `
CREATE TABLE IF NOT EXISTS price_alerts (
	id              TEXT PRIMARY KEY NOT NULL,
	client_id       TEXT NOT NULL,
	exchange        TEXT NOT NULL,
	symbol          TEXT NOT NULL,
	condition       TEXT NOT NULL,
	target_price    REAL NOT NULL,
	mode            TEXT NOT NULL DEFAULT 'one_time',
	armed           INTEGER NOT NULL DEFAULT 0,
	status          TEXT NOT NULL,
	created_at      TEXT NOT NULL,
	triggered_at    TEXT,
	triggered_price REAL NOT NULL DEFAULT 0,
	kind            TEXT NOT NULL DEFAULT 'price',
	range_pct       REAL NOT NULL DEFAULT 0
);
CREATE INDEX IF NOT EXISTS idx_price_alerts_client ON price_alerts(client_id);
CREATE INDEX IF NOT EXISTS idx_price_alerts_active ON price_alerts(status) WHERE status = 'active';

CREATE TABLE IF NOT EXISTS client_webhooks (
	client_id      TEXT PRIMARY KEY NOT NULL,
	url            TEXT NOT NULL,
	delivery_mode  TEXT NOT NULL DEFAULT 'immediate',
	timezone       TEXT NOT NULL DEFAULT 'UTC',
	quiet_enabled  INTEGER NOT NULL DEFAULT 0,
	quiet_start    TEXT NOT NULL DEFAULT '',
	quiet_end      TEXT NOT NULL DEFAULT '',
	updated_at     TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS alert_digests (
	id              TEXT PRIMARY KEY NOT NULL,
	client_id       TEXT NOT NULL,
	webhook_url     TEXT NOT NULL,
	window_start    TEXT NOT NULL,
	window_end      TEXT NOT NULL,
	status          TEXT NOT NULL,
	attempts        INTEGER NOT NULL DEFAULT 0,
	next_attempt_at TEXT NOT NULL,
	last_error      TEXT NOT NULL DEFAULT '',
	payload_json    TEXT NOT NULL DEFAULT '',
	created_at      TEXT NOT NULL,
	sealed_at       TEXT,
	delivered_at    TEXT,
	UNIQUE (client_id, window_start)
);
CREATE INDEX IF NOT EXISTS idx_alert_digests_due
	ON alert_digests(status, next_attempt_at);
CREATE INDEX IF NOT EXISTS idx_alert_digests_open
	ON alert_digests(status, window_end);

CREATE TABLE IF NOT EXISTS alert_digest_items (
	digest_id    TEXT NOT NULL,
	alert_id     TEXT NOT NULL,
	payload_json TEXT NOT NULL,
	created_at   TEXT NOT NULL,
	PRIMARY KEY (digest_id, alert_id),
	FOREIGN KEY (digest_id) REFERENCES alert_digests(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS alert_notifications (
	id              TEXT PRIMARY KEY NOT NULL,
	alert_id        TEXT NOT NULL,
	client_id       TEXT NOT NULL,
	webhook_url     TEXT NOT NULL,
	payload_json    TEXT NOT NULL,
	status          TEXT NOT NULL,
	attempts        INTEGER NOT NULL DEFAULT 0,
	next_attempt_at TEXT NOT NULL,
	last_error      TEXT NOT NULL DEFAULT '',
	created_at      TEXT NOT NULL,
	delivered_at    TEXT
);
CREATE INDEX IF NOT EXISTS idx_alert_notifications_due
	ON alert_notifications(status, next_attempt_at);
CREATE INDEX IF NOT EXISTS idx_alert_notifications_client
	ON alert_notifications(client_id);
CREATE INDEX IF NOT EXISTS idx_alert_notifications_alert
	ON alert_notifications(alert_id);
`
	if _, err := s.db.Exec(schema); err != nil {
		return fmt.Errorf("alerts sqlite migrate: %w", err)
	}
	// Additive columns for DBs created before newer webhook prefs existed.
	for _, stmt := range []string{
		`ALTER TABLE price_alerts ADD COLUMN mode TEXT NOT NULL DEFAULT 'one_time'`,
		`ALTER TABLE price_alerts ADD COLUMN armed INTEGER NOT NULL DEFAULT 0`,
		`ALTER TABLE client_webhooks ADD COLUMN delivery_mode TEXT NOT NULL DEFAULT 'immediate'`,
		`ALTER TABLE client_webhooks ADD COLUMN timezone TEXT NOT NULL DEFAULT 'UTC'`,
		`ALTER TABLE client_webhooks ADD COLUMN quiet_enabled INTEGER NOT NULL DEFAULT 0`,
		`ALTER TABLE client_webhooks ADD COLUMN quiet_start TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE client_webhooks ADD COLUMN quiet_end TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE price_alerts ADD COLUMN kind TEXT NOT NULL DEFAULT 'price'`,
		`ALTER TABLE price_alerts ADD COLUMN range_pct REAL NOT NULL DEFAULT 0`,
	} {
		_, _ = s.db.Exec(stmt) // ignore "duplicate column" errors
	}
	if err := s.migrateNotificationsDropAlertUnique(); err != nil {
		return err
	}
	return nil
}

// migrateNotificationsDropAlertUnique rebuilds alert_notifications without UNIQUE(alert_id)
// so repeating alerts can enqueue one row per fire.
func (s *SQLite) migrateNotificationsDropAlertUnique() error {
	var createSQL string
	err := s.db.QueryRow(`SELECT sql FROM sqlite_master WHERE type='table' AND name='alert_notifications'`).Scan(&createSQL)
	if err == sql.ErrNoRows || createSQL == "" {
		return nil
	}
	if err != nil {
		return fmt.Errorf("alerts sqlite inspect notifications: %w", err)
	}
	// Old schema: "alert_id ... UNIQUE" (PRIMARY KEY also implies UNIQUE on id only).
	hasAlertUnique := strings.Contains(createSQL, "alert_id") &&
		(strings.Contains(createSQL, "alert_id        TEXT NOT NULL UNIQUE") ||
			strings.Contains(createSQL, "alert_id TEXT NOT NULL UNIQUE") ||
			strings.Contains(createSQL, "UNIQUE(alert_id)") ||
			strings.Contains(createSQL, "UNIQUE (alert_id)"))
	if !hasAlertUnique {
		return nil
	}
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.Exec(`
CREATE TABLE alert_notifications_new (
	id              TEXT PRIMARY KEY NOT NULL,
	alert_id        TEXT NOT NULL,
	client_id       TEXT NOT NULL,
	webhook_url     TEXT NOT NULL,
	payload_json    TEXT NOT NULL,
	status          TEXT NOT NULL,
	attempts        INTEGER NOT NULL DEFAULT 0,
	next_attempt_at TEXT NOT NULL,
	last_error      TEXT NOT NULL DEFAULT '',
	created_at      TEXT NOT NULL,
	delivered_at    TEXT
);
INSERT INTO alert_notifications_new
	SELECT id, alert_id, client_id, webhook_url, payload_json, status,
	       attempts, next_attempt_at, last_error, created_at, delivered_at
	FROM alert_notifications;
DROP TABLE alert_notifications;
ALTER TABLE alert_notifications_new RENAME TO alert_notifications;
CREATE INDEX IF NOT EXISTS idx_alert_notifications_due
	ON alert_notifications(status, next_attempt_at);
CREATE INDEX IF NOT EXISTS idx_alert_notifications_client
	ON alert_notifications(client_id);
CREATE INDEX IF NOT EXISTS idx_alert_notifications_alert
	ON alert_notifications(alert_id);
`); err != nil {
		return fmt.Errorf("alerts sqlite rebuild notifications: %w", err)
	}
	return tx.Commit()
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

const alertSelectCols = `
	id, client_id, exchange, symbol, condition, target_price,
	mode, armed, status, created_at, triggered_at, triggered_price,
	kind, range_pct
`

// Create inserts a new alert. Caller supplies a unique ID and Active status.
func (s *SQLite) Create(ctx context.Context, alert domain.PriceAlert) (*domain.PriceAlert, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if alert.CreatedAt.IsZero() {
		alert.CreatedAt = time.Now().UTC()
	}
	if alert.Status == "" {
		alert.Status = domain.AlertStatusActive
	}
	if alert.Mode == "" {
		alert.Mode = domain.AlertModeOneTime
	}
	if alert.Kind == "" {
		alert.Kind = domain.AlertKindPrice
	}
	armed := 0
	if alert.Armed {
		armed = 1
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO price_alerts (
			id, client_id, exchange, symbol, condition, target_price,
			mode, armed, status, created_at, triggered_at, triggered_price,
			kind, range_pct
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		alert.ID, alert.ClientID, string(alert.Exchange), alert.Symbol,
		string(alert.Condition), alert.TargetPrice,
		string(alert.Mode), armed, string(alert.Status),
		alert.CreatedAt.UTC().Format(time.RFC3339Nano),
		nullTime(alert.TriggeredAt), alert.TriggeredPrice,
		string(alert.Kind), alert.RangePct,
	)
	if err != nil {
		return nil, fmt.Errorf("alerts sqlite insert: %w", err)
	}
	return cloneAlert(&alert), nil
}

// Get returns one alert for the client or ErrNotFound.
func (s *SQLite) Get(ctx context.Context, clientID, id string) (*domain.PriceAlert, error) {
	a, err := s.scanOne(ctx, s.db, `
		SELECT `+alertSelectCols+`
		FROM price_alerts WHERE id = ? AND client_id = ?
	`, id, clientID)
	if err == sql.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	return a, err
}

// ListByClient returns all alerts for a client (newest first).
func (s *SQLite) ListByClient(ctx context.Context, clientID string) ([]domain.PriceAlert, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT `+alertSelectCols+`
		FROM price_alerts WHERE client_id = ?
		ORDER BY created_at DESC
	`, clientID)
	if err != nil {
		return nil, fmt.Errorf("alerts sqlite list: %w", err)
	}
	defer rows.Close()
	return scanAll(rows)
}

// ListActive returns every active alert (any client).
func (s *SQLite) ListActive(ctx context.Context) ([]domain.PriceAlert, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT `+alertSelectCols+`
		FROM price_alerts WHERE status = ?
		ORDER BY created_at ASC
	`, string(domain.AlertStatusActive))
	if err != nil {
		return nil, fmt.Errorf("alerts sqlite list active: %w", err)
	}
	defer rows.Close()
	return scanAll(rows)
}

// MarkTriggered transitions active → triggered exactly once (one_time alerts).
func (s *SQLite) MarkTriggered(ctx context.Context, id string, price float64, at time.Time) (*domain.PriceAlert, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if at.IsZero() {
		at = time.Now().UTC()
	}
	res, err := s.db.ExecContext(ctx, `
		UPDATE price_alerts
		SET status = ?, triggered_at = ?, triggered_price = ?, armed = 0
		WHERE id = ? AND status = ?
	`, string(domain.AlertStatusTriggered), at.UTC().Format(time.RFC3339Nano), price,
		id, string(domain.AlertStatusActive))
	if err != nil {
		return nil, fmt.Errorf("alerts sqlite mark triggered: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return nil, err
	}
	if n == 0 {
		return nil, domain.ErrNotFound
	}
	return s.getByIDUnlocked(ctx, id)
}

// RecordRepeatingTrigger records a fire for a repeating alert (stays active, disarmed).
func (s *SQLite) RecordRepeatingTrigger(ctx context.Context, id string, price float64, at time.Time) (*domain.PriceAlert, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if at.IsZero() {
		at = time.Now().UTC()
	}
	res, err := s.db.ExecContext(ctx, `
		UPDATE price_alerts
		SET triggered_at = ?, triggered_price = ?, armed = 0
		WHERE id = ? AND status = ? AND mode = ?
	`, at.UTC().Format(time.RFC3339Nano), price,
		id, string(domain.AlertStatusActive), string(domain.AlertModeRepeating))
	if err != nil {
		return nil, fmt.Errorf("alerts sqlite record repeating: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return nil, err
	}
	if n == 0 {
		return nil, domain.ErrNotFound
	}
	return s.getByIDUnlocked(ctx, id)
}

// SetArmed updates the armed flag used by repeating edge detection.
func (s *SQLite) SetArmed(ctx context.Context, id string, armed bool) (*domain.PriceAlert, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	v := 0
	if armed {
		v = 1
	}
	res, err := s.db.ExecContext(ctx, `
		UPDATE price_alerts SET armed = ? WHERE id = ? AND status = ?
	`, v, id, string(domain.AlertStatusActive))
	if err != nil {
		return nil, fmt.Errorf("alerts sqlite set armed: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return nil, err
	}
	if n == 0 {
		return nil, domain.ErrNotFound
	}
	return s.getByIDUnlocked(ctx, id)
}

func (s *SQLite) getByIDUnlocked(ctx context.Context, id string) (*domain.PriceAlert, error) {
	a, err := s.scanOne(ctx, s.db, `
		SELECT `+alertSelectCols+`
		FROM price_alerts WHERE id = ?
	`, id)
	if err == sql.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	return a, err
}

// Delete removes an alert owned by clientID.
func (s *SQLite) Delete(ctx context.Context, clientID, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	res, err := s.db.ExecContext(ctx, `DELETE FROM price_alerts WHERE id = ? AND client_id = ?`, id, clientID)
	if err != nil {
		return fmt.Errorf("alerts sqlite delete: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return domain.ErrNotFound
	}
	return nil
}

// PurgeClient deletes all alerts and webhook settings for clientID.
func (s *SQLite) PurgeClient(ctx context.Context, clientID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, err := s.db.ExecContext(ctx, `DELETE FROM alert_notifications WHERE client_id = ?`, clientID); err != nil {
		return err
	}
	if _, err := s.db.ExecContext(ctx, `
		DELETE FROM alert_digest_items WHERE digest_id IN (
			SELECT id FROM alert_digests WHERE client_id = ?
		)
	`, clientID); err != nil {
		return err
	}
	if _, err := s.db.ExecContext(ctx, `DELETE FROM alert_digests WHERE client_id = ?`, clientID); err != nil {
		return err
	}
	if _, err := s.db.ExecContext(ctx, `DELETE FROM price_alerts WHERE client_id = ?`, clientID); err != nil {
		return err
	}
	if _, err := s.db.ExecContext(ctx, `DELETE FROM client_webhooks WHERE client_id = ?`, clientID); err != nil {
		return err
	}
	return nil
}

// CountByClient returns how many alerts the client owns.
func (s *SQLite) CountByClient(ctx context.Context, clientID string) (int, error) {
	var n int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM price_alerts WHERE client_id = ?`, clientID).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("alerts sqlite count: %w", err)
	}
	return n, nil
}

// GetWebhook returns the client's webhook, or empty URL if unset (not an error).
func (s *SQLite) GetWebhook(ctx context.Context, clientID string) (*domain.ClientWebhook, error) {
	return s.getWebhookUnlocked(ctx, clientID)
}

func (s *SQLite) getWebhookUnlocked(ctx context.Context, clientID string) (*domain.ClientWebhook, error) {
	var url, mode, tz, qStart, qEnd, updatedRaw string
	var quietEn int
	err := s.db.QueryRowContext(ctx, `
		SELECT url,
		       COALESCE(delivery_mode, 'immediate'),
		       COALESCE(timezone, 'UTC'),
		       COALESCE(quiet_enabled, 0),
		       COALESCE(quiet_start, ''),
		       COALESCE(quiet_end, ''),
		       updated_at
		FROM client_webhooks WHERE client_id = ?
	`, clientID).Scan(&url, &mode, &tz, &quietEn, &qStart, &qEnd, &updatedRaw)
	if err == sql.ErrNoRows {
		return &domain.ClientWebhook{
			ClientID:     clientID,
			URL:          "",
			DeliveryMode: domain.DeliveryImmediate,
			TimeZone:     "UTC",
			UpdatedAt:    time.Time{},
		}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("alerts sqlite get webhook: %w", err)
	}
	dm, ok := domain.NormalizeDeliveryMode(mode)
	if !ok {
		dm = domain.DeliveryImmediate
	}
	if strings.TrimSpace(tz) == "" {
		tz = "UTC"
	}
	return &domain.ClientWebhook{
		ClientID:          clientID,
		URL:               url,
		DeliveryMode:      dm,
		TimeZone:          tz,
		QuietHoursEnabled: quietEn != 0,
		QuietStart:        qStart,
		QuietEnd:          qEnd,
		UpdatedAt:         parseTime(updatedRaw),
	}, nil
}

// SetWebhook upserts the client's webhook URL and notification preferences.
func (s *SQLite) SetWebhook(ctx context.Context, clientID string, settings domain.WebhookSettings) (*domain.ClientWebhook, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	dm, ok := domain.NormalizeDeliveryMode(settings.DeliveryMode)
	if !ok {
		dm = domain.DeliveryImmediate
	}
	tz := strings.TrimSpace(settings.TimeZone)
	if tz == "" {
		tz = "UTC"
	}
	quietEn := 0
	if settings.QuietHoursEnabled {
		quietEn = 1
	}
	now := time.Now().UTC()
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO client_webhooks (
			client_id, url, delivery_mode, timezone, quiet_enabled, quiet_start, quiet_end, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(client_id) DO UPDATE SET
			url = excluded.url,
			delivery_mode = excluded.delivery_mode,
			timezone = excluded.timezone,
			quiet_enabled = excluded.quiet_enabled,
			quiet_start = excluded.quiet_start,
			quiet_end = excluded.quiet_end,
			updated_at = excluded.updated_at
	`, clientID, settings.URL, string(dm), tz, quietEn, settings.QuietStart, settings.QuietEnd, now.Format(time.RFC3339Nano))
	if err != nil {
		return nil, fmt.Errorf("alerts sqlite set webhook: %w", err)
	}
	return &domain.ClientWebhook{
		ClientID:          clientID,
		URL:               settings.URL,
		DeliveryMode:      dm,
		TimeZone:          tz,
		QuietHoursEnabled: settings.QuietHoursEnabled,
		QuietStart:        settings.QuietStart,
		QuietEnd:          settings.QuietEnd,
		UpdatedAt:         now,
	}, nil
}

// DeleteWebhook removes the client's webhook URL.
func (s *SQLite) DeleteWebhook(ctx context.Context, clientID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, err := s.db.ExecContext(ctx, `DELETE FROM client_webhooks WHERE client_id = ?`, clientID)
	if err != nil {
		return fmt.Errorf("alerts sqlite delete webhook: %w", err)
	}
	return nil
}

// EnqueueNotification inserts a new pending notification row (one per fire event).
func (s *SQLite) EnqueueNotification(ctx context.Context, n domain.AlertNotification) (*domain.AlertNotification, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if n.CreatedAt.IsZero() {
		n.CreatedAt = time.Now().UTC()
	}
	if n.NextAttemptAt.IsZero() {
		n.NextAttemptAt = n.CreatedAt
	}
	if n.Status == "" {
		n.Status = domain.NotificationPending
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO alert_notifications (
			id, alert_id, client_id, webhook_url, payload_json, status,
			attempts, next_attempt_at, last_error, created_at, delivered_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		n.ID, n.AlertID, n.ClientID, n.WebhookURL, n.PayloadJSON, string(n.Status),
		n.Attempts, n.NextAttemptAt.UTC().Format(time.RFC3339Nano), n.LastError,
		n.CreatedAt.UTC().Format(time.RFC3339Nano), nullTime(n.DeliveredAt),
	)
	if err != nil {
		return nil, fmt.Errorf("alerts sqlite enqueue: %w", err)
	}
	return cloneNotification(&n), nil
}

// ListDueNotifications returns pending deliveries ready to send.
func (s *SQLite) ListDueNotifications(ctx context.Context, now time.Time, limit int) ([]domain.AlertNotification, error) {
	if limit <= 0 {
		limit = 50
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, alert_id, client_id, webhook_url, payload_json, status,
		       attempts, next_attempt_at, last_error, created_at, delivered_at
		FROM alert_notifications
		WHERE status = ? AND next_attempt_at <= ?
		ORDER BY next_attempt_at ASC
		LIMIT ?
	`, string(domain.NotificationPending), now.UTC().Format(time.RFC3339Nano), limit)
	if err != nil {
		return nil, fmt.Errorf("alerts sqlite list due: %w", err)
	}
	defer rows.Close()
	return scanNotifications(rows)
}

// MarkNotificationDelivered marks a row delivered (idempotent if already delivered).
func (s *SQLite) MarkNotificationDelivered(ctx context.Context, id string, at time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if at.IsZero() {
		at = time.Now().UTC()
	}
	res, err := s.db.ExecContext(ctx, `
		UPDATE alert_notifications
		SET status = ?, delivered_at = ?, last_error = ''
		WHERE id = ? AND status = ?
	`, string(domain.NotificationDelivered), at.UTC().Format(time.RFC3339Nano),
		id, string(domain.NotificationPending))
	if err != nil {
		return fmt.Errorf("alerts sqlite mark delivered: %w", err)
	}
	// Already delivered is fine (0 rows).
	_, _ = res.RowsAffected()
	return nil
}

// ScheduleNotificationRetry keeps the row pending for a later attempt.
func (s *SQLite) ScheduleNotificationRetry(ctx context.Context, id string, attempts int, nextAt time.Time, lastErr string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if nextAt.IsZero() {
		nextAt = time.Now().UTC()
	}
	if len(lastErr) > 500 {
		lastErr = lastErr[:500]
	}
	res, err := s.db.ExecContext(ctx, `
		UPDATE alert_notifications
		SET attempts = ?, next_attempt_at = ?, last_error = ?
		WHERE id = ? AND status = ?
	`, attempts, nextAt.UTC().Format(time.RFC3339Nano), lastErr,
		id, string(domain.NotificationPending))
	if err != nil {
		return fmt.Errorf("alerts sqlite schedule retry: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return domain.ErrNotFound
	}
	return nil
}

// FailNotification permanently fails a notification after max attempts.
func (s *SQLite) FailNotification(ctx context.Context, id string, lastErr string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(lastErr) > 500 {
		lastErr = lastErr[:500]
	}
	res, err := s.db.ExecContext(ctx, `
		UPDATE alert_notifications
		SET status = ?, last_error = ?
		WHERE id = ? AND status = ?
	`, string(domain.NotificationFailed), lastErr, id, string(domain.NotificationPending))
	if err != nil {
		return fmt.Errorf("alerts sqlite fail notification: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return domain.ErrNotFound
	}
	return nil
}

// GetNotificationByAlertID returns the outbox row for an alert.
func (s *SQLite) GetNotificationByAlertID(ctx context.Context, alertID string) (*domain.AlertNotification, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.getNotificationByAlertIDUnlocked(ctx, alertID)
}

func (s *SQLite) getNotificationByAlertIDUnlocked(ctx context.Context, alertID string) (*domain.AlertNotification, error) {
	// Latest notification for this alert (repeating alerts may have many).
	n, err := s.scanNotification(ctx, s.db, `
		SELECT id, alert_id, client_id, webhook_url, payload_json, status,
		       attempts, next_attempt_at, last_error, created_at, delivered_at
		FROM alert_notifications WHERE alert_id = ?
		ORDER BY created_at DESC
		LIMIT 1
	`, alertID)
	if err == sql.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	return n, err
}

func (s *SQLite) scanNotification(ctx context.Context, q rowScanner, query string, args ...any) (*domain.AlertNotification, error) {
	var (
		n                       domain.AlertNotification
		st, nextRaw, createdRaw string
		deliveredRaw            sql.NullString
	)
	err := q.QueryRowContext(ctx, query, args...).Scan(
		&n.ID, &n.AlertID, &n.ClientID, &n.WebhookURL, &n.PayloadJSON, &st,
		&n.Attempts, &nextRaw, &n.LastError, &createdRaw, &deliveredRaw,
	)
	if err != nil {
		return nil, err
	}
	n.Status = domain.NotificationStatus(st)
	n.NextAttemptAt = parseTime(nextRaw)
	n.CreatedAt = parseTime(createdRaw)
	if deliveredRaw.Valid && deliveredRaw.String != "" {
		t := parseTime(deliveredRaw.String)
		n.DeliveredAt = &t
	}
	return &n, nil
}

func scanNotifications(rows *sql.Rows) ([]domain.AlertNotification, error) {
	out := make([]domain.AlertNotification, 0)
	for rows.Next() {
		var (
			n                       domain.AlertNotification
			st, nextRaw, createdRaw string
			deliveredRaw            sql.NullString
		)
		if err := rows.Scan(
			&n.ID, &n.AlertID, &n.ClientID, &n.WebhookURL, &n.PayloadJSON, &st,
			&n.Attempts, &nextRaw, &n.LastError, &createdRaw, &deliveredRaw,
		); err != nil {
			return nil, fmt.Errorf("alerts sqlite scan notification: %w", err)
		}
		n.Status = domain.NotificationStatus(st)
		n.NextAttemptAt = parseTime(nextRaw)
		n.CreatedAt = parseTime(createdRaw)
		if deliveredRaw.Valid && deliveredRaw.String != "" {
			t := parseTime(deliveredRaw.String)
			n.DeliveredAt = &t
		}
		out = append(out, n)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func cloneNotification(n *domain.AlertNotification) *domain.AlertNotification {
	if n == nil {
		return nil
	}
	cp := *n
	if n.DeliveredAt != nil {
		t := n.DeliveredAt.UTC()
		cp.DeliveredAt = &t
	}
	return &cp
}

type rowScanner interface {
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

func (s *SQLite) scanOne(ctx context.Context, q rowScanner, query string, args ...any) (*domain.PriceAlert, error) {
	var (
		a                              domain.PriceAlert
		ex, cond, mode, st, createdRaw string
		armed                          int
		trigRaw                        sql.NullString
	)
	var kind string
	err := q.QueryRowContext(ctx, query, args...).Scan(
		&a.ID, &a.ClientID, &ex, &a.Symbol, &cond, &a.TargetPrice,
		&mode, &armed, &st, &createdRaw, &trigRaw, &a.TriggeredPrice,
		&kind, &a.RangePct,
	)
	if err != nil {
		return nil, err
	}
	a.Exchange = domain.Exchange(ex)
	a.Condition = domain.AlertCondition(cond)
	a.Mode = domain.AlertMode(mode)
	if a.Mode == "" {
		a.Mode = domain.AlertModeOneTime
	}
	a.Kind = domain.AlertKind(kind)
	if a.Kind == "" {
		a.Kind = domain.AlertKindPrice
	}
	a.Armed = armed != 0
	a.Status = domain.AlertStatus(st)
	a.CreatedAt = parseTime(createdRaw)
	if trigRaw.Valid && trigRaw.String != "" {
		t := parseTime(trigRaw.String)
		a.TriggeredAt = &t
	}
	return &a, nil
}

func scanAll(rows *sql.Rows) ([]domain.PriceAlert, error) {
	out := make([]domain.PriceAlert, 0)
	for rows.Next() {
		var (
			a                              domain.PriceAlert
			ex, cond, mode, st, createdRaw string
			armed                          int
			trigRaw                        sql.NullString
		)
		var kind string
		if err := rows.Scan(
			&a.ID, &a.ClientID, &ex, &a.Symbol, &cond, &a.TargetPrice,
			&mode, &armed, &st, &createdRaw, &trigRaw, &a.TriggeredPrice,
			&kind, &a.RangePct,
		); err != nil {
			return nil, fmt.Errorf("alerts sqlite scan: %w", err)
		}
		a.Exchange = domain.Exchange(ex)
		a.Condition = domain.AlertCondition(cond)
		a.Mode = domain.AlertMode(mode)
		if a.Mode == "" {
			a.Mode = domain.AlertModeOneTime
		}
		a.Kind = domain.AlertKind(kind)
		if a.Kind == "" {
			a.Kind = domain.AlertKindPrice
		}
		a.Armed = armed != 0
		a.Status = domain.AlertStatus(st)
		a.CreatedAt = parseTime(createdRaw)
		if trigRaw.Valid && trigRaw.String != "" {
			t := parseTime(trigRaw.String)
			a.TriggeredAt = &t
		}
		out = append(out, a)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func parseTime(raw string) time.Time {
	t, err := time.Parse(time.RFC3339Nano, raw)
	if err != nil {
		t, err = time.Parse(time.RFC3339, raw)
		if err != nil {
			return time.Time{}
		}
	}
	return t.UTC()
}

func nullTime(t *time.Time) any {
	if t == nil || t.IsZero() {
		return nil
	}
	return t.UTC().Format(time.RFC3339Nano)
}

func cloneAlert(a *domain.PriceAlert) *domain.PriceAlert {
	if a == nil {
		return nil
	}
	cp := *a
	if a.TriggeredAt != nil {
		t := a.TriggeredAt.UTC()
		cp.TriggeredAt = &t
	}
	return &cp
}

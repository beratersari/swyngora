package alertstore

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

// AddDigestItem adds or updates an alert in the client's open digest for the current hour.
// UNIQUE(digest_id, alert_id) ensures the same alert appears at most once; payload is refreshed.
func (s *SQLite) AddDigestItem(ctx context.Context, clientID, webhookURL, alertID, itemPayloadJSON string, at time.Time) (*domain.AlertDigest, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if at.IsZero() {
		at = time.Now().UTC()
	}
	ws, we := domain.DigestHourWindow(at)
	d, err := s.getOrCreateOpenDigestLocked(ctx, clientID, webhookURL, ws, we, at)
	if err != nil {
		return nil, err
	}
	_, err = s.db.ExecContext(ctx, `
		INSERT INTO alert_digest_items (digest_id, alert_id, payload_json, created_at)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(digest_id, alert_id) DO UPDATE SET
			payload_json = excluded.payload_json,
			created_at = excluded.created_at
	`, d.ID, alertID, itemPayloadJSON, at.UTC().Format(time.RFC3339Nano))
	if err != nil {
		return nil, fmt.Errorf("alerts sqlite digest item: %w", err)
	}
	// Keep webhook URL current for open digests.
	if webhookURL != "" && webhookURL != d.WebhookURL {
		_, _ = s.db.ExecContext(ctx, `UPDATE alert_digests SET webhook_url = ? WHERE id = ? AND status = ?`,
			webhookURL, d.ID, string(domain.DigestOpen))
		d.WebhookURL = webhookURL
	}
	return d, nil
}

func (s *SQLite) getOrCreateOpenDigestLocked(ctx context.Context, clientID, webhookURL string, windowStart, windowEnd, at time.Time) (*domain.AlertDigest, error) {
	d, err := s.scanDigest(ctx, s.db, `
		SELECT id, client_id, webhook_url, window_start, window_end, status,
		       attempts, next_attempt_at, last_error, payload_json, created_at, sealed_at, delivered_at
		FROM alert_digests
		WHERE client_id = ? AND window_start = ? AND status = ?
	`, clientID, windowStart.UTC().Format(time.RFC3339Nano), string(domain.DigestOpen))
	if err == nil {
		return d, nil
	}
	if err != sql.ErrNoRows {
		return nil, fmt.Errorf("alerts sqlite get open digest: %w", err)
	}
	id := uuid.NewString()
	_, err = s.db.ExecContext(ctx, `
		INSERT INTO alert_digests (
			id, client_id, webhook_url, window_start, window_end, status,
			attempts, next_attempt_at, last_error, payload_json, created_at, sealed_at, delivered_at
		) VALUES (?, ?, ?, ?, ?, ?, 0, ?, '', '', ?, NULL, NULL)
	`, id, clientID, webhookURL,
		windowStart.UTC().Format(time.RFC3339Nano),
		windowEnd.UTC().Format(time.RFC3339Nano),
		string(domain.DigestOpen),
		windowEnd.UTC().Format(time.RFC3339Nano), // next attempt after window ends
		at.UTC().Format(time.RFC3339Nano),
	)
	if err != nil {
		// Race: another writer created the same (client_id, window_start).
		d2, gerr := s.scanDigest(ctx, s.db, `
			SELECT id, client_id, webhook_url, window_start, window_end, status,
			       attempts, next_attempt_at, last_error, payload_json, created_at, sealed_at, delivered_at
			FROM alert_digests
			WHERE client_id = ? AND window_start = ?
		`, clientID, windowStart.UTC().Format(time.RFC3339Nano))
		if gerr == nil {
			return d2, nil
		}
		return nil, fmt.Errorf("alerts sqlite create digest: %w", err)
	}
	return &domain.AlertDigest{
		ID:            id,
		ClientID:      clientID,
		WebhookURL:    webhookURL,
		WindowStart:   windowStart.UTC(),
		WindowEnd:     windowEnd.UTC(),
		Status:        domain.DigestOpen,
		NextAttemptAt: windowEnd.UTC(),
		CreatedAt:     at.UTC(),
	}, nil
}

// SealOpenDigests seals open digests whose window has ended into pending with a combined payload.
func (s *SQLite) SealOpenDigests(ctx context.Context, now time.Time) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if now.IsZero() {
		now = time.Now().UTC()
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT id FROM alert_digests
		WHERE status = ? AND window_end <= ?
	`, string(domain.DigestOpen), now.UTC().Format(time.RFC3339Nano))
	if err != nil {
		return 0, fmt.Errorf("alerts sqlite list open digests: %w", err)
	}
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return 0, err
		}
		ids = append(ids, id)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, err
	}

	sealed := 0
	for _, id := range ids {
		if err := s.sealDigestLocked(ctx, id, now); err != nil {
			return sealed, err
		}
		sealed++
	}
	return sealed, nil
}

func (s *SQLite) sealDigestLocked(ctx context.Context, id string, now time.Time) error {
	d, err := s.scanDigest(ctx, s.db, `
		SELECT id, client_id, webhook_url, window_start, window_end, status,
		       attempts, next_attempt_at, last_error, payload_json, created_at, sealed_at, delivered_at
		FROM alert_digests WHERE id = ? AND status = ?
	`, id, string(domain.DigestOpen))
	if err == sql.ErrNoRows {
		return nil
	}
	if err != nil {
		return err
	}
	items, err := s.listDigestItemsUnlocked(ctx, id)
	if err != nil {
		return err
	}
	// Empty digests should not be delivered — mark failed/skip as delivered empty.
	if len(items) == 0 {
		_, err = s.db.ExecContext(ctx, `
			UPDATE alert_digests SET status = ?, sealed_at = ?, payload_json = ?, next_attempt_at = ?
			WHERE id = ? AND status = ?
		`, string(domain.DigestFailed), now.UTC().Format(time.RFC3339Nano), `{"type":"price_alert.digest","count":0}`,
			now.UTC().Format(time.RFC3339Nano), id, string(domain.DigestOpen))
		return err
	}
	payload, err := buildDigestPayload(d, items)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, `
		UPDATE alert_digests
		SET status = ?, payload_json = ?, sealed_at = ?, next_attempt_at = ?, attempts = 0, last_error = ''
		WHERE id = ? AND status = ?
	`, string(domain.DigestPending), payload, now.UTC().Format(time.RFC3339Nano),
		now.UTC().Format(time.RFC3339Nano), id, string(domain.DigestOpen))
	return err
}

func buildDigestPayload(d *domain.AlertDigest, items []domain.AlertDigestItem) (string, error) {
	alerts := make([]json.RawMessage, 0, len(items))
	for _, it := range items {
		alerts = append(alerts, json.RawMessage(it.PayloadJSON))
	}
	body := map[string]any{
		"type":         "price_alert.digest",
		"digestId":     d.ID,
		"clientId":     d.ClientID,
		"windowStart":  d.WindowStart.UTC().Format(time.RFC3339Nano),
		"windowEnd":    d.WindowEnd.UTC().Format(time.RFC3339Nano),
		"count":        len(items),
		"alerts":       alerts,
		"note":         "Informational only — not financial advice.",
	}
	b, err := json.Marshal(body)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// ListDueDigests returns sealed digests ready for delivery.
func (s *SQLite) ListDueDigests(ctx context.Context, now time.Time, limit int) ([]domain.AlertDigest, error) {
	if limit <= 0 {
		limit = 50
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, client_id, webhook_url, window_start, window_end, status,
		       attempts, next_attempt_at, last_error, payload_json, created_at, sealed_at, delivered_at
		FROM alert_digests
		WHERE status = ? AND next_attempt_at <= ?
		ORDER BY next_attempt_at ASC
		LIMIT ?
	`, string(domain.DigestPending), now.UTC().Format(time.RFC3339Nano), limit)
	if err != nil {
		return nil, fmt.Errorf("alerts sqlite list due digests: %w", err)
	}
	defer rows.Close()
	return scanDigests(rows)
}

// GetDigest returns one digest by id.
func (s *SQLite) GetDigest(ctx context.Context, id string) (*domain.AlertDigest, error) {
	d, err := s.scanDigest(ctx, s.db, `
		SELECT id, client_id, webhook_url, window_start, window_end, status,
		       attempts, next_attempt_at, last_error, payload_json, created_at, sealed_at, delivered_at
		FROM alert_digests WHERE id = ?
	`, id)
	if err == sql.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	return d, err
}

// ListDigestItems returns items for a digest ordered by created_at.
func (s *SQLite) ListDigestItems(ctx context.Context, digestID string) ([]domain.AlertDigestItem, error) {
	return s.listDigestItemsUnlocked(ctx, digestID)
}

func (s *SQLite) listDigestItemsUnlocked(ctx context.Context, digestID string) ([]domain.AlertDigestItem, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT digest_id, alert_id, payload_json, created_at
		FROM alert_digest_items WHERE digest_id = ?
		ORDER BY created_at ASC
	`, digestID)
	if err != nil {
		return nil, fmt.Errorf("alerts sqlite list digest items: %w", err)
	}
	defer rows.Close()
	out := make([]domain.AlertDigestItem, 0)
	for rows.Next() {
		var it domain.AlertDigestItem
		var createdRaw string
		if err := rows.Scan(&it.DigestID, &it.AlertID, &it.PayloadJSON, &createdRaw); err != nil {
			return nil, err
		}
		it.CreatedAt = parseTime(createdRaw)
		out = append(out, it)
	}
	return out, rows.Err()
}

// MarkDigestDelivered marks a pending digest delivered.
func (s *SQLite) MarkDigestDelivered(ctx context.Context, id string, at time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if at.IsZero() {
		at = time.Now().UTC()
	}
	_, err := s.db.ExecContext(ctx, `
		UPDATE alert_digests
		SET status = ?, delivered_at = ?, last_error = ''
		WHERE id = ? AND status = ?
	`, string(domain.DigestDelivered), at.UTC().Format(time.RFC3339Nano), id, string(domain.DigestPending))
	return err
}

// ScheduleDigestRetry keeps a digest pending for a later attempt.
func (s *SQLite) ScheduleDigestRetry(ctx context.Context, id string, attempts int, nextAt time.Time, lastErr string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if nextAt.IsZero() {
		nextAt = time.Now().UTC()
	}
	if len(lastErr) > 500 {
		lastErr = lastErr[:500]
	}
	res, err := s.db.ExecContext(ctx, `
		UPDATE alert_digests
		SET attempts = ?, next_attempt_at = ?, last_error = ?
		WHERE id = ? AND status = ?
	`, attempts, nextAt.UTC().Format(time.RFC3339Nano), lastErr, id, string(domain.DigestPending))
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return domain.ErrNotFound
	}
	return nil
}

// FailDigest permanently fails a digest.
func (s *SQLite) FailDigest(ctx context.Context, id string, lastErr string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(lastErr) > 500 {
		lastErr = lastErr[:500]
	}
	res, err := s.db.ExecContext(ctx, `
		UPDATE alert_digests SET status = ?, last_error = ?
		WHERE id = ? AND status = ?
	`, string(domain.DigestFailed), lastErr, id, string(domain.DigestPending))
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (s *SQLite) scanDigest(ctx context.Context, q rowScanner, query string, args ...any) (*domain.AlertDigest, error) {
	var (
		d                                  domain.AlertDigest
		ws, we, st, nextRaw, createdRaw    string
		sealedRaw, deliveredRaw            sql.NullString
	)
	err := q.QueryRowContext(ctx, query, args...).Scan(
		&d.ID, &d.ClientID, &d.WebhookURL, &ws, &we, &st,
		&d.Attempts, &nextRaw, &d.LastError, &d.PayloadJSON, &createdRaw, &sealedRaw, &deliveredRaw,
	)
	if err != nil {
		return nil, err
	}
	d.WindowStart = parseTime(ws)
	d.WindowEnd = parseTime(we)
	d.Status = domain.DigestStatus(st)
	d.NextAttemptAt = parseTime(nextRaw)
	d.CreatedAt = parseTime(createdRaw)
	if sealedRaw.Valid && sealedRaw.String != "" {
		t := parseTime(sealedRaw.String)
		d.SealedAt = &t
	}
	if deliveredRaw.Valid && deliveredRaw.String != "" {
		t := parseTime(deliveredRaw.String)
		d.DeliveredAt = &t
	}
	return &d, nil
}

func scanDigests(rows *sql.Rows) ([]domain.AlertDigest, error) {
	out := make([]domain.AlertDigest, 0)
	for rows.Next() {
		var (
			d                               domain.AlertDigest
			ws, we, st, nextRaw, createdRaw string
			sealedRaw, deliveredRaw         sql.NullString
		)
		if err := rows.Scan(
			&d.ID, &d.ClientID, &d.WebhookURL, &ws, &we, &st,
			&d.Attempts, &nextRaw, &d.LastError, &d.PayloadJSON, &createdRaw, &sealedRaw, &deliveredRaw,
		); err != nil {
			return nil, err
		}
		d.WindowStart = parseTime(ws)
		d.WindowEnd = parseTime(we)
		d.Status = domain.DigestStatus(st)
		d.NextAttemptAt = parseTime(nextRaw)
		d.CreatedAt = parseTime(createdRaw)
		if sealedRaw.Valid && sealedRaw.String != "" {
			t := parseTime(sealedRaw.String)
			d.SealedAt = &t
		}
		if deliveredRaw.Valid && deliveredRaw.String != "" {
			t := parseTime(deliveredRaw.String)
			d.DeliveredAt = &t
		}
		out = append(out, d)
	}
	return out, rows.Err()
}
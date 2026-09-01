package scannerstore

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

// CreateProfile inserts a named custom signal mix.
func (s *SQLite) CreateProfile(ctx context.Context, p domain.CustomSignalProfile) (*domain.CustomSignalProfile, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if p.CreatedAt.IsZero() {
		p.CreatedAt = time.Now().UTC()
	}
	if p.UpdatedAt.IsZero() {
		p.UpdatedAt = p.CreatedAt
	}
	raw, err := encodeCustomWeights(p.Weights)
	if err != nil {
		return nil, err
	}
	def := 0
	if p.IsDefault {
		def = 1
	}
	err = s.withTx(ctx, func(tx *sql.Tx) error {
		if p.IsDefault {
			if _, err := tx.ExecContext(ctx, `UPDATE custom_signal_profiles SET is_default = 0 WHERE client_id = ?`, p.ClientID); err != nil {
				return err
			}
		}
		_, err := tx.ExecContext(ctx, `
			INSERT INTO custom_signal_profiles (id, client_id, name, weights_json, is_default, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?)
		`, p.ID, p.ClientID, p.Name, raw, def,
			p.CreatedAt.UTC().Format(time.RFC3339Nano), p.UpdatedAt.UTC().Format(time.RFC3339Nano))
		if err != nil {
			return fmt.Errorf("insert custom signal profile: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// GetProfile loads one profile for the owner.
func (s *SQLite) GetProfile(ctx context.Context, clientID, id string) (*domain.CustomSignalProfile, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	row := s.db.QueryRowContext(ctx, `
		SELECT id, client_id, name, weights_json, is_default, created_at, updated_at
		FROM custom_signal_profiles WHERE client_id = ? AND id = ?
	`, clientID, id)
	p, err := scanCustomProfile(row)
	if err == sql.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	return p, err
}

// UpdateProfile replaces name and weights.
func (s *SQLite) UpdateProfile(ctx context.Context, p domain.CustomSignalProfile) (*domain.CustomSignalProfile, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if p.UpdatedAt.IsZero() {
		p.UpdatedAt = time.Now().UTC()
	}
	raw, err := encodeCustomWeights(p.Weights)
	if err != nil {
		return nil, err
	}
	def := 0
	if p.IsDefault {
		def = 1
	}
	err = s.withTx(ctx, func(tx *sql.Tx) error {
		if p.IsDefault {
			if _, err := tx.ExecContext(ctx, `UPDATE custom_signal_profiles SET is_default = 0 WHERE client_id = ?`, p.ClientID); err != nil {
				return err
			}
		}
		res, err := tx.ExecContext(ctx, `
			UPDATE custom_signal_profiles
			SET name = ?, weights_json = ?, is_default = ?, updated_at = ?
			WHERE client_id = ? AND id = ?
		`, p.Name, raw, def, p.UpdatedAt.UTC().Format(time.RFC3339Nano), p.ClientID, p.ID)
		if err != nil {
			return fmt.Errorf("update custom signal profile: %w", err)
		}
		n, _ := res.RowsAffected()
		if n == 0 {
			return domain.ErrNotFound
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// SetDefaultProfile marks one mix as the client's default (clears the others).
// Clear + set run in one transaction so a mid-update failure cannot leave
// zero or two defaults.
func (s *SQLite) SetDefaultProfile(ctx context.Context, clientID, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.setDefaultLocked(ctx, clientID, id)
}

func (s *SQLite) setDefaultLocked(ctx context.Context, clientID, id string) error {
	return s.withTx(ctx, func(tx *sql.Tx) error {
		var n int
		if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM custom_signal_profiles WHERE client_id = ? AND id = ?`, clientID, id).Scan(&n); err != nil {
			return err
		}
		if n == 0 {
			return domain.ErrNotFound
		}
		if _, err := tx.ExecContext(ctx, `UPDATE custom_signal_profiles SET is_default = 0 WHERE client_id = ?`, clientID); err != nil {
			return err
		}
		res, err := tx.ExecContext(ctx, `UPDATE custom_signal_profiles SET is_default = 1 WHERE client_id = ? AND id = ?`, clientID, id)
		if err != nil {
			return err
		}
		got, _ := res.RowsAffected()
		if got == 0 {
			return domain.ErrNotFound
		}
		return nil
	})
}

func (s *SQLite) withTx(ctx context.Context, fn func(*sql.Tx) error) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("custom signal begin: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	if err := fn(tx); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("custom signal commit: %w", err)
	}
	return nil
}

// ListProfiles returns newest-updated first.
func (s *SQLite) ListProfiles(ctx context.Context, clientID string) ([]domain.CustomSignalProfile, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, client_id, name, weights_json, is_default, created_at, updated_at
		FROM custom_signal_profiles WHERE client_id = ?
		ORDER BY is_default DESC, updated_at DESC
	`, clientID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]domain.CustomSignalProfile, 0)
	for rows.Next() {
		p, err := scanCustomProfile(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *p)
	}
	return out, rows.Err()
}

// DeleteProfile removes one profile.
func (s *SQLite) DeleteProfile(ctx context.Context, clientID, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	res, err := s.db.ExecContext(ctx, `DELETE FROM custom_signal_profiles WHERE client_id = ? AND id = ?`, clientID, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return domain.ErrNotFound
	}
	return nil
}

// CountProfiles is the number of saved mixes for a client.
func (s *SQLite) CountProfiles(ctx context.Context, clientID string) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var n int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM custom_signal_profiles WHERE client_id = ?`, clientID).Scan(&n)
	return n, err
}

// PurgeProfiles deletes every mix for a client.
func (s *SQLite) PurgeProfiles(ctx context.Context, clientID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, err := s.db.ExecContext(ctx, `DELETE FROM custom_signal_profiles WHERE client_id = ?`, clientID)
	return err
}

type customProfileRow interface {
	Scan(dest ...any) error
}

func scanCustomProfile(row customProfileRow) (*domain.CustomSignalProfile, error) {
	var p domain.CustomSignalProfile
	var raw, cAt, uAt string
	var def int
	if err := row.Scan(&p.ID, &p.ClientID, &p.Name, &raw, &def, &cAt, &uAt); err != nil {
		return nil, err
	}
	p.IsDefault = def != 0
	w, err := decodeCustomWeights(raw)
	if err != nil {
		return nil, err
	}
	p.Weights = w
	p.CreatedAt, _ = time.Parse(time.RFC3339Nano, cAt)
	p.UpdatedAt, _ = time.Parse(time.RFC3339Nano, uAt)
	return &p, nil
}

func encodeCustomWeights(w domain.CustomSignalWeights) (string, error) {
	if w.Pct == nil {
		w = domain.DefaultCustomSignalWeights()
	}
	b, err := json.Marshal(w.Pct)
	if err != nil {
		return "", fmt.Errorf("%w: encode custom signal weights", domain.ErrInvalidArgument)
	}
	return string(b), nil
}

func decodeCustomWeights(raw string) (domain.CustomSignalWeights, error) {
	var pct map[string]float64
	if err := json.Unmarshal([]byte(raw), &pct); err != nil {
		return domain.CustomSignalWeights{}, fmt.Errorf("%w: invalid stored custom signal weights", domain.ErrInvalidArgument)
	}
	return domain.ParseCustomSignalWeightMap(pct)
}

package export

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

const maxClientIDLen = 128

// DataSources provides user-owned data for export (ports only — no HTTP).
type DataSources struct {
	Watchlist domain.WatchlistPort
	Alerts    domain.PriceAlertPort
	Scanner   domain.ScannerPort
}

// Service orchestrates user data exports.
type Service struct {
	store    domain.ExportPort
	data     DataSources
	fileDir  string
	fileTTL  time.Duration
}

// Options configures the export service.
type Options struct {
	// FileDir is the directory for export files (created if missing).
	FileDir string
	// FileTTL is how long completed files remain downloadable (default 1h).
	FileTTL time.Duration
}

// New constructs an export service.
func New(store domain.ExportPort, data DataSources, opts Options) (*Service, error) {
	dir := strings.TrimSpace(opts.FileDir)
	if dir == "" {
		dir = "data/exports"
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(abs, 0o755); err != nil {
		return nil, fmt.Errorf("export file dir: %w", err)
	}
	ttl := opts.FileTTL
	if ttl <= 0 {
		ttl = domain.DefaultExportFileTTL
	}
	if ttl < domain.MinExportFileTTL {
		ttl = domain.MinExportFileTTL
	}
	if ttl > domain.MaxExportFileTTL {
		ttl = domain.MaxExportFileTTL
	}
	return &Service{store: store, data: data, fileDir: abs, fileTTL: ttl}, nil
}

// FileDir returns the absolute export files directory.
func (s *Service) FileDir() string { return s.fileDir }

// FileTTL returns configured file retention.
func (s *Service) FileTTL() time.Duration { return s.fileTTL }

// StartInput starts a new export job.
type StartInput struct {
	ClientID string
	Format   string   // json | csv
	Sections []string // optional; empty = all
}

// Start queues an export. Fails with ErrConflict if one is already pending/running.
func (s *Service) Start(ctx context.Context, in StartInput) (*domain.ExportJob, error) {
	if s.store == nil {
		return nil, fmt.Errorf("%w: export store not configured", domain.ErrUpstream)
	}
	clientID, err := normalizeClientID(in.ClientID)
	if err != nil {
		return nil, err
	}
	format, err := domain.NormalizeExportFormat(in.Format)
	if err != nil {
		return nil, err
	}
	sections, err := domain.NormalizeExportSections(in.Sections)
	if err != nil {
		return nil, err
	}
	if active, err := s.store.FindActive(ctx, clientID); err == nil && active != nil {
		return nil, fmt.Errorf("%w: an export is already in progress (id=%s)", domain.ErrConflict, active.ID)
	} else if err != nil && err != domain.ErrNotFound {
		return nil, err
	}
	// Soft-cap retained jobs: delete oldest terminal jobs over limit.
	if err := s.pruneOldJobs(ctx, clientID); err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	job := domain.ExportJob{
		ID: uuid.NewString(), ClientID: clientID, Format: format, Sections: sections,
		Status: domain.ExportPending, ProgressPct: 0, Stage: "queued", CreatedAt: now,
	}
	return s.store.Create(ctx, job)
}

func (s *Service) pruneOldJobs(ctx context.Context, clientID string) error {
	n, err := s.store.CountByClient(ctx, clientID)
	if err != nil {
		return err
	}
	if n < domain.MaxExportsPerClient {
		return nil
	}
	// List more than limit and delete oldest terminal ones.
	list, err := s.store.ListByClient(ctx, clientID, n, 0)
	if err != nil {
		return err
	}
	// list is newest-first; walk from end
	for i := len(list) - 1; i >= 0 && n >= domain.MaxExportsPerClient; i-- {
		j := list[i]
		if j.IsActive() {
			continue
		}
		_ = s.deleteJobAndFile(ctx, &j)
		n--
	}
	return nil
}

func (s *Service) deleteJobAndFile(ctx context.Context, j *domain.ExportJob) error {
	if j.FilePath != "" {
		_ = os.Remove(j.FilePath)
	}
	return s.store.Delete(ctx, j.ID)
}

// Get returns job status for the owner.
func (s *Service) Get(ctx context.Context, clientID, id string) (*domain.ExportJob, error) {
	if s.store == nil {
		return nil, fmt.Errorf("%w: export store not configured", domain.ErrUpstream)
	}
	clientID, err := normalizeClientID(clientID)
	if err != nil {
		return nil, err
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, fmt.Errorf("%w: export id is required", domain.ErrInvalidArgument)
	}
	return s.store.Get(ctx, clientID, id)
}

// List returns recent exports for the client.
func (s *Service) List(ctx context.Context, clientID string, limit, offset int) ([]domain.ExportJob, error) {
	if s.store == nil {
		return nil, fmt.Errorf("%w: export store not configured", domain.ErrUpstream)
	}
	clientID, err := normalizeClientID(clientID)
	if err != nil {
		return nil, err
	}
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}
	return s.store.ListByClient(ctx, clientID, limit, offset)
}

// Cancel cancels a pending or running export.
func (s *Service) Cancel(ctx context.Context, clientID, id string) (*domain.ExportJob, error) {
	if s.store == nil {
		return nil, fmt.Errorf("%w: export store not configured", domain.ErrUpstream)
	}
	clientID, err := normalizeClientID(clientID)
	if err != nil {
		return nil, err
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, fmt.Errorf("%w: export id is required", domain.ErrInvalidArgument)
	}
	return s.store.Cancel(ctx, clientID, id, time.Now().UTC())
}

// OpenDownload returns an absolute file path for a completed, non-expired export owned by clientID.
// Caller must not delete the file; only stream it.
func (s *Service) OpenDownload(ctx context.Context, clientID, id string) (*domain.ExportJob, error) {
	job, err := s.Get(ctx, clientID, id)
	if err != nil {
		return nil, err
	}
	if job.Status != domain.ExportCompleted {
		return nil, fmt.Errorf("%w: export is not ready for download (status=%s)", domain.ErrInvalidArgument, job.Status)
	}
	if job.ExpiresAt != nil && time.Now().UTC().After(job.ExpiresAt.UTC()) {
		return nil, fmt.Errorf("%w: export file has expired", domain.ErrNotFound)
	}
	if job.FilePath == "" {
		return nil, fmt.Errorf("%w: export file missing", domain.ErrNotFound)
	}
	if _, err := os.Stat(job.FilePath); err != nil {
		return nil, fmt.Errorf("%w: export file not found", domain.ErrNotFound)
	}
	// Only the owning client reaches here (Get filters by clientID).
	return job, nil
}

// CleanupExpired removes expired completed exports (files + rows).
func (s *Service) CleanupExpired(ctx context.Context) (int, error) {
	if s.store == nil {
		return 0, nil
	}
	list, err := s.store.ListExpiredCompleted(ctx, time.Now().UTC(), 100)
	if err != nil {
		return 0, err
	}
	n := 0
	for i := range list {
		if err := s.deleteJobAndFile(ctx, &list[i]); err == nil {
			n++
		}
	}
	return n, nil
}

// RequeueStuckRunning recovers jobs left "running" after a process restart.
func (s *Service) RequeueStuckRunning(ctx context.Context, olderThan time.Duration) (int, error) {
	if s.store == nil {
		return 0, nil
	}
	if olderThan <= 0 {
		olderThan = 2 * time.Minute
	}
	return s.store.RequeueStuckRunning(ctx, time.Now().UTC().Add(-olderThan))
}

func normalizeClientID(id string) (string, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return "", fmt.Errorf("%w: clientId is required", domain.ErrInvalidArgument)
	}
	if len(id) > maxClientIDLen {
		return "", fmt.Errorf("%w: clientId too long", domain.ErrInvalidArgument)
	}
	if strings.EqualFold(id, "default") {
		return "", fmt.Errorf("%w: clientId must not be the shared name \"default\"", domain.ErrInvalidArgument)
	}
	for _, r := range id {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' || r == '.' {
			continue
		}
		return "", fmt.Errorf("%w: clientId has invalid characters", domain.ErrInvalidArgument)
	}
	if utf8.RuneCountInString(id) == 0 {
		return "", fmt.Errorf("%w: clientId is required", domain.ErrInvalidArgument)
	}
	return id, nil
}

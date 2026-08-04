package dataimport

import (
	"context"
	"encoding/json"
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

// DataSources provides ports used when applying an import.
type DataSources struct {
	Watchlist domain.WatchlistPort
	Alerts    domain.PriceAlertPort
	Scanner   domain.ScannerPort
}

// Service orchestrates import preview and apply.
type Service struct {
	store   domain.ImportPort
	data    DataSources
	fileDir string
	fileTTL time.Duration
}

// Options configures the import service.
type Options struct {
	FileDir string
	FileTTL time.Duration
}

// New constructs an import service.
func New(store domain.ImportPort, data DataSources, opts Options) (*Service, error) {
	dir := strings.TrimSpace(opts.FileDir)
	if dir == "" {
		dir = "data/imports"
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(abs, 0o755); err != nil {
		return nil, fmt.Errorf("import file dir: %w", err)
	}
	ttl := opts.FileTTL
	if ttl <= 0 {
		ttl = domain.DefaultImportFileTTL
	}
	if ttl < domain.MinImportFileTTL {
		ttl = domain.MinImportFileTTL
	}
	if ttl > domain.MaxImportFileTTL {
		ttl = domain.MaxImportFileTTL
	}
	return &Service{store: store, data: data, fileDir: abs, fileTTL: ttl}, nil
}

// FileDir returns the absolute imports directory.
func (s *Service) FileDir() string { return s.fileDir }

// PreviewInput is an uploaded export file for validation.
type PreviewInput struct {
	ClientID    string
	FileName    string
	FileBytes   []byte
	FormatHint  string // optional json|csv
}

// Preview parses the file, computes valid/invalid/willAdd counts (merge-oriented),
// and stores a previewed job that can be confirmed later.
func (s *Service) Preview(ctx context.Context, in PreviewInput) (*domain.ImportJob, error) {
	if s.store == nil {
		return nil, fmt.Errorf("%w: import store not configured", domain.ErrUpstream)
	}
	clientID, err := normalizeClientID(in.ClientID)
	if err != nil {
		return nil, err
	}
	if len(in.FileBytes) == 0 {
		return nil, fmt.Errorf("%w: empty upload", domain.ErrInvalidArgument)
	}
	if len(in.FileBytes) > domain.MaxImportUploadBytes {
		return nil, fmt.Errorf("%w: file too large (max %d bytes)", domain.ErrInvalidArgument, domain.MaxImportUploadBytes)
	}
	format := domain.ExportFormat(strings.ToLower(strings.TrimSpace(in.FormatHint)))
	if format != domain.ExportFormatJSON && format != domain.ExportFormatCSV {
		format = detectFormat(in.FileName, in.FileBytes)
	}
	parsed, err := parseExportFile(format, in.FileBytes, clientID)
	if err != nil {
		return nil, err
	}
	// Compute willAdd/duplicates against current data (merge baseline).
	counts, totals, err := s.computePreviewCounts(ctx, clientID, parsed, domain.ImportModeMerge)
	if err != nil {
		return nil, err
	}
	if err := s.pruneOldJobs(ctx, clientID); err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	exp := now.Add(s.fileTTL)
	id := uuid.NewString()
	clientDir := filepath.Join(s.fileDir, sanitizePathPart(clientID))
	if err := os.MkdirAll(clientDir, 0o755); err != nil {
		return nil, err
	}
	srcPath := filepath.Join(clientDir, id+".src")
	if err := os.WriteFile(srcPath, in.FileBytes, 0o600); err != nil {
		return nil, err
	}
	payloadPath := filepath.Join(clientDir, id+".payload.json")
	pb, err := json.Marshal(parsed.Payload)
	if err != nil {
		return nil, err
	}
	if err := os.WriteFile(payloadPath, pb, 0o600); err != nil {
		return nil, err
	}
	name := strings.TrimSpace(in.FileName)
	if name == "" {
		name = "export." + string(format)
	}
	job := domain.ImportJob{
		ID: id, ClientID: clientID, Format: format, Status: domain.ImportPreviewed,
		ProgressPct: 0, Stage: "preview", SectionCounts: counts, Totals: totals,
		FileName: name, FilePath: srcPath, PayloadPath: payloadPath,
		ByteSize: int64(len(in.FileBytes)), ExpiresAt: &exp, CreatedAt: now,
	}
	created, err := s.store.Create(ctx, job)
	if err != nil {
		return nil, err
	}
	// Stats already in Create; ensure payload path stored.
	_ = s.store.UpdatePreviewStats(ctx, id, counts, totals, payloadPath)
	return created, nil
}

func (s *Service) computePreviewCounts(ctx context.Context, clientID string, parsed *parseResult, mode domain.ImportMode) (map[domain.ExportSection]domain.ImportSectionCount, domain.ImportPreviewTotals, error) {
	counts := map[domain.ExportSection]domain.ImportSectionCount{}
	// Watchlist
	{
		c := domain.ImportSectionCount{
			Valid:      len(parsed.Payload.WatchlistItems),
			Invalid:    parsed.Invalid[domain.ExportSectionWatchlist],
			Duplicates: parsed.FileDuplicates[domain.ExportSectionWatchlist],
		}
		if mode == domain.ImportModeReplace {
			c.WillAdd = c.Valid
		} else {
			existing := map[string]struct{}{}
			if s.data.Watchlist != nil {
				wl, err := s.data.Watchlist.Get(ctx, clientID)
				if err == nil && wl != nil {
					for _, it := range wl.Items {
						existing[string(it.Exchange)+"|"+it.Symbol] = struct{}{}
					}
				}
			}
			for _, it := range parsed.Payload.WatchlistItems {
				key := string(it.Exchange) + "|" + it.Symbol
				if _, ok := existing[key]; ok {
					c.Duplicates++
				} else {
					c.WillAdd++
				}
			}
		}
		counts[domain.ExportSectionWatchlist] = c
	}
	// Shares
	{
		c := domain.ImportSectionCount{
			Valid:      len(parsed.Payload.Shares),
			Invalid:    parsed.Invalid[domain.ExportSectionShares],
			Duplicates: parsed.FileDuplicates[domain.ExportSectionShares],
		}
		if mode == domain.ImportModeReplace {
			c.WillAdd = c.Valid
		} else {
			existing := map[string]struct{}{}
			if s.data.Watchlist != nil {
				list, err := s.data.Watchlist.ListSharesByOwner(ctx, clientID)
				if err == nil {
					for _, sh := range list {
						existing[sh.GranteeClientID] = struct{}{}
					}
				}
			}
			for _, sh := range parsed.Payload.Shares {
				if _, ok := existing[sh.GranteeClientID]; ok {
					c.Duplicates++
				} else {
					c.WillAdd++
				}
			}
		}
		counts[domain.ExportSectionShares] = c
	}
	// Alerts
	{
		c := domain.ImportSectionCount{
			Valid:      len(parsed.Payload.Alerts),
			Invalid:    parsed.Invalid[domain.ExportSectionAlerts],
			Duplicates: parsed.FileDuplicates[domain.ExportSectionAlerts],
		}
		if mode == domain.ImportModeReplace {
			c.WillAdd = c.Valid
		} else {
			existing := map[string]struct{}{}
			if s.data.Alerts != nil {
				list, err := s.data.Alerts.ListByClient(ctx, clientID)
				if err == nil {
					for _, a := range list {
						existing[a.ID] = struct{}{}
					}
				}
			}
			for _, a := range parsed.Payload.Alerts {
				if _, ok := existing[a.ID]; ok {
					c.Duplicates++
				} else {
					c.WillAdd++
				}
			}
		}
		counts[domain.ExportSectionAlerts] = c
	}
	// Backtests
	{
		c := domain.ImportSectionCount{
			Valid:      len(parsed.Payload.Backtests),
			Invalid:    parsed.Invalid[domain.ExportSectionBacktests],
			Duplicates: parsed.FileDuplicates[domain.ExportSectionBacktests],
		}
		if mode == domain.ImportModeReplace {
			c.WillAdd = c.Valid
		} else {
			existing := map[string]struct{}{}
			if s.data.Scanner != nil {
				list, err := s.data.Scanner.ListBacktests(ctx, clientID, 500, 0)
				if err == nil {
					for _, b := range list {
						existing[b.ID] = struct{}{}
					}
				}
			}
			for _, b := range parsed.Payload.Backtests {
				if _, ok := existing[b.Job.ID]; ok {
					c.Duplicates++
				} else {
					c.WillAdd++
				}
			}
		}
		counts[domain.ExportSectionBacktests] = c
	}
	var totals domain.ImportPreviewTotals
	for _, c := range counts {
		totals.Valid += c.Valid
		totals.Invalid += c.Invalid
		totals.WillAdd += c.WillAdd
		totals.Duplicates += c.Duplicates
	}
	return counts, totals, nil
}

// Confirm starts applying a previewed import with the given mode.
func (s *Service) Confirm(ctx context.Context, clientID, id, mode string) (*domain.ImportJob, error) {
	if s.store == nil {
		return nil, fmt.Errorf("%w: import store not configured", domain.ErrUpstream)
	}
	clientID, err := normalizeClientID(clientID)
	if err != nil {
		return nil, err
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, fmt.Errorf("%w: import id is required", domain.ErrInvalidArgument)
	}
	m, err := domain.NormalizeImportMode(mode)
	if err != nil {
		return nil, err
	}
	if active, err := s.store.FindActiveApply(ctx, clientID); err == nil && active != nil {
		return nil, fmt.Errorf("%w: an import is already in progress (id=%s)", domain.ErrConflict, active.ID)
	} else if err != nil && err != domain.ErrNotFound {
		return nil, err
	}
	// Recompute willAdd under chosen mode for accurate preview before queue.
	job, err := s.store.Get(ctx, clientID, id)
	if err != nil {
		return nil, err
	}
	if job.Status != domain.ImportPreviewed {
		return nil, fmt.Errorf("%w: import is not awaiting confirm (status=%s)", domain.ErrConflict, job.Status)
	}
	pl, err := s.loadPayload(job)
	if err != nil {
		return nil, err
	}
	// Rebuild parseResult-like for counts
	parsed := &parseResult{Payload: *pl, Invalid: map[domain.ExportSection]int{}, FileDuplicates: map[domain.ExportSection]int{}}
	// Keep invalid from stored totals as-is; recompute willAdd
	counts, totals, err := s.computePreviewCounts(ctx, clientID, parsed, m)
	if err != nil {
		return nil, err
	}
	// Preserve invalid counts from original preview
	for sec, c := range job.SectionCounts {
		cc := counts[sec]
		cc.Invalid = c.Invalid
		counts[sec] = cc
	}
	totals.Invalid = job.Totals.Invalid
	_ = s.store.UpdatePreviewStats(ctx, id, counts, totals, job.PayloadPath)
	return s.store.Confirm(ctx, clientID, id, m, time.Now().UTC())
}

func (s *Service) loadPayload(job *domain.ImportJob) (*payload, error) {
	if job.PayloadPath == "" {
		return nil, fmt.Errorf("%w: import payload missing", domain.ErrNotFound)
	}
	raw, err := os.ReadFile(job.PayloadPath)
	if err != nil {
		return nil, fmt.Errorf("%w: import payload not found", domain.ErrNotFound)
	}
	var pl payload
	if err := json.Unmarshal(raw, &pl); err != nil {
		return nil, fmt.Errorf("%w: corrupt import payload", domain.ErrInvalidArgument)
	}
	return &pl, nil
}

// Get returns an import job for the owner.
func (s *Service) Get(ctx context.Context, clientID, id string) (*domain.ImportJob, error) {
	if s.store == nil {
		return nil, fmt.Errorf("%w: import store not configured", domain.ErrUpstream)
	}
	clientID, err := normalizeClientID(clientID)
	if err != nil {
		return nil, err
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, fmt.Errorf("%w: import id is required", domain.ErrInvalidArgument)
	}
	return s.store.Get(ctx, clientID, id)
}

// List returns recent imports.
func (s *Service) List(ctx context.Context, clientID string, limit, offset int) ([]domain.ImportJob, error) {
	if s.store == nil {
		return nil, fmt.Errorf("%w: import store not configured", domain.ErrUpstream)
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

// Cancel cancels previewed, pending, or running imports.
func (s *Service) Cancel(ctx context.Context, clientID, id string) (*domain.ImportJob, error) {
	if s.store == nil {
		return nil, fmt.Errorf("%w: import store not configured", domain.ErrUpstream)
	}
	clientID, err := normalizeClientID(clientID)
	if err != nil {
		return nil, err
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, fmt.Errorf("%w: import id is required", domain.ErrInvalidArgument)
	}
	return s.store.Cancel(ctx, clientID, id, time.Now().UTC())
}

// CleanupExpired removes expired jobs and their files.
func (s *Service) CleanupExpired(ctx context.Context) (int, error) {
	if s.store == nil {
		return 0, nil
	}
	list, err := s.store.ListExpired(ctx, time.Now().UTC(), 100)
	if err != nil {
		return 0, err
	}
	n := 0
	for i := range list {
		if err := s.deleteJobAndFiles(ctx, &list[i]); err == nil {
			n++
		}
	}
	return n, nil
}

func (s *Service) deleteJobAndFiles(ctx context.Context, j *domain.ImportJob) error {
	if j.FilePath != "" {
		_ = os.Remove(j.FilePath)
	}
	if j.PayloadPath != "" {
		_ = os.Remove(j.PayloadPath)
	}
	return s.store.Delete(ctx, j.ID)
}

func (s *Service) pruneOldJobs(ctx context.Context, clientID string) error {
	n, err := s.store.CountByClient(ctx, clientID)
	if err != nil {
		return err
	}
	if n < domain.MaxImportsPerClient {
		return nil
	}
	list, err := s.store.ListByClient(ctx, clientID, n, 0)
	if err != nil {
		return err
	}
	for i := len(list) - 1; i >= 0 && n >= domain.MaxImportsPerClient; i-- {
		j := list[i]
		if j.IsActiveApply() || j.Status == domain.ImportPreviewed {
			continue
		}
		_ = s.deleteJobAndFiles(ctx, &j)
		n--
	}
	return nil
}

// RequeueStuckRunning recovers interrupted applies.
func (s *Service) RequeueStuckRunning(ctx context.Context, olderThan time.Duration) (int, error) {
	if s.store == nil {
		return 0, nil
	}
	if olderThan <= 0 {
		olderThan = 2 * time.Minute
	}
	return s.store.RequeueStuckRunning(ctx, time.Now().UTC().Add(-olderThan))
}

// ProcessPending claims and runs all pending imports once.
func (s *Service) ProcessPending(ctx context.Context) (int, error) {
	if s.store == nil {
		return 0, nil
	}
	pending, err := s.store.ListPending(ctx)
	if err != nil {
		return 0, err
	}
	n := 0
	for i := range pending {
		job := &pending[i]
		ok, err := s.store.Claim(ctx, job.ID, time.Now().UTC())
		if err != nil {
			return n, err
		}
		if !ok {
			continue
		}
		claimed, err := s.store.GetByID(ctx, job.ID)
		if err != nil {
			continue
		}
		if err := s.runJob(ctx, claimed); err != nil {
			_ = s.store.Finish(ctx, job.ID, domain.ImportFailed, nil, err.Error(), time.Now().UTC())
		}
		n++
	}
	return n, nil
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

func sanitizePathPart(s string) string {
	var b strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' || r == '.' {
			b.WriteRune(r)
		} else {
			b.WriteByte('_')
		}
	}
	out := b.String()
	if out == "" {
		return "_empty"
	}
	return out
}

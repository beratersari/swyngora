package domain

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// Export lifecycle limits.
const (
	// MaxExportsPerClient caps retained export jobs per client (completed + failed + canceled).
	MaxExportsPerClient = 20
	// DefaultExportFileTTL is how long completed files remain downloadable.
	DefaultExportFileTTL = 1 * time.Hour
	// MaxExportFileTTL caps configurable TTL.
	MaxExportFileTTL = 24 * time.Hour
	// MinExportFileTTL is the minimum retention for completed files.
	MinExportFileTTL = 5 * time.Minute
)

// ExportFormat is the download payload encoding.
type ExportFormat string

const (
	ExportFormatJSON ExportFormat = "json"
	ExportFormatCSV  ExportFormat = "csv"
)

// ExportStatus is the lifecycle of a user data export job.
type ExportStatus string

const (
	ExportPending   ExportStatus = "pending"
	ExportRunning   ExportStatus = "running"
	ExportCompleted ExportStatus = "completed"
	ExportCanceled  ExportStatus = "canceled"
	ExportFailed    ExportStatus = "failed"
)

// ExportSection names a slice of user data included in an export.
type ExportSection string

const (
	ExportSectionWatchlist  ExportSection = "watchlist"
	ExportSectionShares     ExportSection = "shares"
	ExportSectionAlerts     ExportSection = "alerts"
	ExportSectionBacktests  ExportSection = "backtests"
	ExportSectionPortfolios ExportSection = "portfolios"
)

// AllExportSections is the default set when the client omits sections.
var AllExportSections = []ExportSection{
	ExportSectionWatchlist,
	ExportSectionShares,
	ExportSectionAlerts,
	ExportSectionBacktests,
	ExportSectionPortfolios,
}

// ExportJob is a background (or fast) user data export.
type ExportJob struct {
	ID           string
	ClientID     string
	Format       ExportFormat
	Sections     []ExportSection
	Status       ExportStatus
	ProgressPct  float64 // 0–100
	Stage        string  // current section or phase label
	ErrorMessage string
	FileName     string
	FilePath     string // absolute path on disk (not exposed to clients)
	ByteSize     int64
	ExpiresAt    *time.Time
	CreatedAt    time.Time
	StartedAt    *time.Time
	FinishedAt   *time.Time
}

// IsTerminal reports whether the job will not change further (except expiry cleanup).
func (j ExportJob) IsTerminal() bool {
	switch j.Status {
	case ExportCompleted, ExportCanceled, ExportFailed:
		return true
	default:
		return false
	}
}

// IsActive reports pending or running (blocks a second concurrent start for the client).
func (j ExportJob) IsActive() bool {
	return j.Status == ExportPending || j.Status == ExportRunning
}

// IsValidExportFormat reports json|csv.
func IsValidExportFormat(s string) bool {
	switch ExportFormat(strings.ToLower(strings.TrimSpace(s))) {
	case ExportFormatJSON, ExportFormatCSV:
		return true
	default:
		return false
	}
}

// NormalizeExportFormat parses json|csv.
func NormalizeExportFormat(s string) (ExportFormat, error) {
	f := ExportFormat(strings.ToLower(strings.TrimSpace(s)))
	if !IsValidExportFormat(string(f)) {
		return "", fmt.Errorf("%w: format must be json or csv", ErrInvalidArgument)
	}
	return f, nil
}

// IsValidExportSection reports a known section name.
func IsValidExportSection(s string) bool {
	switch ExportSection(strings.ToLower(strings.TrimSpace(s))) {
	case ExportSectionWatchlist, ExportSectionShares, ExportSectionAlerts, ExportSectionBacktests, ExportSectionPortfolios:
		return true
	default:
		return false
	}
}

// NormalizeExportSections parses and de-duplicates sections; empty → all.
func NormalizeExportSections(raw []string) ([]ExportSection, error) {
	if len(raw) == 0 {
		out := make([]ExportSection, len(AllExportSections))
		copy(out, AllExportSections)
		return out, nil
	}
	seen := map[ExportSection]struct{}{}
	out := make([]ExportSection, 0, len(raw))
	for _, s := range raw {
		s = strings.ToLower(strings.TrimSpace(s))
		if !IsValidExportSection(s) {
			return nil, fmt.Errorf("%w: unknown export section %q (watchlist|shares|alerts|backtests|portfolios)", ErrInvalidArgument, s)
		}
		sec := ExportSection(s)
		if _, ok := seen[sec]; ok {
			continue
		}
		seen[sec] = struct{}{}
		out = append(out, sec)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("%w: at least one export section is required", ErrInvalidArgument)
	}
	return out, nil
}

// SectionsCSV joins sections for storage.
func SectionsCSV(secs []ExportSection) string {
	parts := make([]string, len(secs))
	for i, s := range secs {
		parts[i] = string(s)
	}
	return strings.Join(parts, ",")
}

// ParseSectionsCSV splits a stored sections list.
func ParseSectionsCSV(s string) []ExportSection {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	var out []ExportSection
	for _, p := range strings.Split(s, ",") {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, ExportSection(p))
		}
	}
	return out
}

// ExportPort persists export jobs. File bytes live on disk under a configured directory.
// Implementations must be concurrent-safe.
type ExportPort interface {
	Create(ctx context.Context, job ExportJob) (*ExportJob, error)
	Get(ctx context.Context, clientID, id string) (*ExportJob, error)
	// GetByID is for the worker (no client filter).
	GetByID(ctx context.Context, id string) (*ExportJob, error)
	ListByClient(ctx context.Context, clientID string, limit, offset int) ([]ExportJob, error)
	CountByClient(ctx context.Context, clientID string) (int, error)
	// FindActive returns the pending/running job for a client, or ErrNotFound.
	FindActive(ctx context.Context, clientID string) (*ExportJob, error)
	ListPending(ctx context.Context) ([]ExportJob, error)
	// Claim moves pending → running if still pending. Returns false if not claimed.
	Claim(ctx context.Context, id string, at time.Time) (bool, error)
	UpdateProgress(ctx context.Context, id string, progressPct float64, stage string) error
	// Finish sets a terminal status and optional file metadata.
	Finish(ctx context.Context, id string, status ExportStatus, fileName, filePath string, byteSize int64, expiresAt *time.Time, errMsg string, at time.Time) error
	// Cancel sets canceled if pending or running and owned by clientID.
	Cancel(ctx context.Context, clientID, id string, at time.Time) (*ExportJob, error)
	// GetStatus returns only status (cancel checks during run).
	GetStatus(ctx context.Context, id string) (ExportStatus, error)
	// ListExpiredCompleted returns completed jobs with expires_at <= now.
	ListExpiredCompleted(ctx context.Context, now time.Time, limit int) ([]ExportJob, error)
	// Delete removes the job row (caller deletes the file).
	Delete(ctx context.Context, id string) error
	// RequeueStuckRunning moves running jobs older than before back to pending (restart recovery).
	RequeueStuckRunning(ctx context.Context, before time.Time) (int, error)
	// PurgeClient deletes all export jobs (and caller should remove files).
	PurgeClient(ctx context.Context, clientID string) ([]ExportJob, error)
}

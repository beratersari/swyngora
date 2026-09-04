package domain

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// Import lifecycle limits.
const (
	MaxImportsPerClient     = 20
	DefaultImportFileTTL    = 1 * time.Hour
	MaxImportFileTTL        = 24 * time.Hour
	MinImportFileTTL        = 5 * time.Minute
	MaxImportUploadBytes    = 20 << 20 // 20 MiB
)

// ImportMode controls how valid records are applied to existing data.
type ImportMode string

const (
	// ImportModeMerge adds only missing records (no duplicates).
	ImportModeMerge ImportMode = "merge"
	// ImportModeReplace clears existing section data for the client, then imports from the file.
	ImportModeReplace ImportMode = "replace"
)

// ImportStatus is the lifecycle of a user data import job.
type ImportStatus string

const (
	// ImportPreviewed: file uploaded and validated; not applied yet.
	ImportPreviewed ImportStatus = "previewed"
	ImportPending   ImportStatus = "pending"
	ImportRunning   ImportStatus = "running"
	ImportCompleted ImportStatus = "completed"
	ImportCanceled  ImportStatus = "canceled"
	ImportFailed    ImportStatus = "failed"
)

// ImportSectionCount is per-section stats from preview / result.
type ImportSectionCount struct {
	Valid        int `json:"valid"`
	Invalid      int `json:"invalid"`
	WillAdd      int `json:"willAdd"`      // valid records that would be inserted under the chosen mode
	WillUpdate   int `json:"willUpdate"`   // valid records that match exactly one existing name
	WillConflict int `json:"willConflict"` // custom-signal names that need a user pick
	Duplicates   int `json:"duplicates"`   // valid but already present (merge skip) or duplicates within file
	// AddNames / UpdateNames list custom-signal profile names in the preview
	// (empty for other sections). Sorted for stable responses.
	AddNames    []string                `json:"addNames,omitempty"`
	UpdateNames []string                `json:"updateNames,omitempty"`
	Conflicts   []ImportProfileConflict `json:"conflicts,omitempty"`
	// DeskCandidates is the preview snapshot of saved mixes whose names appear
	// in the file. Confirm compares this to the live desk and rejects if it changed.
	DeskCandidates []ImportProfileCandidate `json:"deskCandidates,omitempty"`
}

// ImportProfileIncoming is one file row in a same-name conflict group.
type ImportProfileIncoming struct {
	Index     int                `json:"index"`
	IsDefault bool               `json:"isDefault,omitempty"`
	Weights   map[string]float64 `json:"weights,omitempty"`
}

// ImportProfileCandidate is an existing saved mix the user can update.
type ImportProfileCandidate struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	IsDefault bool   `json:"isDefault,omitempty"`
	UpdatedAt string `json:"updatedAt,omitempty"`
}

// ImportProfileConflict is a custom-signal name with more than one file row
// and/or more than one existing mix. Confirm must pick incomingIndex and/or
// profileId; omitting the name skips that group.
type ImportProfileConflict struct {
	Name       string                    `json:"name"`
	Incoming   []ImportProfileIncoming   `json:"incoming"`
	Candidates []ImportProfileCandidate  `json:"candidates"`
}

// CustomSignalResolution is the user's pick for one conflicted name.
// Empty ProfileID or a missing IncomingIndex (when required) means skip.
type CustomSignalResolution struct {
	Name          string `json:"name"`
	IncomingIndex *int   `json:"incomingIndex,omitempty"`
	ProfileID     string `json:"profileId,omitempty"`
}

// ImportPreviewTotals aggregates section counts.
type ImportPreviewTotals struct {
	Valid        int `json:"valid"`
	Invalid      int `json:"invalid"`
	WillAdd      int `json:"willAdd"`
	WillUpdate   int `json:"willUpdate"`
	WillConflict int `json:"willConflict"`
	Duplicates   int `json:"duplicates"`
}

// CloneImportSectionCount copies name and conflict slices so store clones cannot mutate the source.
func CloneImportSectionCount(v ImportSectionCount) ImportSectionCount {
	if len(v.AddNames) > 0 {
		v.AddNames = append([]string(nil), v.AddNames...)
	}
	if len(v.UpdateNames) > 0 {
		v.UpdateNames = append([]string(nil), v.UpdateNames...)
	}
	if len(v.Conflicts) > 0 {
		cp := make([]ImportProfileConflict, len(v.Conflicts))
		for i, c := range v.Conflicts {
			cp[i] = cloneImportProfileConflict(c)
		}
		v.Conflicts = cp
	}
	if len(v.DeskCandidates) > 0 {
		v.DeskCandidates = append([]ImportProfileCandidate(nil), v.DeskCandidates...)
	}
	return v
}

func cloneImportProfileConflict(c ImportProfileConflict) ImportProfileConflict {
	if len(c.Incoming) > 0 {
		inc := make([]ImportProfileIncoming, len(c.Incoming))
		for i, in := range c.Incoming {
			inc[i] = in
			if in.Weights != nil {
				w := make(map[string]float64, len(in.Weights))
				for k, val := range in.Weights {
					w[k] = val
				}
				inc[i].Weights = w
			}
		}
		c.Incoming = inc
	}
	if len(c.Candidates) > 0 {
		c.Candidates = append([]ImportProfileCandidate(nil), c.Candidates...)
	}
	return c
}

// ImportJob tracks an upload, preview, and optional background apply.
type ImportJob struct {
	ID           string
	ClientID     string
	Format       ExportFormat // json | csv (same encodings as export)
	Mode         ImportMode   // empty until confirm
	Status       ImportStatus
	ProgressPct  float64
	Stage        string
	ErrorMessage string
	// SectionCounts is filled at preview (and refreshed after apply for willAdd actuals).
	SectionCounts map[ExportSection]ImportSectionCount
	Totals        ImportPreviewTotals
	// AddedCounts is filled on completion (records actually inserted).
	AddedCounts map[ExportSection]int
	FileName    string
	FilePath    string // uploaded source file
	// PayloadPath is the normalized JSON payload used for apply (after parse).
	PayloadPath  string
	ByteSize     int64
	ExpiresAt    *time.Time
	CreatedAt    time.Time
	StartedAt    *time.Time
	FinishedAt   *time.Time
}

// IsTerminal reports terminal status.
func (j ImportJob) IsTerminal() bool {
	switch j.Status {
	case ImportCompleted, ImportCanceled, ImportFailed:
		return true
	default:
		return false
	}
}

// IsActiveApply reports pending or running apply (blocks another confirm).
func (j ImportJob) IsActiveApply() bool {
	return j.Status == ImportPending || j.Status == ImportRunning
}

// IsValidImportMode reports merge|replace.
func IsValidImportMode(s string) bool {
	switch ImportMode(strings.ToLower(strings.TrimSpace(s))) {
	case ImportModeMerge, ImportModeReplace:
		return true
	default:
		return false
	}
}

// NormalizeImportMode parses merge|replace.
func NormalizeImportMode(s string) (ImportMode, error) {
	m := ImportMode(strings.ToLower(strings.TrimSpace(s)))
	if !IsValidImportMode(string(m)) {
		return "", fmt.Errorf("%w: mode must be merge or replace", ErrInvalidArgument)
	}
	return m, nil
}

// ImportPort persists import jobs. Source files live under a configured directory.
type ImportPort interface {
	Create(ctx context.Context, job ImportJob) (*ImportJob, error)
	Get(ctx context.Context, clientID, id string) (*ImportJob, error)
	GetByID(ctx context.Context, id string) (*ImportJob, error)
	ListByClient(ctx context.Context, clientID string, limit, offset int) ([]ImportJob, error)
	CountByClient(ctx context.Context, clientID string) (int, error)
	// FindActiveApply returns pending/running job for client, or ErrNotFound.
	FindActiveApply(ctx context.Context, clientID string) (*ImportJob, error)
	ListPending(ctx context.Context) ([]ImportJob, error)
	Claim(ctx context.Context, id string, at time.Time) (bool, error)
	UpdateProgress(ctx context.Context, id string, progressPct float64, stage string) error
	// Confirm sets mode and moves previewed → pending.
	Confirm(ctx context.Context, clientID, id string, mode ImportMode, at time.Time) (*ImportJob, error)
	// UpdatePreviewStats stores section counts after parse.
	UpdatePreviewStats(ctx context.Context, id string, counts map[ExportSection]ImportSectionCount, totals ImportPreviewTotals, payloadPath string) error
	Finish(ctx context.Context, id string, status ImportStatus, added map[ExportSection]int, errMsg string, at time.Time) error
	Cancel(ctx context.Context, clientID, id string, at time.Time) (*ImportJob, error)
	GetStatus(ctx context.Context, id string) (ImportStatus, error)
	ListExpired(ctx context.Context, now time.Time, limit int) ([]ImportJob, error)
	Delete(ctx context.Context, id string) error
	RequeueStuckRunning(ctx context.Context, before time.Time) (int, error)
	// PurgeClient deletes all import jobs (and caller should remove files).
	PurgeClient(ctx context.Context, clientID string) ([]ImportJob, error)
}

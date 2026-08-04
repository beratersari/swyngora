package exportstore

import (
	"context"
	"sync"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

// Memory is an in-process export job store for tests.
type Memory struct {
	mu   sync.Mutex
	byID map[string]*domain.ExportJob
}

// NewMemory constructs an empty store.
func NewMemory() *Memory {
	return &Memory{byID: map[string]*domain.ExportJob{}}
}

func cloneJob(j *domain.ExportJob) *domain.ExportJob {
	if j == nil {
		return nil
	}
	cp := *j
	if j.Sections != nil {
		cp.Sections = append([]domain.ExportSection(nil), j.Sections...)
	}
	if j.ExpiresAt != nil {
		t := j.ExpiresAt.UTC()
		cp.ExpiresAt = &t
	}
	if j.StartedAt != nil {
		t := j.StartedAt.UTC()
		cp.StartedAt = &t
	}
	if j.FinishedAt != nil {
		t := j.FinishedAt.UTC()
		cp.FinishedAt = &t
	}
	return &cp
}

// Create inserts a job.
func (m *Memory) Create(_ context.Context, job domain.ExportJob) (*domain.ExportJob, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	cp := cloneJob(&job)
	m.byID[cp.ID] = cp
	return cloneJob(cp), nil
}

// Get returns a job owned by clientID.
func (m *Memory) Get(_ context.Context, clientID, id string) (*domain.ExportJob, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	j, ok := m.byID[id]
	if !ok || j.ClientID != clientID {
		return nil, domain.ErrNotFound
	}
	return cloneJob(j), nil
}

// GetByID returns a job by id.
func (m *Memory) GetByID(_ context.Context, id string) (*domain.ExportJob, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	j, ok := m.byID[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return cloneJob(j), nil
}

// ListByClient lists newest-first.
func (m *Memory) ListByClient(_ context.Context, clientID string, limit, offset int) ([]domain.ExportJob, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var all []domain.ExportJob
	for _, j := range m.byID {
		if j.ClientID == clientID {
			all = append(all, *cloneJob(j))
		}
	}
	// newest first
	for i := 0; i < len(all); i++ {
		for k := i + 1; k < len(all); k++ {
			if all[k].CreatedAt.After(all[i].CreatedAt) {
				all[i], all[k] = all[k], all[i]
			}
		}
	}
	if offset > len(all) {
		return []domain.ExportJob{}, nil
	}
	all = all[offset:]
	if limit > 0 && len(all) > limit {
		all = all[:limit]
	}
	return all, nil
}

// CountByClient counts jobs.
func (m *Memory) CountByClient(_ context.Context, clientID string) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	n := 0
	for _, j := range m.byID {
		if j.ClientID == clientID {
			n++
		}
	}
	return n, nil
}

// FindActive returns pending/running job for client.
func (m *Memory) FindActive(_ context.Context, clientID string) (*domain.ExportJob, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, j := range m.byID {
		if j.ClientID == clientID && j.IsActive() {
			return cloneJob(j), nil
		}
	}
	return nil, domain.ErrNotFound
}

// ListPending returns pending jobs oldest-first.
func (m *Memory) ListPending(_ context.Context) ([]domain.ExportJob, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []domain.ExportJob
	for _, j := range m.byID {
		if j.Status == domain.ExportPending {
			out = append(out, *cloneJob(j))
		}
	}
	for i := 0; i < len(out); i++ {
		for k := i + 1; k < len(out); k++ {
			if out[k].CreatedAt.Before(out[i].CreatedAt) {
				out[i], out[k] = out[k], out[i]
			}
		}
	}
	return out, nil
}

// Claim moves pending → running.
func (m *Memory) Claim(_ context.Context, id string, at time.Time) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	j, ok := m.byID[id]
	if !ok || j.Status != domain.ExportPending {
		return false, nil
	}
	j.Status = domain.ExportRunning
	t := at.UTC()
	j.StartedAt = &t
	j.Stage = "starting"
	return true, nil
}

// UpdateProgress updates progress fields.
func (m *Memory) UpdateProgress(_ context.Context, id string, progressPct float64, stage string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	j, ok := m.byID[id]
	if !ok {
		return domain.ErrNotFound
	}
	j.ProgressPct = progressPct
	j.Stage = stage
	return nil
}

// Finish sets terminal status.
func (m *Memory) Finish(_ context.Context, id string, status domain.ExportStatus, fileName, filePath string, byteSize int64, expiresAt *time.Time, errMsg string, at time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	j, ok := m.byID[id]
	if !ok {
		return domain.ErrNotFound
	}
	j.Status = status
	j.FileName = fileName
	j.FilePath = filePath
	j.ByteSize = byteSize
	j.ErrorMessage = errMsg
	if expiresAt != nil {
		t := expiresAt.UTC()
		j.ExpiresAt = &t
	}
	t := at.UTC()
	j.FinishedAt = &t
	if status == domain.ExportCompleted {
		j.ProgressPct = 100
		j.Stage = "done"
	}
	return nil
}

// Cancel cancels pending/running job.
func (m *Memory) Cancel(_ context.Context, clientID, id string, at time.Time) (*domain.ExportJob, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	j, ok := m.byID[id]
	if !ok || j.ClientID != clientID {
		return nil, domain.ErrNotFound
	}
	if j.IsTerminal() {
		return cloneJob(j), nil
	}
	if !j.IsActive() {
		return nil, domain.ErrNotFound
	}
	j.Status = domain.ExportCanceled
	t := at.UTC()
	j.FinishedAt = &t
	j.Stage = "canceled"
	return cloneJob(j), nil
}

// GetStatus returns status only.
func (m *Memory) GetStatus(_ context.Context, id string) (domain.ExportStatus, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	j, ok := m.byID[id]
	if !ok {
		return "", domain.ErrNotFound
	}
	return j.Status, nil
}

// ListExpiredCompleted returns expired completed jobs.
func (m *Memory) ListExpiredCompleted(_ context.Context, now time.Time, limit int) ([]domain.ExportJob, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []domain.ExportJob
	for _, j := range m.byID {
		if j.Status == domain.ExportCompleted && j.ExpiresAt != nil && !j.ExpiresAt.After(now) {
			out = append(out, *cloneJob(j))
			if limit > 0 && len(out) >= limit {
				break
			}
		}
	}
	return out, nil
}

// Delete removes a job.
func (m *Memory) Delete(_ context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.byID[id]; !ok {
		return domain.ErrNotFound
	}
	delete(m.byID, id)
	return nil
}

// RequeueStuckRunning resets old running jobs to pending.
func (m *Memory) RequeueStuckRunning(_ context.Context, before time.Time) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	n := 0
	for _, j := range m.byID {
		if j.Status == domain.ExportRunning && j.StartedAt != nil && j.StartedAt.Before(before) {
			j.Status = domain.ExportPending
			j.StartedAt = nil
			j.Stage = "requeued"
			n++
		}
	}
	return n, nil
}

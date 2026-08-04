package importstore

import (
	"context"
	"sync"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
)

// Memory is an in-process import job store for tests.
type Memory struct {
	mu   sync.Mutex
	byID map[string]*domain.ImportJob
}

// NewMemory constructs an empty store.
func NewMemory() *Memory {
	return &Memory{byID: map[string]*domain.ImportJob{}}
}

func cloneImport(j *domain.ImportJob) *domain.ImportJob {
	if j == nil {
		return nil
	}
	cp := *j
	if j.SectionCounts != nil {
		cp.SectionCounts = make(map[domain.ExportSection]domain.ImportSectionCount, len(j.SectionCounts))
		for k, v := range j.SectionCounts {
			cp.SectionCounts[k] = v
		}
	}
	if j.AddedCounts != nil {
		cp.AddedCounts = make(map[domain.ExportSection]int, len(j.AddedCounts))
		for k, v := range j.AddedCounts {
			cp.AddedCounts[k] = v
		}
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

func (m *Memory) Create(_ context.Context, job domain.ImportJob) (*domain.ImportJob, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	cp := cloneImport(&job)
	m.byID[cp.ID] = cp
	return cloneImport(cp), nil
}

func (m *Memory) Get(_ context.Context, clientID, id string) (*domain.ImportJob, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	j, ok := m.byID[id]
	if !ok || j.ClientID != clientID {
		return nil, domain.ErrNotFound
	}
	return cloneImport(j), nil
}

func (m *Memory) GetByID(_ context.Context, id string) (*domain.ImportJob, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	j, ok := m.byID[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return cloneImport(j), nil
}

func (m *Memory) ListByClient(_ context.Context, clientID string, limit, offset int) ([]domain.ImportJob, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var all []domain.ImportJob
	for _, j := range m.byID {
		if j.ClientID == clientID {
			all = append(all, *cloneImport(j))
		}
	}
	for i := 0; i < len(all); i++ {
		for k := i + 1; k < len(all); k++ {
			if all[k].CreatedAt.After(all[i].CreatedAt) {
				all[i], all[k] = all[k], all[i]
			}
		}
	}
	if offset > len(all) {
		return []domain.ImportJob{}, nil
	}
	all = all[offset:]
	if limit > 0 && len(all) > limit {
		all = all[:limit]
	}
	return all, nil
}

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

func (m *Memory) FindActiveApply(_ context.Context, clientID string) (*domain.ImportJob, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, j := range m.byID {
		if j.ClientID == clientID && j.IsActiveApply() {
			return cloneImport(j), nil
		}
	}
	return nil, domain.ErrNotFound
}

func (m *Memory) ListPending(_ context.Context) ([]domain.ImportJob, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []domain.ImportJob
	for _, j := range m.byID {
		if j.Status == domain.ImportPending {
			out = append(out, *cloneImport(j))
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

func (m *Memory) Claim(_ context.Context, id string, at time.Time) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	j, ok := m.byID[id]
	if !ok || j.Status != domain.ImportPending {
		return false, nil
	}
	j.Status = domain.ImportRunning
	t := at.UTC()
	j.StartedAt = &t
	j.Stage = "starting"
	return true, nil
}

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

func (m *Memory) Confirm(_ context.Context, clientID, id string, mode domain.ImportMode, at time.Time) (*domain.ImportJob, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	j, ok := m.byID[id]
	if !ok || j.ClientID != clientID {
		return nil, domain.ErrNotFound
	}
	if j.Status != domain.ImportPreviewed {
		return nil, domain.ErrConflict
	}
	j.Mode = mode
	j.Status = domain.ImportPending
	j.Stage = "queued"
	return cloneImport(j), nil
}

func (m *Memory) UpdatePreviewStats(_ context.Context, id string, counts map[domain.ExportSection]domain.ImportSectionCount, totals domain.ImportPreviewTotals, payloadPath string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	j, ok := m.byID[id]
	if !ok {
		return domain.ErrNotFound
	}
	j.SectionCounts = make(map[domain.ExportSection]domain.ImportSectionCount, len(counts))
	for k, v := range counts {
		j.SectionCounts[k] = v
	}
	j.Totals = totals
	j.PayloadPath = payloadPath
	return nil
}

func (m *Memory) Finish(_ context.Context, id string, status domain.ImportStatus, added map[domain.ExportSection]int, errMsg string, at time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	j, ok := m.byID[id]
	if !ok {
		return domain.ErrNotFound
	}
	j.Status = status
	j.ErrorMessage = errMsg
	if added != nil {
		j.AddedCounts = make(map[domain.ExportSection]int, len(added))
		for k, v := range added {
			j.AddedCounts[k] = v
		}
	}
	t := at.UTC()
	j.FinishedAt = &t
	if status == domain.ImportCompleted {
		j.ProgressPct = 100
		j.Stage = "done"
	}
	return nil
}

func (m *Memory) Cancel(_ context.Context, clientID, id string, at time.Time) (*domain.ImportJob, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	j, ok := m.byID[id]
	if !ok || j.ClientID != clientID {
		return nil, domain.ErrNotFound
	}
	if j.IsTerminal() {
		return cloneImport(j), nil
	}
	switch j.Status {
	case domain.ImportPreviewed, domain.ImportPending, domain.ImportRunning:
		j.Status = domain.ImportCanceled
		t := at.UTC()
		j.FinishedAt = &t
		j.Stage = "canceled"
		return cloneImport(j), nil
	default:
		return nil, domain.ErrNotFound
	}
}

func (m *Memory) GetStatus(_ context.Context, id string) (domain.ImportStatus, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	j, ok := m.byID[id]
	if !ok {
		return "", domain.ErrNotFound
	}
	return j.Status, nil
}

func (m *Memory) ListExpired(_ context.Context, now time.Time, limit int) ([]domain.ImportJob, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []domain.ImportJob
	for _, j := range m.byID {
		if j.ExpiresAt != nil && !j.ExpiresAt.After(now) {
			out = append(out, *cloneImport(j))
			if limit > 0 && len(out) >= limit {
				break
			}
		}
	}
	return out, nil
}

func (m *Memory) Delete(_ context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.byID[id]; !ok {
		return domain.ErrNotFound
	}
	delete(m.byID, id)
	return nil
}

// PurgeClient removes all import jobs for clientID.
func (m *Memory) PurgeClient(_ context.Context, clientID string) ([]domain.ImportJob, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []domain.ImportJob
	for id, j := range m.byID {
		if j.ClientID == clientID {
			out = append(out, *cloneImport(j))
			delete(m.byID, id)
		}
	}
	return out, nil
}

func (m *Memory) RequeueStuckRunning(_ context.Context, before time.Time) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	n := 0
	for _, j := range m.byID {
		if j.Status == domain.ImportRunning && j.StartedAt != nil && j.StartedAt.Before(before) {
			j.Status = domain.ImportPending
			j.StartedAt = nil
			j.Stage = "requeued"
			n++
		}
	}
	return n, nil
}

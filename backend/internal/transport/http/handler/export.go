package handler

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
	exportsvc "gitlab.com/trace-analysis/swyngora/backend/internal/service/export"
)

// ExportHandler is the transport adapter for user data exports.
type ExportHandler struct {
	svc *exportsvc.Service
}

// NewExportHandler constructs the handler.
func NewExportHandler(svc *exportsvc.Service) *ExportHandler {
	return &ExportHandler{svc: svc}
}

type exportJobDTO struct {
	ID           string   `json:"id"`
	ClientID     string   `json:"clientId"`
	Format       string   `json:"format"`
	Sections     []string `json:"sections"`
	Status       string   `json:"status"`
	ProgressPct  float64  `json:"progressPct"`
	Stage        string   `json:"stage,omitempty"`
	ErrorMessage string   `json:"errorMessage,omitempty"`
	FileName     string   `json:"fileName,omitempty"`
	ByteSize     int64    `json:"byteSize,omitempty"`
	ExpiresAt    *string  `json:"expiresAt,omitempty"`
	CreatedAt    string   `json:"createdAt"`
	StartedAt    *string  `json:"startedAt,omitempty"`
	FinishedAt   *string  `json:"finishedAt,omitempty"`
	DownloadURL  string   `json:"downloadUrl,omitempty"`
}

func exportToDTO(j *domain.ExportJob) exportJobDTO {
	secs := make([]string, 0, len(j.Sections))
	for _, s := range j.Sections {
		secs = append(secs, string(s))
	}
	d := exportJobDTO{
		ID: j.ID, ClientID: j.ClientID, Format: string(j.Format), Sections: secs,
		Status: string(j.Status), ProgressPct: j.ProgressPct, Stage: j.Stage,
		ErrorMessage: j.ErrorMessage, FileName: j.FileName, ByteSize: j.ByteSize,
		CreatedAt: j.CreatedAt.UTC().Format(time.RFC3339Nano),
	}
	if j.ExpiresAt != nil {
		s := j.ExpiresAt.UTC().Format(time.RFC3339Nano)
		d.ExpiresAt = &s
	}
	if j.StartedAt != nil {
		s := j.StartedAt.UTC().Format(time.RFC3339Nano)
		d.StartedAt = &s
	}
	if j.FinishedAt != nil {
		s := j.FinishedAt.UTC().Format(time.RFC3339Nano)
		d.FinishedAt = &s
	}
	if j.Status == domain.ExportCompleted && j.FileName != "" {
		d.DownloadURL = "/api/v1/export/" + j.ID + "/download"
	}
	return d
}

type startExportBody struct {
	ClientID string   `json:"clientId"`
	Format   string   `json:"format"`
	Sections []string `json:"sections"`
}

// Start handles POST /api/v1/export
func (h *ExportHandler) Start(w http.ResponseWriter, r *http.Request) {
	var body startExportBody
	if err := decodeJSON(r, &body, DefaultMaxJSONBody); err != nil {
		writeError(w, fmt.Errorf("%w: invalid JSON body", domain.ErrInvalidArgument))
		return
	}
	clientID, ok := mustResolveClientID(w, r, body.ClientID)
	if !ok {
		return
	}
	format := body.Format
	if format == "" {
		format = "json"
	}
	job, err := h.svc.Start(r.Context(), exportsvc.StartInput{
		ClientID: clientID, Format: format, Sections: body.Sections,
	})
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusAccepted, exportToDTO(job))
}

// List handles GET /api/v1/export
func (h *ExportHandler) List(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	limit, _ := strconv.Atoi(q.Get("limit"))
	offset, _ := strconv.Atoi(q.Get("offset"))
	list, err := h.svc.List(r.Context(), clientIDFrom(r), limit, offset)
	if err != nil {
		writeError(w, err)
		return
	}
	items := make([]exportJobDTO, 0, len(list))
	for i := range list {
		items = append(items, exportToDTO(&list[i]))
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"clientId": clientIDFrom(r), "exports": items, "count": len(items),
	})
}

// Get handles GET /api/v1/export/{id}
func (h *ExportHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	job, err := h.svc.Get(r.Context(), clientIDFrom(r), id)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, exportToDTO(job))
}

// Cancel handles POST /api/v1/export/{id}/cancel
func (h *ExportHandler) Cancel(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	job, err := h.svc.Cancel(r.Context(), clientIDFrom(r), id)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, exportToDTO(job))
}

// Download handles GET /api/v1/export/{id}/download
// Only the owning clientId may download; file is deleted after TTL by the worker.
func (h *ExportHandler) Download(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	job, err := h.svc.OpenDownload(r.Context(), clientIDFrom(r), id)
	if err != nil {
		writeError(w, err)
		return
	}
	f, err := os.Open(job.FilePath)
	if err != nil {
		writeError(w, fmt.Errorf("%w: export file not found", domain.ErrNotFound))
		return
	}
	defer f.Close()

	ct := "application/json; charset=utf-8"
	if job.Format == domain.ExportFormatCSV {
		ct = "text/csv; charset=utf-8"
	}
	name := job.FileName
	if name == "" {
		name = filepath.Base(job.FilePath)
	}
	// Prevent path injection in Content-Disposition
	name = strings.ReplaceAll(name, `"`, "")
	w.Header().Set("Content-Type", ct)
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, name))
	if job.ByteSize > 0 {
		w.Header().Set("Content-Length", strconv.FormatInt(job.ByteSize, 10))
	}
	w.WriteHeader(http.StatusOK)
	_, _ = io.Copy(w, f)
}

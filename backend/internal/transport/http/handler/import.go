package handler

import (
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"gitlab.com/trace-analysis/swyngora/backend/internal/domain"
	"gitlab.com/trace-analysis/swyngora/backend/internal/service/dataimport"
)

// ImportHandler is the transport adapter for user data imports.
type ImportHandler struct {
	svc *dataimport.Service
}

// NewImportHandler constructs the handler.
func NewImportHandler(svc *dataimport.Service) *ImportHandler {
	return &ImportHandler{svc: svc}
}

type importSectionCountDTO struct {
	Valid      int `json:"valid"`
	Invalid    int `json:"invalid"`
	WillAdd    int `json:"willAdd"`
	Duplicates int `json:"duplicates"`
}

type importJobDTO struct {
	ID            string                            `json:"id"`
	ClientID      string                            `json:"clientId"`
	Format        string                            `json:"format"`
	Mode          string                            `json:"mode,omitempty"`
	Status        string                            `json:"status"`
	ProgressPct   float64                           `json:"progressPct"`
	Stage         string                            `json:"stage,omitempty"`
	ErrorMessage  string                            `json:"errorMessage,omitempty"`
	Sections      map[string]importSectionCountDTO  `json:"sections,omitempty"`
	Totals        importSectionCountDTO             `json:"totals"`
	Added         map[string]int                    `json:"added,omitempty"`
	FileName      string                            `json:"fileName,omitempty"`
	ByteSize      int64                             `json:"byteSize,omitempty"`
	ExpiresAt     *string                           `json:"expiresAt,omitempty"`
	CreatedAt     string                            `json:"createdAt"`
	StartedAt     *string                           `json:"startedAt,omitempty"`
	FinishedAt    *string                           `json:"finishedAt,omitempty"`
}

func importToDTO(j *domain.ImportJob) importJobDTO {
	d := importJobDTO{
		ID: j.ID, ClientID: j.ClientID, Format: string(j.Format), Mode: string(j.Mode),
		Status: string(j.Status), ProgressPct: j.ProgressPct, Stage: j.Stage,
		ErrorMessage: j.ErrorMessage, FileName: j.FileName, ByteSize: j.ByteSize,
		Totals: importSectionCountDTO{
			Valid: j.Totals.Valid, Invalid: j.Totals.Invalid,
			WillAdd: j.Totals.WillAdd, Duplicates: j.Totals.Duplicates,
		},
		CreatedAt: j.CreatedAt.UTC().Format(time.RFC3339Nano),
	}
	if len(j.SectionCounts) > 0 {
		d.Sections = make(map[string]importSectionCountDTO, len(j.SectionCounts))
		for k, v := range j.SectionCounts {
			d.Sections[string(k)] = importSectionCountDTO{
				Valid: v.Valid, Invalid: v.Invalid, WillAdd: v.WillAdd, Duplicates: v.Duplicates,
			}
		}
	}
	if len(j.AddedCounts) > 0 {
		d.Added = make(map[string]int, len(j.AddedCounts))
		for k, v := range j.AddedCounts {
			d.Added[string(k)] = v
		}
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
	return d
}

// Preview handles POST /api/v1/import/preview
// Accepts multipart field "file" or raw body with Content-Type application/json|text/csv.
func (h *ImportHandler) Preview(w http.ResponseWriter, r *http.Request) {
	clientID := clientIDFrom(r)
	var fileName string
	var data []byte
	var formatHint string

	ct := strings.ToLower(r.Header.Get("Content-Type"))
	if strings.HasPrefix(ct, "multipart/") {
		if err := r.ParseMultipartForm(domain.MaxImportUploadBytes + (1 << 20)); err != nil {
			writeError(w, fmt.Errorf("%w: invalid multipart form", domain.ErrInvalidArgument))
			return
		}
		if v := strings.TrimSpace(r.FormValue("clientId")); v != "" {
			clientID = v
		}
		formatHint = r.FormValue("format")
		f, hdr, err := r.FormFile("file")
		if err != nil {
			writeError(w, fmt.Errorf("%w: file field is required", domain.ErrInvalidArgument))
			return
		}
		defer f.Close()
		if hdr != nil {
			fileName = hdr.Filename
		}
		data, err = io.ReadAll(io.LimitReader(f, domain.MaxImportUploadBytes+1))
		if err != nil {
			writeError(w, fmt.Errorf("%w: failed to read upload", domain.ErrInvalidArgument))
			return
		}
	} else {
		if v := r.URL.Query().Get("clientId"); v != "" && clientID == "" {
			clientID = v
		}
		formatHint = r.URL.Query().Get("format")
		fileName = r.URL.Query().Get("fileName")
		var err error
		data, err = io.ReadAll(io.LimitReader(r.Body, domain.MaxImportUploadBytes+1))
		if err != nil {
			writeError(w, fmt.Errorf("%w: failed to read body", domain.ErrInvalidArgument))
			return
		}
		if strings.Contains(ct, "csv") {
			formatHint = "csv"
		} else if strings.Contains(ct, "json") {
			formatHint = "json"
		}
	}
	if len(data) > domain.MaxImportUploadBytes {
		writeError(w, fmt.Errorf("%w: file too large", domain.ErrInvalidArgument))
		return
	}
	job, err := h.svc.Preview(r.Context(), dataimport.PreviewInput{
		ClientID: clientID, FileName: fileName, FileBytes: data, FormatHint: formatHint,
	})
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, importToDTO(job))
}

type confirmBody struct {
	ClientID string `json:"clientId"`
	Mode     string `json:"mode"`
}

// Confirm handles POST /api/v1/import/{id}/confirm
func (h *ImportHandler) Confirm(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var body confirmBody
	if err := decodeJSON(r, &body, DefaultMaxJSONBody); err != nil {
		// allow empty body with query mode
		body.Mode = r.URL.Query().Get("mode")
	}
	clientID := body.ClientID
	if clientID == "" {
		clientID = clientIDFrom(r)
	}
	mode := body.Mode
	if mode == "" {
		mode = r.URL.Query().Get("mode")
	}
	job, err := h.svc.Confirm(r.Context(), clientID, id, mode)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusAccepted, importToDTO(job))
}

// List handles GET /api/v1/import
func (h *ImportHandler) List(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	limit, _ := strconv.Atoi(q.Get("limit"))
	offset, _ := strconv.Atoi(q.Get("offset"))
	list, err := h.svc.List(r.Context(), clientIDFrom(r), limit, offset)
	if err != nil {
		writeError(w, err)
		return
	}
	items := make([]importJobDTO, 0, len(list))
	for i := range list {
		items = append(items, importToDTO(&list[i]))
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"clientId": clientIDFrom(r), "imports": items, "count": len(items),
	})
}

// Get handles GET /api/v1/import/{id}
func (h *ImportHandler) Get(w http.ResponseWriter, r *http.Request) {
	job, err := h.svc.Get(r.Context(), clientIDFrom(r), r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, importToDTO(job))
}

// Cancel handles POST /api/v1/import/{id}/cancel
func (h *ImportHandler) Cancel(w http.ResponseWriter, r *http.Request) {
	job, err := h.svc.Cancel(r.Context(), clientIDFrom(r), r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, importToDTO(job))
}

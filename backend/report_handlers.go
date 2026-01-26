package cashtrack

import (
	dbgen "cashtrack/backend/gen/db"
	"encoding/json"
	"io"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const maxReportUploadSize = 10 << 20

type ReportUploadHandler Handler
type ReportListHandler Handler
type ReportDownloadHandler Handler
type ReportDeleteHandler Handler

type ReportInfo struct {
	ID         int64     `json:"id"`
	Filename   string    `json:"filename"`
	SizeBytes  int64     `json:"size_bytes"`
	Status     string    `json:"status"`
	UploadedAt time.Time `json:"uploaded_at"`
	ErrorText  string    `json:"status_description,omitempty"`
}

func NewReportUploadHandler(db *Db) *ReportUploadHandler {
	return &ReportUploadHandler{
		Path: "/api/reports/upload",
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				w.Header().Set("Allow", http.MethodPost)
				w.WriteHeader(http.StatusMethodNotAllowed)
				return
			}

			user, ok := userFromRequest(r.Context(), db, r.Header)
			if !ok {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}

			r.Body = http.MaxBytesReader(w, r.Body, maxReportUploadSize+1024)
			if err := r.ParseMultipartForm(maxReportUploadSize); err != nil {
				http.Error(w, "invalid form data", http.StatusBadRequest)
				return
			}

			file, header, err := r.FormFile("file")
			if err != nil {
				http.Error(w, "missing file", http.StatusBadRequest)
				return
			}
			defer file.Close()

			filename := filepath.Base(header.Filename)
			if strings.ToLower(filepath.Ext(filename)) != ".csv" {
				http.Error(w, "only csv files are allowed", http.StatusBadRequest)
				return
			}

			data, err := io.ReadAll(file)
			if err != nil {
				http.Error(w, "failed to read file", http.StatusBadRequest)
				return
			}
			if int64(len(data)) > maxReportUploadSize {
				http.Error(w, "file too large", http.StatusRequestEntityTooLarge)
				return
			}

			contentType := header.Header.Get("Content-Type")
			if contentType == "" {
				contentType = "application/octet-stream"
			}

			err = db.Queries.CreateReport(r.Context(), dbgen.CreateReportParams{
				UserID:      user.ID,
				Filename:    filename,
				ContentType: contentType,
				Data:        data,
				Status:      "pending",
			})
			if err != nil {
				http.Error(w, "failed to save file", http.StatusInternalServerError)
				return
			}

			w.WriteHeader(http.StatusCreated)
		}),
	}
}

func NewReportListHandler(db *Db) *ReportListHandler {
	return &ReportListHandler{
		Path: "/api/reports",
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				w.Header().Set("Allow", http.MethodGet)
				w.WriteHeader(http.StatusMethodNotAllowed)
				return
			}

			user, ok := userFromRequest(r.Context(), db, r.Header)
			if !ok {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}

			rows, err := db.Queries.ListReportsByUser(r.Context(), user.ID)
			if err != nil {
				http.Error(w, "failed to load reports", http.StatusInternalServerError)
				return
			}

			reports := make([]ReportInfo, 0, len(rows))
			for _, row := range rows {
				report := ReportInfo{
					ID:         row.ID,
					Filename:   row.Filename,
					SizeBytes:  row.SizeBytes,
					Status:     row.Status,
					UploadedAt: row.UploadedAt.Time,
				}
				if row.StatusDescription.Valid {
					report.ErrorText = row.StatusDescription.String
				}
				reports = append(reports, report)
			}

			w.Header().Set("Content-Type", "application/json")
			if err := json.NewEncoder(w).Encode(reports); err != nil {
				http.Error(w, "failed to encode response", http.StatusInternalServerError)
			}
		}),
	}
}

func NewReportDownloadHandler(db *Db) *ReportDownloadHandler {
	return &ReportDownloadHandler{
		Path: "/api/reports/download",
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				w.Header().Set("Allow", http.MethodGet)
				w.WriteHeader(http.StatusMethodNotAllowed)
				return
			}

			user, ok := userFromRequest(r.Context(), db, r.Header)
			if !ok {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}

			id := r.URL.Query().Get("id")
			if id == "" {
				http.Error(w, "missing id", http.StatusBadRequest)
				return
			}

			reportID, err := strconv.ParseInt(id, 10, 64)
			if err != nil {
				http.Error(w, "invalid id", http.StatusBadRequest)
				return
			}

			report, err := db.Queries.GetReportByID(r.Context(), dbgen.GetReportByIDParams{
				ID:     reportID,
				UserID: user.ID,
			})
			if err != nil {
				http.Error(w, "file not found", http.StatusNotFound)
				return
			}

			contentType := report.ContentType
			if contentType == "" {
				contentType = "application/octet-stream"
			}
			w.Header().Set("Content-Type", contentType)
			w.Header().Set("Content-Disposition", `attachment; filename="`+report.Filename+`"`)
			w.Header().Set("Content-Length", strconv.Itoa(len(report.Data)))
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(report.Data)
		}),
	}
}

func NewReportDeleteHandler(db *Db) *ReportDeleteHandler {
	return &ReportDeleteHandler{
		Path: "/api/reports/delete",
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodDelete {
				w.Header().Set("Allow", http.MethodDelete)
				w.WriteHeader(http.StatusMethodNotAllowed)
				return
			}

			user, ok := userFromRequest(r.Context(), db, r.Header)
			if !ok {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}

			idParam := r.URL.Query().Get("id")
			if idParam == "" {
				http.Error(w, "missing id", http.StatusBadRequest)
				return
			}

			id, err := strconv.ParseInt(idParam, 10, 64)
			if err != nil {
				http.Error(w, "invalid id", http.StatusBadRequest)
				return
			}

			if err := db.Queries.DeleteReportByID(r.Context(), dbgen.DeleteReportByIDParams{
				ID:     id,
				UserID: user.ID,
			}); err != nil {
				http.Error(w, "failed to delete report", http.StatusInternalServerError)
				return
			}

			w.WriteHeader(http.StatusNoContent)
		}),
	}
}

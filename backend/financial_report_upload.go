package cashtrack

import (
	"encoding/json"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"time"
)

const maxReportUploadSize = 10 << 20

type FinancialReportUploadHandler Handler
type FinancialReportListHandler Handler

type ReportInfo struct {
	ID         int64     `json:"id"`
	Filename   string    `json:"filename"`
	SizeBytes  int64     `json:"size_bytes"`
	UploadedAt time.Time `json:"uploaded_at"`
}

func NewFinancialReportUploadHandler(db *Db) *FinancialReportUploadHandler {
	return &FinancialReportUploadHandler{
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

			_, err = db.conn.Exec(
				r.Context(),
				`INSERT INTO financial_reports (user_id, filename, content_type, data) VALUES ($1, $2, $3, $4)`,
				user.ID,
				filename,
				contentType,
				data,
			)
			if err != nil {
				http.Error(w, "failed to save file", http.StatusInternalServerError)
				return
			}

			w.WriteHeader(http.StatusCreated)
		}),
	}
}

func NewFinancialReportListHandler(db *Db) *FinancialReportListHandler {
	return &FinancialReportListHandler{
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

			rows, err := db.conn.Query(
				r.Context(),
				`SELECT id, filename, octet_length(data) AS size_bytes, uploaded_at
				FROM financial_reports
				WHERE user_id = $1
				ORDER BY uploaded_at DESC, id DESC`,
				user.ID,
			)
			if err != nil {
				http.Error(w, "failed to load reports", http.StatusInternalServerError)
				return
			}
			defer rows.Close()

			reports := make([]ReportInfo, 0)
			for rows.Next() {
				var report ReportInfo
				if err := rows.Scan(&report.ID, &report.Filename, &report.SizeBytes, &report.UploadedAt); err != nil {
					http.Error(w, "failed to load reports", http.StatusInternalServerError)
					return
				}
				reports = append(reports, report)
			}
			if err := rows.Err(); err != nil {
				http.Error(w, "failed to load reports", http.StatusInternalServerError)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			if err := json.NewEncoder(w).Encode(reports); err != nil {
				http.Error(w, "failed to encode response", http.StatusInternalServerError)
			}
		}),
	}
}

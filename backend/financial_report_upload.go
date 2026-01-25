package cashtrack

import (
	"io"
	"net/http"
	"path/filepath"
	"strings"
)

const maxReportUploadSize = 10 << 20

type FinancialReportUploadHandler Handler

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

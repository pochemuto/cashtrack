package cashtrack

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"time"

	apiv1 "cashtrack/backend/gen/api/v1"
	"cashtrack/backend/gen/api/v1/apiv1connect"
	dbgen "cashtrack/backend/gen/db"
	"connectrpc.com/connect"
	"connectrpc.com/validate"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

const maxReportUploadSize = 10 << 20

type ReportService struct {
	db *Db
}

type ReportServiceHandler Handler

func NewReportServiceHandler(db *Db) *ReportServiceHandler {
	service := &ReportService{db: db}
	path, handler := apiv1connect.NewReportServiceHandler(
		service,
		connect.WithInterceptors(validate.NewInterceptor(), NewAuthInterceptor(db)),
	)
	return &ReportServiceHandler{Path: path, Handler: handler}
}

func (s *ReportService) UploadReport(ctx context.Context, req *apiv1.UploadReportRequest) (*apiv1.UploadReportResponse, error) {
	user, err := requireUser(ctx)
	if err != nil {
		return nil, err
	}

	filename := filepath.Base(strings.TrimSpace(req.Filename))
	if filename == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("filename is required"))
	}
	if strings.ToLower(filepath.Ext(filename)) != ".csv" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("only csv files are allowed"))
	}
	if len(req.Data) == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("file is empty"))
	}
	if int64(len(req.Data)) > maxReportUploadSize {
		return nil, connect.NewError(connect.CodeResourceExhausted, errors.New("file too large"))
	}

	if err := s.db.Queries.CreateReport(ctx, dbgen.CreateReportParams{
		UserID:      user.Id,
		Filename:    filename,
		ContentType: pgtype.Text{String: defaultContentType(req.ContentType), Valid: true},
		Data:        req.Data,
		Status:      "pending",
	}); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return &apiv1.UploadReportResponse{}, nil
}

func (s *ReportService) ListReports(ctx context.Context, req *apiv1.ListReportsRequest) (*apiv1.ListReportsResponse, error) {
	user, err := requireUser(ctx)
	if err != nil {
		return nil, err
	}

	rows, err := s.db.Queries.ListReportsByUser(ctx, user.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	reports := make([]*apiv1.ReportInfo, 0, len(rows))
	for _, row := range rows {
		report := &apiv1.ReportInfo{
			Id:         int32(row.ID),
			Filename:   row.Filename,
			SizeBytes:  int32(row.SizeBytes),
			Status:     row.Status,
			UploadedAt: row.UploadedAt.Time.Format(time.RFC3339Nano),
		}
		if row.StatusDescription.Valid {
			report.StatusDescription = row.StatusDescription.String
		}
		reports = append(reports, report)
	}

	return &apiv1.ListReportsResponse{Reports: reports}, nil
}

func (s *ReportService) DownloadReport(ctx context.Context, req *apiv1.DownloadReportRequest) (*apiv1.DownloadReportResponse, error) {
	user, err := requireUser(ctx)
	if err != nil {
		return nil, err
	}

	if req.Id == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("id is required"))
	}

	report, err := s.db.Queries.GetReportByID(ctx, dbgen.GetReportByIDParams{
		ID:     int64(req.Id),
		UserID: user.Id,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, connect.NewError(connect.CodeNotFound, errors.New("file not found"))
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	contentType := report.ContentType.String
	if !report.ContentType.Valid || strings.TrimSpace(contentType) == "" {
		contentType = "application/octet-stream"
	}

	return &apiv1.DownloadReportResponse{
		Data:        report.Data,
		Filename:    report.Filename,
		ContentType: contentType,
	}, nil
}

func defaultContentType(value string) string {
	contentType := strings.TrimSpace(value)
	if contentType == "" {
		return "application/octet-stream"
	}
	return contentType
}

func (s *ReportService) DeleteReport(ctx context.Context, req *apiv1.DeleteReportRequest) (*apiv1.DeleteReportResponse, error) {
	user, err := requireUser(ctx)
	if err != nil {
		return nil, err
	}

	if req.Id == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("id is required"))
	}
	if err := s.db.Queries.DeleteReportByID(ctx, dbgen.DeleteReportByIDParams{
		ID:     int64(req.Id),
		UserID: user.Id,
	}); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return &apiv1.DeleteReportResponse{}, nil
}

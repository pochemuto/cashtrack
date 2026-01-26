package cashtrack

import (
	db "cashtrack/backend/gen/db"
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

type ReportProcessor struct {
	db           *Db
	parsing      *ReportParsingService
	transactions *TransactionsService
}

func NewReportProcessor(db *Db, parsing *ReportParsingService, transactions *TransactionsService) *ReportProcessor {
	return &ReportProcessor{
		db:           db,
		parsing:      parsing,
		transactions: transactions,
	}
}

func (p *ReportProcessor) ProcessPendingReports(ctx context.Context) error {
	reports, err := p.db.Queries.ListPendingReports(ctx)
	if err != nil {
		return fmt.Errorf("load pending reports: %w", err)
	}

	for _, report := range reports {
		parsed, err := p.parsing.Parse(report.Data, report.Filename)
		if err != nil {
			log.Error().Err(err).Int64("report_id", report.ID).Msg("failed to parse report")
			if updateErr := p.updateReportStatus(ctx, report.ID, report.UserID, "failed", err.Error()); updateErr != nil {
				return updateErr
			}
			continue
		}

		if err := p.replaceTransactionsForReport(ctx, report.ID, report.UserID, parsed.Transactions); err != nil {
			log.Error().Err(err).Int64("report_id", report.ID).Msg("failed to store transactions")
			if updateErr := p.updateReportStatus(ctx, report.ID, report.UserID, "failed", err.Error()); updateErr != nil {
				return updateErr
			}
			continue
		}
	}

	return nil
}

func (p *ReportProcessor) Run(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		if err := p.ProcessPendingReports(ctx); err != nil {
			log.Error().Err(err).Msg("failed to process pending reports")
		}

		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (p *ReportProcessor) replaceTransactionsForReport(ctx context.Context, reportID int64, userID int32, entries []ParsedTransaction) error {
	tx, err := p.db.conn.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	if err := p.transactions.ReplaceForSourceTx(ctx, tx, userID, reportID, entries); err != nil {
		return err
	}
	txQueries := p.db.Queries.WithTx(tx)
	if err := txQueries.UpdateReportStatusWithError(ctx, db.UpdateReportStatusWithErrorParams{
		Status:            "processed",
		StatusDescription: errorTextOrNull(fmt.Sprintf("transactions: %d", len(entries))),
		ID:                reportID,
		UserID:            userID,
	}); err != nil {
		return fmt.Errorf("update report status: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}
	return nil
}

func (p *ReportProcessor) updateReportStatus(ctx context.Context, reportID int64, userID int32, status string, errorText string) error {
	err := p.db.Queries.UpdateReportStatusWithError(ctx, db.UpdateReportStatusWithErrorParams{
		Status:            status,
		StatusDescription: errorTextOrNull(errorText),
		ID:                reportID,
		UserID:            userID,
	})
	if err != nil {
		return fmt.Errorf("update report status: %w", err)
	}
	return nil
}

func errorTextOrNull(value string) pgtype.Text {
	if strings.TrimSpace(value) == "" {
		return pgtype.Text{}
	}
	return pgtype.Text{String: value, Valid: true}
}

//go:build wireinject
// +build wireinject

package cashtrack

import (
	"context"
	"net/http"
)
import "github.com/google/wire"

func handlers(todo *TodoHandler, greet *GreetHandler, auth *AuthHandler, authMe *AuthMeHandler, authLogout *AuthLogoutHandler, upload *ReportUploadHandler, reportList *ReportListHandler, reportDownload *ReportDownloadHandler, reportDelete *ReportDeleteHandler) []*Handler {
	return []*Handler{(*Handler)(todo), (*Handler)(greet), (*Handler)(auth), (*Handler)(authMe), (*Handler)(authLogout), (*Handler)(upload), (*Handler)(reportList), (*Handler)(reportDownload), (*Handler)(reportDelete)}
}

func InitializeApp(ctx context.Context) (*http.Server, *ReportProcessor, error) {
	wire.Build(
		handlers,
		NewGreetHandler, NewTodoHandler, NewAuthHandler, NewAuthMeHandler, NewAuthLogoutHandler, NewReportUploadHandler, NewReportListHandler, NewReportDownloadHandler, NewReportDeleteHandler,
		NewReportParsingService, NewTransactionsService, NewReportProcessor,
		ProvideConfig,
		wire.FieldsOf(new(Config), "ServerConfig", "db"),
		NewHttpServer, NewPgxPool, NewDB,
	)
	return nil, nil
}

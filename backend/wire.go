//go:build wireinject
// +build wireinject

package cashtrack

import (
	"context"
	"net/http"
)
import "github.com/google/wire"

func handlers(todo *TodoHandler, greet *GreetHandler, auth *AuthHandler, authMe *AuthMeHandler, authLogout *AuthLogoutHandler, upload *ReportUploadHandler, reportList *ReportListHandler, reportDownload *ReportDownloadHandler, reportDelete *ReportDeleteHandler, transactionsList *TransactionsListHandler, categories *CategoriesHandler, category *CategoryHandler, categoryRules *CategoryRulesHandler, categoryRule *CategoryRuleHandler, categoryRulesApply *CategoryRulesApplyHandler, categoryRulesReorder *CategoryRulesReorderHandler, transactionCategory *TransactionCategoryHandler) []*Handler {
	return []*Handler{(*Handler)(todo), (*Handler)(greet), (*Handler)(auth), (*Handler)(authMe), (*Handler)(authLogout), (*Handler)(upload), (*Handler)(reportList), (*Handler)(reportDownload), (*Handler)(reportDelete), (*Handler)(transactionsList), (*Handler)(categories), (*Handler)(category), (*Handler)(categoryRules), (*Handler)(categoryRule), (*Handler)(categoryRulesApply), (*Handler)(categoryRulesReorder), (*Handler)(transactionCategory)}
}

func InitializeApp(ctx context.Context) (*http.Server, *ReportProcessor, error) {
	wire.Build(
		handlers,
		NewGreetHandler, NewTodoHandler, NewAuthHandler, NewAuthMeHandler, NewAuthLogoutHandler, NewReportUploadHandler, NewReportListHandler, NewReportDownloadHandler, NewReportDeleteHandler, NewTransactionsListHandler, NewCategoriesHandler, NewCategoryHandler, NewCategoryRulesHandler, NewCategoryRuleHandler, NewCategoryRulesApplyHandler, NewCategoryRulesReorderHandler, NewTransactionCategoryHandler,
		NewReportParsingService, NewTransactionsService, NewReportProcessor,
		ProvideConfig,
		wire.FieldsOf(new(Config), "ServerConfig", "db"),
		NewHttpServer, NewPgxPool, NewDB,
	)
	return nil, nil
}

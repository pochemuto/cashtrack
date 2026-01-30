//go:build wireinject
// +build wireinject

package cashtrack

import "context"
import "github.com/google/wire"

func handlers(
	todo *TodoHandler,
	greet *GreetHandler,
	auth *AuthHandler,
	authService *AuthServiceHandler,
	reportService *ReportServiceHandler,
	transactionService *TransactionServiceHandler,
	categoryService *CategoryServiceHandler,
) []*Handler {
	return []*Handler{
		(*Handler)(todo),
		(*Handler)(greet),
		(*Handler)(auth),
		(*Handler)(authService),
		(*Handler)(reportService),
		(*Handler)(transactionService),
		(*Handler)(categoryService),
	}
}

func InitializeApp(ctx context.Context) (*App, error) {
	wire.Build(
		handlers,
		NewGreetHandler,
		NewTodoHandler,
		NewAuthHandler,
		NewAuthServiceHandler,
		NewReportServiceHandler,
		NewTransactionServiceHandler,
		NewCategoryServiceHandler,
		NewReportParsingService, NewTransactionsService, NewReportProcessor,
		ProvideConfig,
		wire.FieldsOf(new(Config), "ServerConfig", "Db"),
		NewHttpServer, NewPgxPool, NewDB,
		wire.Struct(new(App), "*"),
	)
	return nil, nil
}

package cashtrack

func Handlers(
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

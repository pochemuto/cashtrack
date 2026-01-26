package cashtrack

func Handlers(todo *TodoHandler, greet *GreetHandler, auth *AuthHandler, authMe *AuthMeHandler, authLogout *AuthLogoutHandler, upload *ReportUploadHandler, reportList *ReportListHandler, reportDownload *ReportDownloadHandler, reportDelete *ReportDeleteHandler, transactionsList *TransactionsListHandler, categories *CategoriesHandler, category *CategoryHandler, categoryRules *CategoryRulesHandler, categoryRule *CategoryRuleHandler, categoryRulesApply *CategoryRulesApplyHandler, transactionCategory *TransactionCategoryHandler) []*Handler {
	return []*Handler{(*Handler)(todo), (*Handler)(greet), (*Handler)(auth), (*Handler)(authMe), (*Handler)(authLogout), (*Handler)(upload), (*Handler)(reportList), (*Handler)(reportDownload), (*Handler)(reportDelete), (*Handler)(transactionsList), (*Handler)(categories), (*Handler)(category), (*Handler)(categoryRules), (*Handler)(categoryRule), (*Handler)(categoryRulesApply), (*Handler)(transactionCategory)}
}

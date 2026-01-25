package cashtrack

func Handlers(todo *TodoHandler, greet *GreetHandler, auth *AuthHandler, authMe *AuthMeHandler, authLogout *AuthLogoutHandler, upload *ReportUploadHandler, reportList *ReportListHandler, reportDownload *ReportDownloadHandler) []*Handler {
	return []*Handler{(*Handler)(todo), (*Handler)(greet), (*Handler)(auth), (*Handler)(authMe), (*Handler)(authLogout), (*Handler)(upload), (*Handler)(reportList), (*Handler)(reportDownload)}
}

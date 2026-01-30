package cashtrack

import "net/http"

type App struct {
	Server    *http.Server
	Processor *ReportProcessor
}

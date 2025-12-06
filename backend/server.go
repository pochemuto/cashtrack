package cashtrack

import (
	"net/http"
)

type Handler struct {
	Path    string
	Handler http.Handler
}

func cors(next http.Handler) http.Handler {
	log.Info().Msgf("CORS enabled")
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Credentials", "true")
		}

		if r.Method == http.MethodOptions {
			w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Connect-Protocol-Version, Connect-Timeout-Ms, Connect-Accept-Encoding, Connect-Content-Encoding")
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

type ServerConfig struct {
	StaticPath string `envDefault:"./public"`
	CORS       bool   `envDefault:"false"`
	Host       string `envDefault:"0.0.0.0:8080"`
}

func NewHttpServer(config ServerConfig, handlers []*Handler) *http.Server {
	mux := http.NewServeMux()

	filePatten := "/"
	log.Info().Msgf("Serving %s as files from %s", filePatten, config.StaticPath)
	mux.Handle(filePatten, http.FileServer(http.Dir(config.StaticPath)))
	for _, handler := range handlers {
		log.Info().Msgf("Serving %s", handler.Path)
		mux.Handle(handler.Path, handler.Handler)
	}

	p := new(http.Protocols)
	p.SetHTTP1(true)
	// Use h2c so we can serve HTTP/2 without TLS.
	p.SetUnencryptedHTTP2(true)

	var handler http.Handler
	if config.CORS {
		handler = cors(mux)
	} else {
		handler = mux
	}
	s := http.Server{
		Addr:      config.Host,
		Handler:   handler,
		Protocols: p,
	}

	log.Info().Msgf("Server listening on %s", s.Addr)
	return &s
}

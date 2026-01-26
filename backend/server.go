package cashtrack

import (
	"net/http"
	"path/filepath"
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
			w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PATCH, DELETE")
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
	mux.Handle(filePatten, spaFileServer(config.StaticPath))
	for _, handler := range handlers {
		log.Info().Msgf("Serving %s", handler.Path)
		mux.Handle(handler.Path, handler.Handler)
	}
	mux.Handle("/health", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

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

func spaFileServer(root string) http.Handler {
	fileSystem := http.Dir(root)
	fileServer := http.FileServer(fileSystem)
	indexPath := filepath.Join(root, "index.html")

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			fileServer.ServeHTTP(w, r)
			return
		}

		if r.URL.Path == "/" {
			http.ServeFile(w, r, indexPath)
			return
		}

		f, err := fileSystem.Open(r.URL.Path)
		if err == nil {
			defer f.Close()
			info, statErr := f.Stat()
			if statErr == nil && !info.IsDir() {
				r2 := *r
				r2.URL.Path = r.URL.Path
				fileServer.ServeHTTP(w, &r2)
				return
			}
		}

		htmlPath := r.URL.Path + ".html"
		if f, err := fileSystem.Open(htmlPath); err == nil {
			f.Close()
			http.ServeFile(w, r, filepath.Join(root, filepath.FromSlash(htmlPath)))
			return
		}

		http.ServeFile(w, r, indexPath)
	})
}

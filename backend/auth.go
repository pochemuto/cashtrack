package cashtrack

import "net/http"

type AuthHandler Handler

func NewAuthHandler() *AuthHandler {
	return &AuthHandler{
		Path: "/auth",
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			log.Info().
				Str("method", r.Method).
				Str("path", r.URL.Path).
				Msg("Auth handler called")
			w.WriteHeader(http.StatusNoContent)
		}),
	}
}

package routes

import (
	"context"
	"net/http"
)

func pathMiddleware(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		con := context.WithValue(r.Context(), "url", r.URL)
		h.ServeHTTP(w, r.WithContext(con))
	})
}

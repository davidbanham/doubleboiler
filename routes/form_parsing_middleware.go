package routes

import "net/http"

func formParsingMiddleware(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		h.ServeHTTP(w, r)
	})
}

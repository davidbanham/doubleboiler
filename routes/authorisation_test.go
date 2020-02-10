package routes

import (
	"context"
	"net/http"

	m "doubleboiler/models"
)

func contextify(u m.User, r *http.Request) *http.Request {
	con := context.WithValue(r.Context(), "user", u)
	return r.WithContext(con)
}

package routes

import (
	"context"
	"doubleboiler/models"
	"net/http"
)

func contextify(u models.User, r *http.Request) *http.Request {
	con := context.WithValue(r.Context(), "user", u)
	return r.WithContext(con)
}

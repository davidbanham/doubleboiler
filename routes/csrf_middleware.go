package routes

import (
	"doubleboiler/logger"
	"doubleboiler/models"
	"doubleboiler/util"
	"net/http"
)

func csrfMiddleware(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			h.ServeHTTP(w, r)
			return
		}

		unconv := r.Context().Value("user")

		var u models.User

		if unconv == nil {
			h.ServeHTTP(w, r)
			return
		}

		u = unconv.(models.User)

		expectedToken := util.CalcToken(u.ID, "")

		r.ParseMultipartForm(128 << 20)
		r.ParseForm()
		if r.FormValue("csrf") == expectedToken {
			h.ServeHTTP(w, r)
			return
		}
		logger.Log(r.Context(), logger.Warning, "expected csrf token", expectedToken, "recieved", r.FormValue("csrf"), "for user", u.Email, u.ID)
		errRes(w, r, 403, "Invalid csrf token. Please log out, close all tabs of this system and log back in.", nil)
		return
	})
}

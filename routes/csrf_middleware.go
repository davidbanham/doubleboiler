package routes

import (
	"doubleboiler/config"
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

		r.ParseMultipartForm(128 << 20)
		r.ParseForm()

		if err := util.CheckToken(config.SECRET, "", u.ID, r.FormValue("csrf")); err == nil {
			h.ServeHTTP(w, r)
			return
		} else {
			logger.Log(r.Context(), logger.Warning, "invalid csrf token - recieved", r.FormValue("csrf"), "for user", u.Email, u.ID)
			errRes(w, r, 403, "Invalid csrf token. Please log out, close all tabs of this system and log back in.", err)
			return
		}
	})
}

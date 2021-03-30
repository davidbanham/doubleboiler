package routes

import (
	"doubleboiler/models"
	"doubleboiler/util"
	"net/http"
	"time"
)

func loginMiddleware(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/users" {
			if r.Method == "POST" {
				uid := r.FormValue("id")
				if uid == "" {
					h.ServeHTTP(w, r)
					return
				}
				if r.FormValue("token") != "" {
					user := models.User{}
					if err := user.FindByID(r.Context(), r.FormValue("id")); err != nil {
						errRes(w, r, 403, "Invalid user", err)
						return
					}
					expiry := r.FormValue("expiry")
					if err := checkTokenExpiry(expiry); err != nil {
						redirToLogin(w, r)
						return
					}
					expectedToken := util.CalcToken(user.Email, expiry)
					if r.FormValue("token") == expectedToken {
						expiration := time.Now().Add(30 * 24 * time.Hour)
						encoded, err := secureCookie.Encode("user", map[string]string{
							"ID": user.ID,
						})
						if err != nil {
							errRes(w, r, 500, "Error encoding cookie", nil)
							return
						}
						cookie := http.Cookie{
							Path:     "/",
							Name:     "user",
							Value:    encoded,
							Expires:  expiration,
							Secure:   true,
							HttpOnly: true,
						}
						http.SetCookie(w, &cookie)
						h.ServeHTTP(w, r)
						return
					}
				}
			}
		}

		authFree := r.Context().Value("authFree").(bool)
		if authFree {
			h.ServeHTTP(w, r)
			return
		}

		unconv := r.Context().Value("user")
		if unconv == nil {
			redirToLogin(w, r)
			return
		}

		h.ServeHTTP(w, r)
	})
}

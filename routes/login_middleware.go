package routes

import (
	"doubleboiler/config"
	"doubleboiler/models"
	"doubleboiler/util"
	"net/http"
	"time"
)

func loginMiddleware(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authFree := isAuthFree(r.Context())

		if len(r.URL.Path) > 5 && r.URL.Path[0:6] == "/users" {
			if r.Method == "POST" {
				uid := r.FormValue("id")
				if uid == "" {
					h.ServeHTTP(w, r)
					return
				}
				if r.FormValue("token") != "" {
					user := models.User{}
					if err := user.FindByID(r.Context(), r.FormValue("id")); err != nil {
						if authFree {
							h.ServeHTTP(w, r)
							return
						}
						errRes(w, r, 403, "Invalid user", err)
						return
					}

					if err := util.CheckToken(config.SECRET, r.FormValue("expiry"), user.Email, r.FormValue("token")); err != nil {
						errRes(w, r, http.StatusUnauthorized, "Invalid token", err)
						return
					}

					expiration := time.Now().Add(30 * 24 * time.Hour)
					encoded, err := secureCookie.Encode("doubleboiler-user", map[string]string{
						"ID": user.ID,
					})
					if err != nil {
						errRes(w, r, 500, "Error encoding cookie", nil)
						return
					}
					cookie := http.Cookie{
						Path:     "/",
						Name:     "doubleboiler-user",
						Domain:   config.DOMAIN,
						SameSite: http.SameSiteLaxMode,
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

		if authFree {
			h.ServeHTTP(w, r)
			return
		}

		nextVal := r.FormValue("next")
		if r.FormValue("next") == "" && !util.Contains([]string{"/login", "/login-2fa"}, r.URL.Path) {
			nextVal = r.URL.Path
		}

		switch user := r.Context().Value("user").(type) {
		default:
			vals := r.URL.Query()
			vals.Add("next", nextVal)

			http.Redirect(w, r, "/login?"+vals.Encode(), 302)
			return
		case models.User:
			if user.TOTPActive && r.Context().Value("totp-verified") != nil && r.Context().Value("totp-verified").(bool) != true {
				vals := r.URL.Query()
				vals.Add("next", nextVal)

				http.Redirect(w, r, "/login-2fa?"+vals.Encode(), http.StatusFound)
				return
			}
		}

		h.ServeHTTP(w, r)
	})
}

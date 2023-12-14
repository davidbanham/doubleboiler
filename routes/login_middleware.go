package routes

import (
	"doubleboiler/models"
	"doubleboiler/util"
	"net/http"
)

var unAuthedPaths = append([]string{
	"",
	"reset-password",
	"verify",
	"login",
	"login-2fa",
	"signup",
	"signup-successful",
	"health",
	"contact",
	"webhooks",
}, assetPaths...)

func loginMiddleware(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rootPath := util.RootPath(r.URL)
		if util.Contains(unAuthedPaths, rootPath) {
			h.ServeHTTP(w, r)
			return
		}

		if r.Method == "POST" && rootPath == "users" && r.FormValue("id") == "" {
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

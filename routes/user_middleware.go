package routes

import (
	"context"
	"doubleboiler/flashes"
	"doubleboiler/logger"
	"doubleboiler/models"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"time"
)

var assetPath = regexp.MustCompile("/js/|/css/")

func userMiddleware(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := r.Cookie("doubleboiler-user")
		if err != nil {
			h.ServeHTTP(w, r)
			return
		}
		cookieValue := make(map[string]string)
		decodeErr := secureCookie.Decode("doubleboiler-user", c.Value, &cookieValue)
		if decodeErr != nil || cookieValue["ID"] == "" {
			deadCookie := http.Cookie{
				Path:     "/",
				Name:     "doubleboiler-user",
				Value:    "",
				Expires:  time.Date(1970, 0, 0, 0, 0, 0, 0, time.UTC),
				Secure:   true,
				HttpOnly: true,
			}
			http.SetCookie(w, &deadCookie)
			logger.Log(r.Context(), logger.Error, "decoding user ID from cookie", decodeErr, c.Value, cookieValue)
			http.Redirect(w, r, "/login", 302)
			return
		}
		user := models.User{}
		if err := user.FindByID(r.Context(), cookieValue["ID"]); err != nil {
			if isAuthFree(r.Context()) {
				h.ServeHTTP(w, r)
				return
			}
			errRes(w, r, 403, "Invalid user", err)
			return
		}

		if user.TOTPActive {
			if cookieValue["TOTP"] != "true" && r.URL.Path != "/login-2fa" && r.URL.Path != "/logout" {
				vals := url.Values{}
				vals.Add("next", r.URL.Path)
				http.Redirect(w, r, "/login-2fa?"+vals.Encode(), http.StatusFound)
				return
			}
		}

		if !assetPath.MatchString(r.URL.Path) {
			logger.Log(r.Context(), logger.Info, fmt.Sprintf("User seen: %s, %s, %s, %s\n", user.ID, user.Email, r.Method, r.URL.Path))
		}

		if user.HasFlashes {
			if err := user.FetchFlashes(r.Context()); err != nil {
				errRes(w, r, http.StatusInternalServerError, "Error looking up flashes", err)
				return
			}
		}

		if r.URL.Query().Get("test-flash") == "true" {
			flash := flashes.Flash{
				Type: flashes.Success,
				Text: "This is a test flash mesage",
			}
			flashed, _ := flash.Add(r.Context())
			r = r.WithContext(flashed)
		}

		con := context.WithValue(r.Context(), "user", user)
		con = context.WithValue(con, "totp-verified", cookieValue["TOTP"] == "true")
		h.ServeHTTP(w, r.WithContext(con))
	})
}

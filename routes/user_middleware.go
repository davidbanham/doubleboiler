package routes

import (
	"context"
	"doubleboiler/flashes"
	"doubleboiler/logger"
	"doubleboiler/models"
	"fmt"
	"net/http"
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

		if !assetPath.MatchString(r.URL.Path) {
			logger.Log(r.Context(), logger.Info, fmt.Sprintf("User seen: %s, %s, %s, %s\n", user.ID, user.Email, r.Method, r.URL.Path))
		}

		if r.URL.Query().Get("test-flash") == "true" {
			if ctx, err := user.PersistFlash(r.Context(), flashes.Flash{
				Type: flashes.Success,
				Text: "This is a test flash mesage",
			}); err != nil {
				errRes(w, r, http.StatusInternalServerError, "Error adding flash message", err)
				return
			} else {
				r = r.WithContext(ctx)
			}
		}

		con := context.WithValue(r.Context(), "user", user)
		con = context.WithValue(con, "totp-verified", cookieValue["TOTP"] == "true")
		h.ServeHTTP(w, r.WithContext(con))
	})
}

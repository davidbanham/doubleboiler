package routes

import (
	"context"
	"doubleboiler/config"
	"doubleboiler/flashes"
	"doubleboiler/logger"
	"doubleboiler/models"
	"doubleboiler/util"
	"fmt"
	"net/http"
	"time"
)

var assetPaths = []string{
	"css",
	"img",
	"js",
	"fonts",
	"privacy_collection_statement.pdf",
	"eula.pdf",
	// Root favicons
	"android-chrome-192x192.png",
	"android-chrome-512x512.png",
	"apple-touch-icon.png",
	"browserconfig.xml",
	"favicon-16x16.png",
	"favicon-32x32.png",
	"favicon.ico",
	"manifest.json",
	"mstile-144x144.png",
	"mstile-150x150.png",
	"mstile-310x150.png",
	"mstile-310x310.png",
	"mstile-70x70.png",
	"safari-pinned-tab.svg",
}

func userMiddleware(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.FormValue("token") != "" {
			userID := util.FirstNonEmptyString(r.FormValue("id"), r.FormValue("uid"))
			if userID != "" {
				user := models.User{}
				if err := user.FindByID(r.Context(), userID); err != nil {
					errRes(w, r, 403, "Invalid user", err)
					return
				}

				if err := util.CheckToken(config.SECRET, r.FormValue("expiry"), user.Email, r.FormValue("token")); err != nil {
					errRes(w, r, http.StatusUnauthorized, "Invalid token", err)
					return
				}

				expiration, _ := time.Parse(time.RFC3339, r.FormValue("expiry"))

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
		} else {
			c, err := r.Cookie("doubleboiler-user")
			if err != nil {
				if err == http.ErrNoCookie {
					h.ServeHTTP(w, r)
					return
				} else {
					errRes(w, r, http.StatusBadRequest, "Cookie error", err)
					return
				}
			}
			if c.Value == "logged_out" {
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
			}

			user := models.User{}
			if err := user.FindByID(r.Context(), cookieValue["ID"]); err != nil {
				errRes(w, r, 403, "Invalid user", err)
				return
			}

			if util.Contains(assetPaths, util.RootPath(r.URL)) {
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
			r = r.WithContext(con)

			h.ServeHTTP(w, r)
			return
		}
	})
}

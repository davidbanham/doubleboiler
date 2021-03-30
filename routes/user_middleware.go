package routes

import (
	"context"
	"doubleboiler/models"
	"log"
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
		if err := secureCookie.Decode("doubleboiler-user", c.Value, &cookieValue); err != nil {
			deadCookie := http.Cookie{
				Path:     "/",
				Name:     "doubleboiler-user",
				Value:    "",
				Expires:  time.Date(1970, 0, 0, 0, 0, 0, 0, time.UTC),
				Secure:   true,
				HttpOnly: true,
			}
			http.SetCookie(w, &deadCookie)
			log.Println("ERROR decoding user ID from cookie", c.Value, cookieValue)
			http.Redirect(w, r, "/login", 302)
			return
		}
		user := models.User{}
		err = user.FindByID(r.Context(), cookieValue["ID"])
		if err != nil {
			errRes(w, r, 403, "Invalid user", err)
			return
		}

		if !assetPath.MatchString(r.URL.Path) {
			log.Printf("INFO User seen: %s, %s, %s, %s\n", user.ID, user.Email, r.Method, r.URL.Path)
		}

		con := context.WithValue(r.Context(), "user", user)
		h.ServeHTTP(w, r.WithContext(con))
	})
}

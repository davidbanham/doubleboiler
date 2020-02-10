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
		err = secureCookie.Decode("doubleboiler-user", c.Value, &cookieValue)
		if err != nil {
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
		u := models.User{}
		err = u.FindByID(r.Context(), cookieValue["ID"])
		if err != nil {
			//errRes(w, r, 403, "Invalid user", err)
			//return
			// This is a workaround for people having cookies from old versions of the testing system. Rather than bounce them and hit an error, clear their cookie and retry
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
			http.Redirect(w, r, r.URL.String(), 302)
			return
		}

		if !assetPath.MatchString(r.URL.Path) {
			log.Printf("INFO User seen: %s, %s, %s, %s\n", u.ID, u.Email, r.Method, r.URL.Path)
		}

		con := context.WithValue(r.Context(), "user", u)
		h.ServeHTTP(w, r.WithContext(con))
	})
}

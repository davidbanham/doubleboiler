package routes

import (
	"doubleboiler/models"
	"fmt"
	"log"
	"net/http"
)

func logoutHandler(w http.ResponseWriter, r *http.Request) {
	unconv := r.Context().Value("user")

	var u models.User
	username := "unknown user"

	if unconv != nil {
		u = unconv.(models.User)
		username = fmt.Sprintf("%s - %s", u.Email, u.ID)
	}

	log.Printf("INFO Destroying cookie for %s", username)

	userCookie := http.Cookie{
		Path:     "/",
		Name:     "doubleboiler-user",
		Value:    "logged_out",
		MaxAge:   0,
		Secure:   true,
		HttpOnly: true,
	}
	http.SetCookie(w, &userCookie)

	targetOrgCookie := http.Cookie{
		Path:     "/",
		Name:     "doubleboiler-targetorg",
		Value:    "logged_out",
		MaxAge:   0,
		Secure:   true,
		HttpOnly: true,
	}

	http.SetCookie(w, &targetOrgCookie)
	http.Redirect(w, r, "/login?clear_session_cache=true", 302)
}

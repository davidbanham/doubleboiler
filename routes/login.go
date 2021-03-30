package routes

import (
	"context"
	"doubleboiler/models"
	"log"
	"net/http"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

func init() {
	r.Path("/login").
		Methods("GET").
		HandlerFunc(serveLogin)

	r.Path("/login").
		Methods("POST").
		HandlerFunc(loginHandler)

	r.Path("/logout").
		Methods("GET").
		HandlerFunc(logoutHandler)
}

type loginPageData struct {
	Context context.Context
	Next    string
}

func serveLogin(w http.ResponseWriter, r *http.Request) {
	if r.URL.Query().Get("flow") == "signup" {
		flash := Flash{
			Type: Success,
			Text: "Password set successfully. Now please log in.",
		}
		r = r.WithContext(flash.Add(r.Context()))
	}

	next := r.URL.Query().Get("next")
	if next == "" {
		next = "/welcome"
	}

	if isLoggedIn(r.Context()) {
		http.Redirect(w, r, next, 302)
		return
	}

	Tmpl.ExecuteTemplate(w, "login.html", loginPageData{
		Context: r.Context(),
		Next:    next,
	})
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	required := []string{
		"email",
		"password",
	}
	okay := checkFormInput(required, r.Form, w, r)
	if !okay {
		return
	}

	inputEmail := strings.ToLower(r.FormValue("email"))
	user := models.User{}
	err := user.FindByColumn(r.Context(), "email", inputEmail)
	if err != nil {
		log.Println("ERROR finding user for login", err)
		errRes(w, r, 401, "Email not found", err)
		return
	}

	passwordFailed := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(r.Form["password"][0]))
	if passwordFailed != nil {
		errRes(w, r, 403, "Incorrect password", nil)
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
		Value:    encoded,
		Expires:  expiration,
		Secure:   true,
		HttpOnly: true,
	}
	http.SetCookie(w, &cookie)

	next := "/welcome"
	if r.FormValue("next") != "" {
		if r.FormValue("next") != "/login" {
			next = r.FormValue("next")
		}
	}

	http.Redirect(w, r, next, 302)
}

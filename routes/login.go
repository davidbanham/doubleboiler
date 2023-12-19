package routes

import (
	"context"
	"doubleboiler/logger"
	"doubleboiler/models"
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

	r.Path("/login-2fa").
		Methods("POST").
		HandlerFunc(login2FAHandler)

	r.Path("/login-2fa").
		Methods("GET").
		HandlerFunc(login2FAFormHandler)

	r.Path("/logout").
		Methods("GET").
		HandlerFunc(logoutHandler)
}

type loginPageData struct {
	basePageData
}

func serveLogin(w http.ResponseWriter, r *http.Request) {
	if isLoggedIn(r.Context()) {
		http.Redirect(w, r, nextFlow("/dashboard", r.Form), 302)
		return
	}

	if err := Tmpl.ExecuteTemplate(w, "login.html", loginPageData{
		basePageData: basePageData{
			PageTitle: "DoubleBoiler - Login",
			Context:   r.Context(),
			Next:      r.FormValue("next"),
		},
	}); err != nil {
		errRes(w, r, 500, "Problem with template", err)
		return
	}
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
	if err := user.FindByColumn(r.Context(), "email", inputEmail); err != nil {
		logger.Log(r.Context(), logger.Error, "finding user for login", err)
		errRes(w, r, 401, "Email not found", err)
		return
	}

	passwordFailed := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(r.FormValue("password")))
	if passwordFailed != nil {
		errRes(w, r, 403, "Incorrect password", nil)
		return
	}

	expiration := time.Now().Add(30 * 24 * time.Hour)
	if user.TOTPActive {
		expiration = time.Now().Add(10 * time.Minute)
	}
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

	if user.TOTPActive {
		// When posting to login the usual user middleware is bypassed
		ctx := context.WithValue(r.Context(), "user", user)

		Tmpl.ExecuteTemplate(w, "login-2fa.html", login2FAPageData{
			basePageData: basePageData{
				PageTitle: "DoubleBoiler - Login - 2FA",
				Context:   ctx,
				Next:      r.FormValue("next"),
			},
			User: user,
		})
	} else {
		http.Redirect(w, r, nextFlow("/dashboard", r.Form), 302)
	}
}

func login2FAHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()

	user := models.User{}
	if err := user.FindByID(r.Context(), r.FormValue("user_id")); err != nil {
		errRes(w, r, http.StatusInternalServerError, "Error looking up user for 2FA verification", err)
		return
	}

	if !user.TOTPActive {
		errRes(w, r, http.StatusBadRequest, "User does not have 2FA active", nil)
		return
	}

	if ok, err := user.Validate2FA(r.Context(), r.FormValue("totp-code"), r.FormValue("totp-recovery-code")); err != nil {
		errRes(w, r, http.StatusInternalServerError, "Error validating 2FA code", err)
		return
	} else {
		if !ok {
			errRes(w, r, http.StatusForbidden, "Invalid 2FA code", nil)
			return
		}
	}

	expiration := time.Now().Add(30 * 24 * time.Hour)
	encoded, err := secureCookie.Encode("doubleboiler-user", map[string]string{
		"ID":   user.ID,
		"TOTP": "true",
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

	http.Redirect(w, r, r.FormValue("next"), 302)
}

func login2FAFormHandler(w http.ResponseWriter, r *http.Request) {
	user := userFromContext(r.Context())

	if !user.TOTPActive || totpVerifiedFromContext(r.Context()) {
		http.Redirect(w, r, nextFlow("/dashboard", r.Form), http.StatusFound)
		return
	}

	Tmpl.ExecuteTemplate(w, "login-2fa.html", login2FAPageData{
		basePageData: basePageData{
			PageTitle: "DoubleBoiler - Login - 2FA",
			Context:   r.Context(),
			Next:      r.FormValue("next"),
		},
		User: user,
	})
}

type login2FAPageData struct {
	basePageData
	User models.User
}

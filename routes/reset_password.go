package routes

import (
	"context"
	"doubleboiler/config"
	"doubleboiler/copy"
	"doubleboiler/models"
	"doubleboiler/util"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/davidbanham/notifications"
)

func init() {
	r.Path("/reset-password").
		Methods("POST").
		HandlerFunc(passwordResetHandler)

	r.Path("/reset-password").
		Methods("GET").
		HandlerFunc(serveResetPassword)
}

func passwordResetHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()

	user := models.User{}
	if err := user.FindByColumn(r.Context(), "email", strings.ToLower(r.FormValue("email"))); err != nil {
		errRes(w, r, 500, "Error looking up user", err)
		return
	}

	expiry := util.CalcExpiry(1)
	token := util.CalcToken(user.Email, expiry)
	escaped := url.QueryEscape(token)
	resetUrl := fmt.Sprintf("%s/reset-password?expiry=%s&uid=%s&token=%s", config.URI, expiry, user.ID, escaped)

	emailHTML, emailText := copy.PasswordResetEmail(resetUrl)

	if err := notifications.SendEmail(notifications.Email{
		To:      user.Email,
		From:    config.SYSTEM_EMAIL,
		ReplyTo: config.SUPPORT_EMAIL,
		Text:    emailText,
		HTML:    emailHTML,
		Subject: fmt.Sprintf("Password reset for your %s account", config.NAME),
	}); err != nil {
		errRes(w, r, 500, "Error sending email", err)
		return
	}

	if err := Tmpl.ExecuteTemplate(w, "reset-password-confirm.html", nil); err != nil {
		errRes(w, r, 500, "Error rendering template", err)
		return
	}
}

func serveResetPassword(w http.ResponseWriter, r *http.Request) {
	qs := r.URL.Query()

	token := qs.Get("token")

	if token != "" {
		// No need to actually check token validity here since there's nothing sensitive on this page
		// Token validity will be checked when the POST to /users is made
		uid := qs.Get("uid")
		expiry := qs.Get("expiry")

		user := models.User{}
		if err := user.FindByID(r.Context(), uid); err != nil {
			errRes(w, r, 404, "User not found", err)
			return
		}

		if err := Tmpl.ExecuteTemplate(w, "reset-password-set-new.html", setNewPasswordPageData{
			User:    user,
			Token:   token,
			Expiry:  expiry,
			Context: r.Context(),
		}); err != nil {
			errRes(w, r, 500, "Error rendering template", err)
			return
		}
	} else {
		if err := Tmpl.ExecuteTemplate(w, "reset-password.html", nil); err != nil {
			errRes(w, r, 500, "Error rendering template", err)
			return
		}
	}
}

type setNewPasswordPageData struct {
	User    models.User
	Token   string
	Expiry  string
	Context context.Context
}

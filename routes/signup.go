package routes

import (
	"doubleboiler/models"
	"doubleboiler/util"
	"errors"
	"net/http"
	"time"
)

func init() {
	r.Path("/signup").
		Methods("GET").
		HandlerFunc(serveSignup)

	r.Path("/signup-successful").
		Methods("GET").
		HandlerFunc(serveSignupSuccessful)

	r.Path("/verify").
		Methods("GET").
		HandlerFunc(verifyHandler)
}

func verifyHandler(w http.ResponseWriter, r *http.Request) {
	qs := r.URL.Query()
	uid := qs.Get("uid")
	token := qs.Get("token")
	expiry := qs.Get("expiry")

	if err := checkTokenExpiry(expiry); err != nil {
		errRes(w, r, 403, "Your invite token is invalid. "+err.Error(), err)
		return
	}

	user := models.User{}
	user.FindByID(r.Context(), uid)

	if user.Verified {
		http.Redirect(w, r, "/welcome", 302)
		return
	}

	expectedToken := util.CalcToken(user.Email, expiry)
	if token != expectedToken {
		errRes(w, r, 403, "Invalid token", nil)
		return
	}

	if err := Tmpl.ExecuteTemplate(w, "verified.html", verifyPageData{
		User:   user,
		Token:  token,
		Expiry: expiry,
		basePageData: basePageData{
			Context: r.Context(),
		},
	}); err != nil {
		errRes(w, r, 500, "Templating error", err)
		return
	}
}

type signupPageData struct {
	basePageData
}

func serveSignup(w http.ResponseWriter, r *http.Request) {
	if err := Tmpl.ExecuteTemplate(w, "signup.html",
		signupPageData{
			basePageData: basePageData{
				PageTitle: "DoubleBoiler - Sign Up",
				Context:   r.Context(),
			},
		}); err != nil {
		errRes(w, r, 500, "Templating error", err)
		return
	}
}

func serveSignupSuccessful(w http.ResponseWriter, r *http.Request) {
	if err := Tmpl.ExecuteTemplate(w, "signup-response.html", nil); err != nil {
		errRes(w, r, 500, "Templating error", err)
		return
	}
}

type verifyPageData struct {
	basePageData
	User   models.User
	Token  string
	Expiry string
}

func checkTokenExpiry(expiry string) error {
	parsed, err := time.Parse(time.RFC3339, expiry)
	if err != nil {
		return errors.New("Invalid expiry string: " + expiry)
	}
	if parsed.Before(time.Now()) {
		return errors.New("Token is expired")
	}
	return nil
}

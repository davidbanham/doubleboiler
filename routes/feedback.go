package routes

import (
	"context"
	"doubleboiler/config"
	"doubleboiler/logger"
	"doubleboiler/models"
	"fmt"
	"net/http"

	"github.com/davidbanham/notifications"
)

type helpPageData struct {
	basePageData
	Organisations models.Organisations
	Email         string
	Context       context.Context
	User          models.User
	SiteKey       string
}

type contactPagedata struct {
	Text    string
	Context context.Context
}

func init() {
	r.Path("/help").
		Methods("GET").
		HandlerFunc(serveHelp)

	r.Path("/contact").
		Methods("POST").
		HandlerFunc(handleFeedback)
}

func serveHelp(w http.ResponseWriter, r *http.Request) {
	user := models.User{}

	con := r.Context().Value("user")
	if con != nil {
		user = con.(models.User)
	}

	relatedOrganisations := models.Organisations{}
	ptr := r.Context().Value("organisations")
	if ptr != nil {
		relatedOrganisations = ptr.(models.Organisations)
	}

	if err := Tmpl.ExecuteTemplate(w, "help.html", helpPageData{
		Organisations: relatedOrganisations,
		Email:         config.SUPPORT_EMAIL,
		Context:       r.Context(),
		User:          user,
		SiteKey:       config.RECAPTCHA_SITE_KEY,
	}); err != nil {
		errRes(w, r, http.StatusBadRequest, "Templating error", err)
		return
	}
}

func handleFeedback(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()

	required := []string{
		"email",
		"body",
	}

	okay := checkFormInput(required, r.Form, w, r)
	if !okay {
		return
	}

	if !isLoggedIn(r.Context()) {
		// No log in, demand a captcha
		if r.FormValue("g-recaptcha-response") == "" {
			logger.Log(r.Context(), logger.Info, "no captcha received", r.URL.RawPath, r.Referer(), r.Form)
			errRes(w, r, http.StatusBadRequest, "No anti-spam key provided", nil)
			return
		}
		verified, err := config.AntiSpam.Verify(r.FormValue("g-recaptcha-response"))
		if err != nil {
			errRes(w, r, http.StatusBadRequest, "error verifying anti spam protection", err)
			return
		}
		if !verified {
			errRes(w, r, http.StatusForbidden, "error verifying anti spam protection", nil)
			return
		}
	}

	emailText := fmt.Sprintf(`
To: %s

From: %s

Phone: %s

Plan: %s

%s
	`, r.Header.Get("Referer"), r.FormValue("email"), r.FormValue("phone"), r.FormValue("plan"), r.FormValue("body"))

	subject := fmt.Sprintf("%s Feedback", config.NAME)
	inputSubject := r.FormValue("subject")
	if inputSubject != "" {
		subject = inputSubject
	}

	to := config.SYSTEM_EMAIL
	if r.FormValue("target") == config.SUPPORT_EMAIL {
		to = r.FormValue("target")
	}

	if err := notifications.SendEmail(notifications.Email{
		To:      to,
		From:    config.SYSTEM_EMAIL,
		ReplyTo: r.FormValue("email"),
		Text:    emailText,
		Subject: subject,
	}); err != nil {
		fmt.Sprintf("INFO error sending feedback email to: %s subject %s text %s", to, subject, emailText)
		errRes(w, r, 500, "error sending email", err)
		return
	}

	if err := Tmpl.ExecuteTemplate(w, "contact.html", contactPagedata{
		Text:    r.FormValue("thanks_text"),
		Context: r.Context(),
	}); err != nil {
		errRes(w, r, http.StatusInternalServerError, "Templating error", err)
		return
	}
}

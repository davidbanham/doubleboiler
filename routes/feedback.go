package routes

import (
	"context"
	"doubleboiler/config"
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
}

type contactPagedata struct {
	Text    string
	Context context.Context
}

func init() {
	r.Path("/help").
		Methods("GET").
		HandlerFunc(serveHelp)

	r.Path("/sales-enquiry").
		Methods("GET").
		HandlerFunc(serveSalesEnquiry)

	r.Path("/trial-mode-upgrade").
		Methods("GET").
		HandlerFunc(serveTrialModeUpgrade)

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
	}); err != nil {
		errRes(w, r, http.StatusBadRequest, "Templating error", err)
		return
	}
}

func serveSalesEnquiry(w http.ResponseWriter, r *http.Request) {
	if err := Tmpl.ExecuteTemplate(w, "sales_enquiry.html", helpPageData{
		Email:   config.SYSTEM_EMAIL,
		Context: r.Context(),
	}); err != nil {
		errRes(w, r, http.StatusBadRequest, "Templating error", err)
		return
	}
}

func serveTrialModeUpgrade(w http.ResponseWriter, r *http.Request) {
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

	if err := Tmpl.ExecuteTemplate(w, "trial_mode_upgrade.html", helpPageData{
		Organisations: relatedOrganisations,
		Email:         config.SYSTEM_EMAIL,
		Context:       r.Context(),
		User:          user,
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

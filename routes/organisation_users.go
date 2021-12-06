package routes

import (
	"context"
	"database/sql"
	"doubleboiler/config"
	"doubleboiler/copy"
	"doubleboiler/logger"
	"doubleboiler/models"
	"doubleboiler/util"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/davidbanham/notifications"
	"github.com/gorilla/mux"
	uuid "github.com/satori/go.uuid"
)

func init() {
	r.Path("/organisations/remove-user/{id}").
		Methods("POST").
		HandlerFunc(organisationUserDeletionHandler)

	r.Path("/organisation-users/{id}").
		Methods("DELETE").
		HandlerFunc(organisationUserDeletionHandler)

	r.Path("/organisation-users/{id}").
		Methods("POST").
		HandlerFunc(organisationUserCreateOrUpdateHandler)

	r.Path("/organisation-users").
		Methods("POST").
		HandlerFunc(organisationUserCreateOrUpdateHandler)
}

func organisationUserCreateOrUpdateHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	required := []string{
		"email",
		"organisationID",
	}
	okay := checkFormInput(required, r.Form, w, r)
	if !okay {
		return
	}

	org := orgFromContext(r.Context(), r.FormValue("organisationID"))

	if !can(r.Context(), org, "admin") {
		errRes(w, r, http.StatusForbidden, "You are not an admin of that organisation", nil)
	}

	if org.ID == "" {
		errRes(w, r, 404, "Organisation not found", nil)
		return
	}

	targetID := r.FormValue("id")
	ou := models.OrganisationUser{}
	if targetID != "" {
		if err := ou.FindByID(r.Context(), targetID); err != nil {
			if err != sql.ErrNoRows {
				errRes(w, r, http.StatusInternalServerError, "Error looking up org user", err)
				return
			}
		}

		ou.Roles = models.Roles{}
		for _, role := range r.Form["roles"] {
			ou.Roles = append(ou.Roles, models.Role{
				Name: role,
			})
		}
	} else {
		email := strings.ToLower(r.FormValue("email"))
		user := models.User{}
		if err := user.FindByColumn(r.Context(), "email", strings.ToLower(email)); err != nil {
			if err != sql.ErrNoRows {
				errRes(w, r, 500, "Error looking up user", err)
				return
			}
			user.New(
				email,
				uuid.NewV4().String(),
			)
			if err = sendOrgInviteEmail(r.Context(), user, org); err != nil {
				errRes(w, r, 500, "Error inviting user", err)
				return
			}
			if err := user.Save(r.Context()); err != nil {
				errRes(w, r, 500, "Error saving user", err)
				return
			}
		} else if err == nil {
			if err := sendOrgAdditionEmail(r.Context(), user, org); err != nil {
				errRes(w, r, 500, "Error notifying user about new org", err)
				return
			}
		}

		ou.New(
			user.ID,
			org.ID,
			models.Roles{},
		)
	}

	if err := ou.Save(r.Context()); err != nil {
		errRes(w, r, 500, "Error saving organisationUser", err)
		return
	}

	http.Redirect(w, r, "/organisations/"+ou.OrganisationID, 302)
}

func organisationUserDeletionHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	ou := models.OrganisationUser{}
	ou.FindByID(r.Context(), vars["id"])

	org := orgFromContext(r.Context(), ou.OrganisationID)

	if !can(r.Context(), org, "admin") {
		errRes(w, r, http.StatusForbidden, "You are not an admin of that organisation", nil)
	}

	if err := ou.Delete(r.Context()); err != nil {
		errRes(w, r, 500, "error removing user from organisation", err)
		return
	}

	http.Redirect(w, r, "/organisations/"+ou.OrganisationID, 302)
}

func sendOrgAdditionEmail(ctx context.Context, user models.User, org models.Organisation) error {
	emailHTML, emailText := copy.OrgAdditionEmail(org.Name)

	if err := notifications.SendEmail(notifications.Email{
		To:      user.Email,
		From:    config.SYSTEM_EMAIL,
		ReplyTo: config.SUPPORT_EMAIL,
		Text:    emailText,
		HTML:    emailHTML,
		Subject: fmt.Sprintf("%s - New %s organisation", org.Name, config.NAME),
	}); err != nil {
		logger.Log(ctx, logger.Error, "sending verification email", err)
		return err
	}
	return nil
}

func sendOrgInviteEmail(ctx context.Context, user models.User, org models.Organisation) error {
	expiry := util.CalcExpiry(30)
	token := util.CalcToken(user.Email, expiry)
	escaped := url.QueryEscape(token)
	verificationUrl := fmt.Sprintf("%s/verify?expiry=%s&uid=%s&token=%s", config.URI, expiry, user.ID, escaped)

	emailHTML, emailText := copy.OrgInviteEmail(org.Name, verificationUrl)

	if err := notifications.SendEmail(notifications.Email{
		To:      user.Email,
		From:    config.SYSTEM_EMAIL,
		ReplyTo: config.SUPPORT_EMAIL,
		Text:    emailText,
		HTML:    emailHTML,
		Subject: fmt.Sprintf("%s - Confirm your %s account", org.Name, config.NAME),
	}); err != nil {
		logger.Log(ctx, logger.Error, "sending verification email", err)
		return err
	}
	return nil
}

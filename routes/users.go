package routes

import (
	"context"
	"database/sql"
	"doubleboiler/config"
	"doubleboiler/models"
	"doubleboiler/util"
	"errors"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	uuid "github.com/satori/go.uuid"
)

func init() {
	r.Path("/users").
		Methods("POST").
		HandlerFunc(userCreateOrUpdateHandler)

	r.Path("/users/{id}").
		Methods("POST").
		HandlerFunc(userCreateOrUpdateHandler)

	r.Path("/users").
		Methods("GET").
		HandlerFunc(usersHandler)

	r.Path("/users/{id}").
		Methods("GET").
		HandlerFunc(userHandler)

	r.Path("/user-settings").
		Methods("GET").
		HandlerFunc(userSettingsRedir)

	r.Path("/users/{id}/impersonate").
		Methods("POST").
		HandlerFunc(userImpersonater)
}

func userImpersonater(w http.ResponseWriter, r *http.Request) {
	loggedInUser := r.Context().Value("user").(models.User)
	if !loggedInUser.Admin {
		errRes(w, r, 403, "You are not an admin", nil)
		return
	}

	vars := mux.Vars(r)
	user := models.User{}
	if err := user.FindByID(r.Context(), vars["id"]); err != nil {
		errRes(w, r, 500, "error fetching user", err)
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

	relatedOrganisations := r.Context().Value("organisations").(models.Organisations)

	if err := Tmpl.ExecuteTemplate(w, "welcome.html", welcomePageData{
		Organisations: relatedOrganisations,
		Context:       r.Context(),
	}); err != nil {
		errRes(w, r, 500, "Problem with template", err)
		return
	}
}

func userCreateOrUpdateHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()

	required := []string{
		"email",
	}

	okay := checkFormInput(required, r.Form, w, r)
	if !okay {
		return
	}

	user := models.User{}
	if r.FormValue("id") != "" {
		if err := user.FindByID(r.Context(), r.FormValue("id")); err != nil {
			errRes(w, r, 400, "Specified ID does not exist", err)
			return
		}
		if user.Revision != r.FormValue("revision") {
			errRes(w, r, http.StatusBadRequest, models.ErrWrongRev.Message, nil)
			return
		}
	} else {
		if err := user.FindByColumn(r.Context(), "email", r.FormValue("email")); err != nil {
			if err != sql.ErrNoRows {
				errRes(w, r, 500, "Error looking up user", err)
				return
			}
		}
	}

	newUser := false

	if user.ID == "" {
		newUser = true
		if r.FormValue("terms") != "agreed" {
			errRes(w, r, 400, "You must agree to the terms and conditions", nil)
			return
		}
		rawpassword := r.FormValue("password")
		if rawpassword == "" {
			rawpassword = uuid.NewV4().String()
		}

		user.New(
			r.FormValue("email"),
			rawpassword,
		)
	} else {
		if r.FormValue("email") != user.Email {
			if user.HasEmail() {
				if !user.Verified {
					if r.FormValue("terms") != "agreed" {
						errRes(w, r, http.StatusBadRequest, "You must accept the terms and conditions if you wish to sign up", nil)
						return
					}
					orgs := models.Organisations{}
					if err := orgs.FindAll(r.Context(), models.OrganisationsContainingUser{ID: user.ID}); err != nil {
						errRes(w, r, 500, "Error looking up organisations", err)
						return
					}
					for _, org := range orgs.Data {
						if err := user.SendVerificationEmail(r.Context(), org); err != nil {
							errRes(w, r, 500, "Error queueing verification email", err)
							return
						}
					}
				} else {
					if err := sendEmailChangedNotification(r.Context(), r.FormValue("email"), user.Email); err != nil {
						errRes(w, r, 500, "Error queueing notification", err)
						return
					}
				}
			}
		}
		if r.FormValue("email") != "noop" {
			user.Email = r.FormValue("email")
		}
		if r.FormValue("password") != "" {
			hash, err := models.HashPassword(r.FormValue("password"))
			if err != nil {
				errRes(w, r, 500, "Error creating password hash.", err)
				return
			}
			user.Password = hash
		}

		if !user.Verified {
			if r.FormValue("terms") != "agreed" {
				errRes(w, r, http.StatusBadRequest, "You must accept the terms and conditions if you wish to sign up", nil)
				return
			}
			user.Verified = true
		}
	}

	savePermitted := newUser
	untypedUser := r.Context().Value("user")
	if untypedUser != nil {
		loggedInUser := untypedUser.(models.User)
		if loggedInUser.Admin || user.ID == loggedInUser.ID {
			savePermitted = true
		}
	}

	if r.FormValue("token") != "" {
		expiry := r.FormValue("expiry")
		if err := checkTokenExpiry(expiry); err != nil {
			errRes(w, r, 400, "Your invitation has expired. Please use the password reset function of the login form", err)
			return
		}

		expectedToken := util.CalcToken(user.Email, expiry)
		if r.FormValue("token") == expectedToken {
			savePermitted = true
		}

	}

	if savePermitted {
		if err := user.Save(r.Context()); err != nil {
			if err.Error() == `pq: duplicate key value violates unique constraint "users_email_key"` {
				config.ReportError(errors.New("Duplicate email hit: " + user.Email))
				errRes(w, r, 409, "That email address already exists in our system.", err)
				return
			}
			errRes(w, r, 500, "A database error has occurred", err)
			return
		}
	}

	orgname := r.FormValue("orgname")
	orgcountry := r.FormValue("orgcountry")
	orgcurrency := r.FormValue("currency")
	createdOrg := models.Organisation{}
	if orgname != "" {
		var err error
		err, createdOrg = createOrgFromSignup(r.Context(), user, orgname, orgcountry, orgcurrency)
		if err != nil {
			if err.Error() == `pq: duplicate key value violates unique constraint "users_email_key"` {
				errRes(w, r, 409, "That email address already exists in our system.", err)
				return
			}
			errRes(w, r, 500, "Error saving new organisation", err)
			return
		}
	}

	if !user.Verified {
		if err := user.SendVerificationEmail(r.Context(), createdOrg); err != nil {
			errRes(w, r, 500, "Error sending verification email", err)
			return
		}
	}

	defaultNext := "/users/" + user.ID

	if user.Verified && orgname != "" {
		defaultNext = "/organisations"
	}

	http.Redirect(w, r, nextFlow(defaultNext, r.Form), 302)
}

type usersPageData struct {
	Users   models.Users
	Context context.Context
}

func usersHandler(w http.ResponseWriter, r *http.Request) {
	loggedInUser := r.Context().Value("user").(models.User)
	if !loggedInUser.Admin {
		errRes(w, r, http.StatusForbidden, "Only application admins may list users", nil)
		return
	}

	if r.FormValue("email") != "" {
		user := models.User{}
		if err := user.FindByColumn(r.Context(), "email", r.FormValue(("email"))); err != nil {
			if err == sql.ErrNoRows {
				errRes(w, r, http.StatusNotFound, "No user found with that email address", nil)
				return
			}
			errRes(w, r, http.StatusInternalServerError, "Database error", err)
			return
		}
		http.Redirect(w, r, "/users/"+user.ID, 302)
		return
	}

	query := models.All{}
	query.DefaultPageSize = 50
	query.Paginate(r.Form)

	users := models.Users{}
	if err := users.FindAll(r.Context(), query); err != nil {
		errRes(w, r, 500, "error fetching users", err)
		return
	}

	if err := Tmpl.ExecuteTemplate(w, "users.html", usersPageData{
		Users:   users,
		Context: r.Context(),
	}); err != nil {
		errRes(w, r, 500, "Templating error", err)
		return
	}
}

func userHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	loggedInUser := r.Context().Value("user").(models.User)
	if loggedInUser.ID != vars["id"] && !loggedInUser.Admin {
		errRes(w, r, http.StatusForbidden, "You are not authorized to view this user", nil)
		return
	}

	user := models.User{}
	if err := user.FindByID(r.Context(), vars["id"]); err != nil {
		errRes(w, r, 500, "error fetching user", err)
		return
	}

	orgs := map[string]models.Organisation{}
	for _, org := range orgsFromContext(r.Context()).Data {
		orgs[org.ID] = org
	}

	if err := Tmpl.ExecuteTemplate(w, "user.html", userPageData{
		User:     user,
		OrgsByID: orgs,
		Context:  r.Context(),
	}); err != nil {
		errRes(w, r, 500, "Problem with template", err)
		return
	}
}

func userSettingsRedir(w http.ResponseWriter, r *http.Request) {
	user := r.Context().Value("user").(models.User)
	http.Redirect(w, r, "/users/"+user.ID, 302)
}

type userPageData struct {
	Context  context.Context
	User     models.User
	OrgsByID map[string]models.Organisation
}

func createOrgFromSignup(ctx context.Context, user models.User, orgname, orgcountry, orgcurrency string) (error, models.Organisation) {
	org := models.Organisation{}
	org.New(
		orgname,
		orgcountry,
	)

	if err := org.Save(ctx); err != nil {
		return err, org
	}

	orgUser := models.OrganisationUser{}
	orgUser.New(user.ID, org.ID, models.Roles{
		models.Role{
			Name: "admin",
		},
	})
	if err := orgUser.Save(ctx); err != nil {
		return err, org
	}

	if err := copySampleOrgData(ctx, org); err != nil {
		return err, org
	}

	orgUser.OrganisationID = org.ID

	if err := orgUser.Save(ctx); err != nil {
		return err, org
	}
	return nil, org
}

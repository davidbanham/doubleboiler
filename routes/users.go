package routes

import (
	"context"
	"doubleboiler/config"
	"doubleboiler/models"
	m "doubleboiler/models"
	"doubleboiler/util"
	"errors"
	"net/http"
	"time"

	kewpie "github.com/davidbanham/kewpie_go/v3"
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
	loggedInUser := r.Context().Value("user").(m.User)
	if !loggedInUser.Admin {
		errRes(w, r, 403, "You are not an admin", nil)
		return
	}

	vars := mux.Vars(r)
	u := m.User{}
	if err := u.FindByID(r.Context(), vars["id"]); err != nil {
		errRes(w, r, 500, "error fetching user", err)
		return
	}

	expiration := time.Now().Add(30 * 24 * time.Hour)
	encoded, err := secureCookie.Encode("doubleboiler-user", map[string]string{
		"ID": u.ID,
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

	relatedOrganisations := r.Context().Value("organisations").(m.Organisations)

	if err := Tmpl.ExecuteTemplate(w, "welcome.html", welcomePageData{
		Organisations: relatedOrganisations,
		ActiveOrg:     m.Organisation{},
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

	flow := ""

	u := m.User{}
	if r.FormValue("id") != "" {
		if err := u.FindByID(r.Context(), r.FormValue("id")); err != nil {
			errRes(w, r, 400, "Specified ID does not exist", err)
			return
		}
	} else {
		u.FindByColumn(r.Context(), "email", r.FormValue("email"))
		// Ignore errors since we might be just trying to create this user
	}

	newUser := false

	if u.ID == "" {
		newUser = true
		if r.FormValue("terms") != "agreed" {
			errRes(w, r, 400, "You must agree to the terms and conditions", nil)
			return
		}
		rawpassword := r.FormValue("password")
		if rawpassword == "" {
			rawpassword = uuid.NewV4().String()
		}

		u.New(
			r.FormValue("email"),
			rawpassword,
		)
	} else {
		targetID := r.FormValue("id")
		if u.ID != "" {
			targetID = u.ID
		}

		u.FindByID(r.Context(), targetID)
		if r.FormValue("email") != u.Email {

			task := kewpie.Task{}
			if err := task.Marshal(u); err != nil {
				errRes(w, r, 500, "Error queueing stripe member record update", err)
				return
			}

			if u.HasEmail() {
				if !u.Verified {
					orgs := models.Organisations{}
					if err := orgs.FindAll(r.Context(), models.OrganisationsContainingUser, u.ID); err != nil {
						errRes(w, r, 500, "Error looking up organisations", err)
						return
					}
					for _, org := range orgs {
						if err := u.SendVerificationEmail(r.Context(), org); err != nil {
							errRes(w, r, 500, "Error queueing verification email", err)
							return
						}
					}
				} else {
					if err := sendEmailChangedNotification(r.Context(), r.FormValue("email"), u.Email); err != nil {
						errRes(w, r, 500, "Error queueing notification", err)
						return
					}
				}
			}
		}
		if r.FormValue("email") != "noop" {
			u.Email = r.FormValue("email")
		}
		if r.FormValue("password") != "" {
			if r.FormValue("token") != "" {
				flow = "signup"
			}
			hash, err := m.HashPassword(r.FormValue("password"))
			if err != nil {
				errRes(w, r, 500, "Error creating password hash.", err)
				return
			}
			u.Password = hash
		}

		if !u.Verified {
			u.Verified = true
		}
	}

	savePermitted := newUser
	untypedUser := r.Context().Value("user")
	if untypedUser != nil {
		loggedInUser := untypedUser.(m.User)
		if loggedInUser.Admin || u.ID == loggedInUser.ID {
			savePermitted = true
		}
	}

	if r.FormValue("token") != "" {
		expiry := r.FormValue("expiry")
		if err := checkTokenExpiry(expiry); err != nil {
			errRes(w, r, 400, "Your invitation has expired. Please use the password reset function of the login form", err)
			return
		}

		expectedToken := util.CalcToken(u.Email, expiry)
		if r.FormValue("token") == expectedToken {
			savePermitted = true
		}

	}

	if savePermitted {
		if err := u.Save(r.Context()); err != nil {
			if err.Error() == `pq: duplicate key value violates unique constraint "users_email_key"` {
				config.ReportError(errors.New("Duplicate email hit: " + u.Email))
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
	createdOrg := m.Organisation{}
	if orgname != "" {
		var err error
		err, createdOrg = createOrgFromSignup(r.Context(), u, orgname, orgcountry, orgcurrency)
		if err != nil {
			if err.Error() == `pq: duplicate key value violates unique constraint "users_email_key"` {
				errRes(w, r, 409, "That email address already exists in our system.", err)
				return
			}
			errRes(w, r, 500, "Error saving new organisation", err)
			return
		}
	}

	if !u.Verified {
		if err := u.SendVerificationEmail(r.Context(), createdOrg); err != nil {
			errRes(w, r, 500, "Error sending verification email", err)
			return
		}
	}

	next := "/users/" + u.ID
	if r.FormValue("next") != "" {
		next = r.FormValue("next")
	}

	if u.Verified && orgname != "" {
		next = "/organisations"
	}

	if flow != "" {
		next += "?flow=" + flow
	}

	http.Redirect(w, r, next, 302)
}

type usersPageData struct {
	Users   m.Users
	Context context.Context
}

func usersHandler(w http.ResponseWriter, r *http.Request) {
	user := r.Context().Value("user").(m.User)
	if !user.Admin {
		errRes(w, r, http.StatusForbidden, "Only application admins may list users", nil)
		return
	}

	u := m.Users{}
	err := u.FindAll(r.Context(), m.All, "")
	if err != nil {
		errRes(w, r, 500, "error fetching users", err)
		return
	}

	if err := Tmpl.ExecuteTemplate(w, "users.html", usersPageData{
		Users:   u,
		Context: r.Context(),
	}); err != nil {
		errRes(w, r, 500, "Templating error", err)
		return
	}
}

func userHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	user := r.Context().Value("user").(m.User)
	if user.ID != vars["id"] && !user.Admin {
		errRes(w, r, http.StatusForbidden, "You are not authorized to view this user", nil)
		return
	}

	u := m.User{}
	err := u.FindByID(r.Context(), vars["id"])
	if err != nil {
		errRes(w, r, 500, "error fetching user", err)
		return
	}

	orgs := map[string]m.Organisation{}
	for _, org := range orgsFromContext(r.Context()) {
		orgs[org.ID] = org
	}

	Tmpl.ExecuteTemplate(w, "user.html", userPageData{
		TargetUser: u,
		OrgsByID:   orgs,
		Context:    r.Context(),
	})
}

func userSettingsRedir(w http.ResponseWriter, r *http.Request) {
	user := r.Context().Value("user").(m.User)
	http.Redirect(w, r, "/users/"+user.ID, 302)
}

type userPageData struct {
	Context    context.Context
	TargetUser m.User
	OrgsByID   map[string]m.Organisation
}

func createOrgFromSignup(ctx context.Context, u m.User, orgname, orgcountry, orgcurrency string) (error, m.Organisation) {
	orgUser := m.OrganisationUser{}
	orgUser.New(u.ID, "", models.Roles{"admin": true})

	org := m.Organisation{}
	org.New(
		orgname,
		orgcountry,
		[]m.OrganisationUser{
			orgUser,
		},
		orgcurrency,
	)

	if err := org.Save(ctx); err != nil {
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

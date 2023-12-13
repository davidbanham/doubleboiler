package routes

import (
	"bytes"
	"context"
	"database/sql"
	"doubleboiler/config"
	"doubleboiler/flashes"
	"doubleboiler/models"
	"doubleboiler/util"
	"errors"
	"image/png"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	uuid "github.com/satori/go.uuid"
	"golang.org/x/crypto/bcrypt"
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

	r.Path("/users/{id}/generate-totp").
		Methods("POST", "GET").
		HandlerFunc(userGenerateTOTPHandler)

	r.Path("/users/{id}/validate-totp").
		Methods("GET").
		HandlerFunc(userValidateTOTPFormHandler)

	r.Path("/users/{id}/enrol-totp").
		Methods("POST").
		HandlerFunc(userEnrolTOTPHandler)

	r.Path("/users/{id}/show-recovery-codes").
		Methods("POST").
		HandlerFunc(userShowRecoveryCodesHandler)

	r.Path("/users/{id}/disable-totp").
		Methods("POST").
		HandlerFunc(userDisableTOTPHandler)
}

func userImpersonater(w http.ResponseWriter, r *http.Request) {
	loggedInUser := r.Context().Value("user").(models.User)
	if !loggedInUser.SuperAdmin {
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

	orgs := orgsFromContext(r.Context())

	if err := Tmpl.ExecuteTemplate(w, "dashboard.html", dashboardPageData{
		Organisations: orgs,
		basePageData: basePageData{
			PageTitle: "DoubleBoiler - Dashboard",
			Context:   r.Context(),
		},
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
					criteria := models.Criteria{}
					models.AddCustomQuery(models.OrganisationsContainingUser{ID: user.ID}, &criteria)
					if err := orgs.FindAll(r.Context(), criteria); err != nil {
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
					if err := user.SendEmailChangedNotification(r.Context(), r.FormValue("email")); err != nil {
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
			if r.FormValue("confirm-password") != "" {
				if r.FormValue("confirm-password") != r.FormValue("password") {
					errRes(w, r, http.StatusBadRequest, "Submitted passwords do not match", nil)
					return
				}
			}
			hash, err := util.HashPassword(r.FormValue("password"))
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
		if loggedInUser.SuperAdmin || user.ID == loggedInUser.ID {
			savePermitted = true
		}
	}

	if r.FormValue("token") != "" {
		if err := util.CheckToken(config.SECRET, r.FormValue("expiry"), user.Email, r.FormValue("token")); err != nil {
			errRes(w, r, http.StatusUnauthorized, "Invalid token", err)
			return
		} else {
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
			if r.FormValue("password") != "" {
				if util.Contains([]string{"reset_password", "signup"}, r.FormValue("flow")) {
					flash := flashes.Flash{
						Persistent: true,
						Type:       flashes.Success,
						Text:       "Please use your new password to log in.",
					}
					flashed, _ := flash.Add(r.Context())
					r = r.WithContext(flashed)
				} else {
					flash := flashes.Flash{
						Persistent: true,
						Type:       flashes.Success,
						Text:       "Password set successfully.",
					}
					flashed, _ := flash.Add(r.Context())
					r = r.WithContext(flashed)
				}
			}
			errRes(w, r, 500, "A database error has occurred", err)
			return
		}
	}

	orgname := r.FormValue("orgname")
	orgcountry := r.FormValue("country")
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
	basePageData
	Users   models.Users
	Context context.Context
}

func usersHandler(w http.ResponseWriter, r *http.Request) {
	loggedInUser := r.Context().Value("user").(models.User)
	if !loggedInUser.SuperAdmin {
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

	users := models.Users{}

	criteria := models.Criteria{
		Query: &models.All{},
	}
	criteria.Pagination.DefaultPageSize = 50
	criteria.Pagination.Paginate(r.Form)

	criteria.Filters.FromForm(r.Form, users.AvailableFilters())

	if err := users.FindAll(r.Context(), criteria); err != nil {
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
	if loggedInUser.ID != vars["id"] && !loggedInUser.SuperAdmin {
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
	basePageData
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

func userGenerateTOTPHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	user := models.User{}
	if err := user.FindByID(r.Context(), vars["id"]); err != nil {
		errRes(w, r, 500, "error fetching user", err)
		return
	}

	targetOrg := activeOrgFromContext(r.Context())
	loggedInUser := userFromContext(r.Context())

	if loggedInUser.ID != user.ID && !can(r.Context(), targetOrg, "superadmin") {
		errRes(w, r, 403, "You are not logged in as this user, nor are you an application admin", nil)
		return
	}

	if loggedInUser.ID != user.ID && !can(r.Context(), targetOrg, "superadmin") && user.TOTPActive {
		if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(r.FormValue("password"))); err != nil {
			errRes(w, r, 403, "Incorrect password", err)
			return
		}
	}

	key, err := user.Generate2FA(r.Context(), r.FormValue("totp-code"), r.FormValue("totp-recovery-code"))
	if err != nil {
		errRes(w, r, http.StatusInternalServerError, "Error generating 2-step auth code", err)
		return
	}

	// Convert TOTP key into a QR code encoded as a PNG image.
	var buf bytes.Buffer
	img, err := key.Image(200, 200)
	png.Encode(&buf, img)

	orgs := orgsFromContext(r.Context())

	if err := Tmpl.ExecuteTemplate(w, "generate-totp.html", totpPageData{
		User:          user,
		Organisations: orgs,
		ActiveOrg:     models.Organisation{},
		Context:       r.Context(),
		TOTPQRImage:   buf,
		TOTPSecret:    key.Secret(),
	}); err != nil {
		errRes(w, r, 500, "Problem with template", err)
		return
	}
}

func userValidateTOTPFormHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	user := models.User{}
	if err := user.FindByID(r.Context(), vars["id"]); err != nil {
		errRes(w, r, 500, "error fetching user", err)
		return
	}

	targetOrg := activeOrgFromContext(r.Context())
	loggedInUser := userFromContext(r.Context())

	if loggedInUser.ID != user.ID && !can(r.Context(), targetOrg, "superadmin") {
		errRes(w, r, 403, "You are not logged in as this user, nor are you an application admin", nil)
		return
	}

	orgs := orgsFromContext(r.Context())

	if err := Tmpl.ExecuteTemplate(w, "generate-totp.html", totpPageData{
		User:          user,
		Organisations: orgs,
		ActiveOrg:     models.Organisation{},
		Context:       r.Context(),
		TOTPQRImage:   bytes.Buffer{},
	}); err != nil {
		errRes(w, r, 500, "Problem with template", err)
		return
	}
}

func userEnrolTOTPHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	user := models.User{}
	if err := user.FindByID(r.Context(), vars["id"]); err != nil {
		errRes(w, r, 500, "error fetching user", err)
		return
	}

	targetOrg := activeOrgFromContext(r.Context())
	loggedInUser := userFromContext(r.Context())

	if loggedInUser.ID != user.ID && !can(r.Context(), targetOrg, "superadmin") {
		errRes(w, r, 403, "You are not logged in as this user, nor are you an application admin", nil)
		return
	}

	if ok, err := user.Validate2FA(r.Context(), r.FormValue("totp-code"), r.FormValue("totp-recovery-code")); err != nil {
		errRes(w, r, http.StatusInternalServerError, "Error validating 2-step auth code", err)
		return
	} else if !ok {
		errRes(w, r, http.StatusForbidden, "Provided 2FA code did not match expected value", nil)
		return
	}

	flash := flashes.Flash{
		Persistent: true,
		Type:       flashes.Success,
		Text:       "Two factor auth successfully set up",
	}
	if flashed, err := flash.Add(r.Context()); err != nil {
		errRes(w, r, http.StatusInternalServerError, "Error adding flash message", err)
		return
	} else {
		r = r.WithContext(flashed)
	}

	codes, err := user.GenerateRecoveryCodesBypassCheck(r.Context())
	if err != nil {
		errRes(w, r, http.StatusInternalServerError, "Error generating recovery codes", err)
		return
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

	if err := Tmpl.ExecuteTemplate(w, "recovery-codes.html", recoveryCodesPageData{
		User:      user,
		Context:   r.Context(),
		Codes:     codes,
		Generated: time.Now(),
	}); err != nil {
		errRes(w, r, 500, "Problem with template", err)
		return
	}
}

func userShowRecoveryCodesHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	user := models.User{}
	if err := user.FindByID(r.Context(), vars["id"]); err != nil {
		errRes(w, r, 500, "error fetching user", err)
		return
	}

	targetOrg := activeOrgFromContext(r.Context())
	loggedInUser := userFromContext(r.Context())

	if loggedInUser.ID != user.ID && !can(r.Context(), targetOrg, "superadmin") {
		errRes(w, r, 403, "You are not logged in as this user, nor are you an application admin", nil)
		return
	}

	if !user.TOTPActive {
		errRes(w, r, http.StatusBadRequest, "Two factor authentication is not enabled for this user account", nil)
		return
	}

	if loggedInUser.ID == user.ID {
		if ok, err := user.Validate2FA(r.Context(), r.FormValue("totp-code"), r.FormValue("totp-recovery-code")); err != nil {
			errRes(w, r, http.StatusInternalServerError, "Error validating 2-step auth code", err)
			return
		} else if !ok {
			errRes(w, r, http.StatusForbidden, "Provided 2FA code did not match expected value", nil)
			return
		}

		passwordFailed := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(r.FormValue("password")))
		if passwordFailed != nil {
			errRes(w, r, 403, "Incorrect password", nil)
			return
		}
	}

	codes, err := user.GenerateRecoveryCodes(r.Context(), r.FormValue("totp-code"))
	if err != nil {
		errRes(w, r, http.StatusInsufficientStorage, "error generating recovery codes", err)
		return
	}

	orgs := orgsFromContext(r.Context())

	if err := Tmpl.ExecuteTemplate(w, "recovery-codes.html", recoveryCodesPageData{
		User:          user,
		Organisations: orgs,
		ActiveOrg:     models.Organisation{},
		Context:       r.Context(),
		Codes:         codes,
		Generated:     time.Now(),
	}); err != nil {
		errRes(w, r, 500, "Problem with template", err)
		return
	}
}

func userDisableTOTPHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	user := models.User{}
	if err := user.FindByID(r.Context(), vars["id"]); err != nil {
		errRes(w, r, 500, "error fetching user", err)
		return
	}

	targetOrg := activeOrgFromContext(r.Context())
	loggedInUser := userFromContext(r.Context())

	if loggedInUser.ID != user.ID && !can(r.Context(), targetOrg, "superadmin") {
		errRes(w, r, 403, "You are not logged in as this user, nor are you an application admin", nil)
		return
	}

	if !loggedInUser.SuperAdmin || user.SuperAdmin {
		if ok, err := user.Validate2FA(r.Context(), r.FormValue("totp-code"), r.FormValue("totp-recovery-code")); err != nil {
			errRes(w, r, http.StatusInternalServerError, "Error validating 2-step auth code", err)
			return
		} else if !ok {
			errRes(w, r, http.StatusForbidden, "Provided 2FA code did not match expected value", nil)
			return
		}
	}

	if err := user.Disable2FA(r.Context()); err != nil {
		errRes(w, r, http.StatusForbidden, "Error disabling 2-step auth", err)
		return
	}

	flash := flashes.Flash{
		Persistent: true,
		Type:       flashes.Success,
		Text:       "Two factor auth successfully removed",
	}

	if flashed, err := flash.Add(r.Context()); err != nil {
		errRes(w, r, http.StatusInternalServerError, "Error adding flash message", err)
		return
	} else {
		r = r.WithContext(flashed)
	}

	http.Redirect(w, r, "/users/"+user.ID, http.StatusFound)
}

type recoveryCodesPageData struct {
	basePageData
	ActiveOrg     models.Organisation
	Organisations models.Organisations
	User          models.User
	Context       context.Context
	Codes         []string
	Generated     time.Time
}

type totpPageData struct {
	basePageData
	User          models.User
	ActiveOrg     models.Organisation
	Organisations models.Organisations
	Context       context.Context
	TOTPQRImage   bytes.Buffer
	TOTPSecret    string
}

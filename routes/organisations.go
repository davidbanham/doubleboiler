package routes

import (
	"context"
	"database/sql"
	"doubleboiler/config"
	"doubleboiler/models"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
)

func init() {
	r.Path("/create-organisation").
		Methods("GET").
		HandlerFunc(organisationCreationFormHandler)

	r.Path("/organisations").
		Methods("POST").
		HandlerFunc(organisationCreateOrUpdateHandler)

	r.Path("/organisations").
		Methods("GET").
		HandlerFunc(organisationsHandler)

	r.Path("/organisations/{id}").
		Methods("GET").
		HandlerFunc(organisationHandler)

	r.Path("/organisations/{id}").
		Methods("POST").
		HandlerFunc(organisationCreateOrUpdateHandler)
}

type orgCreationPageData struct {
	Context context.Context
	User    models.User
}

func organisationCreationFormHandler(w http.ResponseWriter, r *http.Request) {
	if err := Tmpl.ExecuteTemplate(w, "create-organisation.html", orgCreationPageData{
		Context: r.Context(),
		User:    r.Context().Value("user").(models.User),
	}); err != nil {
		errRes(w, r, 500, "Templating error", err)
		return
	}
}

func organisationCreateOrUpdateHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	user := r.Context().Value("user").(models.User)

	required := []string{
		"name",
	}

	okay := checkFormInput(required, r.Form, w, r)
	if !okay {
		return
	}

	vars := mux.Vars(r)
	if vars["id"] != r.FormValue("id") {
		errRes(w, r, 500, "Submitted ID does not match path", nil)
		return
	}

	r.Form["custom_fields"] = deblank(r.Form["custom_fields"])
	r.Form["read_only_fields"] = deblank(r.Form["read_only_fields"])
	r.Form["private_fields"] = deblank(r.Form["private_fields"])

	var org models.Organisation

	// Org already exists. This is an update.
	if r.FormValue("id") != "" {
		if err := org.FindByID(r.Context(), r.FormValue("id")); err != nil {
			errRes(w, r, 500, "Error looking up organisation", err)
			return
		}

		org.Name = r.FormValue("name")
		org.Country = r.FormValue("country")
	} else {
		// Org doesn't exist. Let's create it.
		org = models.Organisation{}
		org.New(
			r.FormValue("name"),
			r.FormValue("country"),
		)

		if err := org.Save(r.Context()); err != nil {
			errRes(w, r, 500, "A database error has occurred", err)
			return
		}

		if err := copySampleOrgData(r.Context(), org); err != nil {
			errRes(w, r, 500, "Error creating sample data", err)
			return
		}

		ou := models.OrganisationUser{}
		ou.New(user.ID, org.ID, models.Roles{
			models.Role{
				Name: "admin",
			},
		})
		if err := ou.Save(r.Context()); err != nil {
			errRes(w, r, 500, "A database error has occurred", err)
			return
		}

	}

	lowered := strings.ToLower(org.Country)
	if lowered == "aus" || lowered == "australia" || lowered == "au" {
		org.Country = "Australia"
	}

	if err := org.Save(r.Context()); err != nil {
		errRes(w, r, 500, "A database error has occurred", err)
		return
	}

	http.Redirect(w, r, "/organisations/"+org.ID, 302)
}

type organisationsPageData struct {
	Organisations models.Organisations
	Context       context.Context
}

func organisationsHandler(w http.ResponseWriter, r *http.Request) {
	targetOrg := activeOrgFromContext(r.Context())
	if targetOrg.ID == "" {
		redirToDefaultOrg(w, r)
		return
	}

	Tmpl.ExecuteTemplate(w, "organisations.html", organisationsPageData{
		Organisations: orgsFromContext(r.Context()),
		Context:       r.Context(),
	})
}

type organisationPageData struct {
	Context           context.Context
	Organisation      models.Organisation
	OrganisationUsers models.OrganisationUsers
	URI               string
	ProductName       string
	ValidRoles        map[string]models.Role
}

func organisationHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	targetOrg := orgFromContext(r.Context(), vars["id"])

	if !can(r.Context(), targetOrg, "admin") {
		errRes(w, r, http.StatusForbidden, "Only admins may view organisation settings", nil)
		return
	}

	if err := targetOrg.FindByID(r.Context(), vars["id"]); err != nil {
		errRes(w, r, http.StatusInternalServerError, "Error looking up organisation", err)
		return
	}

	orgUsers := models.OrganisationUsers{}
	if err := orgUsers.FindAll(r.Context(), models.ByOrg{ID: targetOrg.ID}); err != nil {
		errRes(w, r, http.StatusInternalServerError, "Error looking up organisation users", err)
		return
	}

	if err := Tmpl.ExecuteTemplate(w, "organisation.html", organisationPageData{
		Organisation:      targetOrg,
		OrganisationUsers: orgUsers,
		ValidRoles:        models.ValidRoles,
		ProductName:       config.NAME,
		Context:           r.Context(),
		URI:               config.URI,
	}); err != nil {
		errRes(w, r, http.StatusInternalServerError, "Templating error", err)
		return
	}
}

func copySampleOrgData(ctx context.Context, target models.Organisation) error {
	sampleOrg := models.Organisation{}
	if err := sampleOrg.FindByID(ctx, config.SAMPLEORG_ID); err != nil {
		if err == sql.ErrNoRows {
			return nil
		}
		return err
	}

	if err := target.Save(ctx); err != nil {
		return err
	}

	return nil
}

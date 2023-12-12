package routes

import (
	"context"
	"database/sql"
	"doubleboiler/config"
	"doubleboiler/models"
	"doubleboiler/util"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
)

func init() {
	r.Path("/organisations").
		Methods("POST").
		HandlerFunc(organisationCreateOrUpdateHandler)

	r.Path("/organisations/{id}").
		Methods("POST").
		HandlerFunc(organisationCreateOrUpdateHandler)

	r.Path("/organisations").
		Methods("GET").
		HandlerFunc(organisationsHandler)

	r.Path("/organisations/create").
		Methods("GET").
		HandlerFunc(organisationCreationFormHandler)

	r.Path("/organisations/{id}").
		Methods("GET").
		HandlerFunc(organisationHandler)

	r.Path("/organisation-settings").
		Methods("GET").
		HandlerFunc(organisationSettingsHandler)
}

type orgCreationPageData struct {
	basePageData
	User models.User
}

func organisationCreationFormHandler(w http.ResponseWriter, r *http.Request) {
	if err := Tmpl.ExecuteTemplate(w, "create-organisation.html", orgCreationPageData{
		basePageData: basePageData{
			PageTitle: "DoubleBoiler - Create Organisation",
			Context:   r.Context(),
		},
		User: r.Context().Value("user").(models.User),
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

	var org models.Organisation

	// Org already exists. This is an update.
	if r.FormValue("id") != "" {
		if err := org.FindByID(r.Context(), r.FormValue("id")); err != nil {
			errRes(w, r, 500, "Error looking up organisation", err)
			return
		}

		if org.Revision != r.FormValue("revision") {
			errRes(w, r, http.StatusBadRequest, models.ErrWrongRev.Message, nil)
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

	org.Toggles.FromForm(r.Form)

	if err := org.Save(r.Context()); err != nil {
		errRes(w, r, 500, "A database error has occurred", err)
		return
	}

	http.Redirect(w, r, nextFlow("/organisations/"+org.ID, r.Form), 302)
}

type organisationsPageData struct {
	basePageData
	Organisations models.Organisations
}

func organisationsHandler(w http.ResponseWriter, r *http.Request) {
	targetOrg := activeOrgFromContext(r.Context())
	if targetOrg.ID == "" {
		redirToDefaultOrg(w, r)
		return
	}

	organisations := models.Organisations{}

	criteria := models.Criteria{
		Query: &models.All{},
	}
	criteria.Pagination.DefaultPageSize = 50
	criteria.Pagination.Paginate(r.Form)

	criteria.Filters.FromForm(r.Form, organisations.AvailableFilters())

	if err := organisations.FindAll(r.Context(), criteria); err != nil {
		errRes(w, r, 500, "error fetching organisations", err)
		return
	}

	filtered := models.Organisations{}
	whitelisted := orgsFromContext(r.Context()).ByID()
	for _, org := range organisations.Data {
		if whitelisted[org.ID].ID == org.ID {
			filtered.Data = append(filtered.Data, org)
		}
	}

	filtered.Criteria = criteria

	if err := Tmpl.ExecuteTemplate(w, "organisations.html", organisationsPageData{
		Organisations: filtered,
		basePageData: basePageData{
			PageTitle: "DoubleBoiler - Organisations",
			Context:   r.Context(),
		},
	}); err != nil {
		errRes(w, r, http.StatusInternalServerError, "Templating error", err)
		return
	}
}

type organisationPageData struct {
	basePageData
	Organisation      models.Organisation
	OrganisationUsers models.OrganisationUsers
	URI               string
	ProductName       string
	ValidRoles        models.Roles
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

	if err := orgUsers.FindAll(r.Context(), models.Criteria{Query: &models.ByOrg{ID: targetOrg.ID}}); err != nil {
		errRes(w, r, http.StatusInternalServerError, "Error looking up organisation users", err)
		return
	}

	if err := Tmpl.ExecuteTemplate(w, "organisation.html", organisationPageData{
		Organisation:      targetOrg,
		OrganisationUsers: orgUsers,
		ValidRoles:        models.ValidRoles,
		ProductName:       config.NAME,
		basePageData: basePageData{
			PageTitle: "DoubleBoiler - Organisation " + util.FirstFiveChars(targetOrg.ID),
			Context:   r.Context(),
		},
		URI: config.URI,
	}); err != nil {
		errRes(w, r, http.StatusInternalServerError, "Templating error", err)
		return
	}
}

func organisationSettingsHandler(w http.ResponseWriter, r *http.Request) {
	targetOrg := activeOrgFromContext(r.Context())
	http.Redirect(w, r, "/organisations/"+targetOrg.ID, 302)
	return
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

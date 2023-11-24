package routes

import (
	"doubleboiler/models"
	"doubleboiler/util"
	"net/http"

	"github.com/gorilla/mux"
)

func init() {
	r.Path("/some-things").
		Methods("POST").
		HandlerFunc(someThingCreateOrUpdateHandler)

	r.Path("/some-things/{id}").
		Methods("POST").
		HandlerFunc(someThingCreateOrUpdateHandler)

	r.Path("/some-things/create").
		Methods("GET").
		HandlerFunc(someThingCreationFormHandler)

	r.Path("/some-things").
		Methods("GET").
		HandlerFunc(someThingsHandler)

	r.Path("/some-things/{id}").
		Methods("GET").
		HandlerFunc(someThingHandler)
}

type someThingCreationPageData struct {
	basePageData
}

func someThingCreationFormHandler(w http.ResponseWriter, r *http.Request) {
	targetOrg := activeOrgFromContext(r.Context())
	if targetOrg.ID == "" {
		redirToDefaultOrg(w, r)
		return
	}

	if !can(r.Context(), targetOrg, "admin") {
		errRes(w, r, http.StatusForbidden, "You cannot create someThings for that organisation", nil)
		return
	}

	if err := Tmpl.ExecuteTemplate(w, "create-some-thing.html", someThingCreationPageData{
		basePageData: basePageData{
			PageTitle: "DoubleBoiler - Create SomeThing",
			Context:   r.Context(),
		},
	}); err != nil {
		errRes(w, r, http.StatusInternalServerError, "Templating error", err)
		return
	}
}

func someThingCreateOrUpdateHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	required := []string{
		"name",
		"description",
		"organisationID",
	}
	okay := checkFormInput(required, r.Form, w, r)
	if !okay {
		return
	}

	org := orgFromContext(r.Context(), r.FormValue("organisationID"))

	if !can(r.Context(), org, "admin") {
		errRes(w, r, http.StatusForbidden, "You cannot create someThings for that organisation", nil)
		return
	}

	var someThing models.SomeThing

	// SomeThing already exists. This is an update.
	if r.FormValue("id") != "" {
		err := someThing.FindByID(r.Context(), r.FormValue("id"))
		if err != nil {
			errRes(w, r, 500, "Error looking up someThing", err)
			return
		}

		someThing.Name = r.FormValue("name")
		someThing.Description = r.FormValue("description")
	} else {
		// SomeThing doesn't exist. Let's create it.

		someThing.New(
			r.FormValue("name"),
			r.FormValue("description"),
			r.FormValue("organisationID"),
		)
	}

	if err := someThing.Save(r.Context()); err != nil {
		errRes(w, r, 500, "A database error has occurred", err)
		return
	}

	http.Redirect(w, r, "/some-things/"+someThing.ID, 302)
}

type someThingPageData struct {
	basePageData
	SomeThing models.SomeThing
}

func someThingHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	someThing := models.SomeThing{}
	if err := someThing.FindByID(r.Context(), vars["id"]); err != nil {
		errRes(w, r, 500, "A database error has occurred", err)
		return
	}

	org := orgFromContext(r.Context(), someThing.OrganisationID)

	if !can(r.Context(), org, "admin") {
		errRes(w, r, http.StatusForbidden, "You cannot view someThings for that organisation", nil)
		return
	}

	if err := Tmpl.ExecuteTemplate(w, "some-thing.html", someThingPageData{
		basePageData: basePageData{
			PageTitle: "DoubleBoiler - SomeThing " + util.FirstFiveChars(someThing.ID),
			Context:   r.Context(),
		},
		SomeThing: someThing,
	}); err != nil {
		errRes(w, r, http.StatusInternalServerError, "Templating error", err)
		return
	}
}

type someThingsPageData struct {
	basePageData
	SomeThings models.SomeThings
}

func someThingsHandler(w http.ResponseWriter, r *http.Request) {
	targetOrg := activeOrgFromContext(r.Context())

	if targetOrg.ID == "" {
		redirToDefaultOrg(w, r)
		return
	}

	if !can(r.Context(), targetOrg, "admin") {
		errRes(w, r, http.StatusForbidden, "You cannot list someThings for that organisation", nil)
		return
	}

	someThings := models.SomeThings{}

	criteria := models.Criteria{
		Query: models.ByOrg{ID: targetOrg.ID},
	}
	criteria.Pagination.DefaultPageSize = 50
	criteria.Pagination.Paginate(r.Form)

	if err := criteria.Filters.FromForm(r.Form, someThings.AvailableFilters()); err != nil {
		errRes(w, r, http.StatusBadRequest, "error interpreting filters", err)
		return
	}

	if err := someThings.FindAll(r.Context(), criteria); err != nil {
		errRes(w, r, 500, "error fetching someThings", err)
		return
	}

	if err := Tmpl.ExecuteTemplate(w, "some-things.html", someThingsPageData{
		SomeThings: someThings,
		basePageData: basePageData{
			PageTitle: "DoubleBoiler - SomeThings",
			Context:   r.Context(),
		},
	}); err != nil {
		errRes(w, r, http.StatusInternalServerError, "Templating error", err)
		return
	}
}

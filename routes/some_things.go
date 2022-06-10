package routes

import (
	"context"
	"doubleboiler/models"
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

	r.Path("/create-some-things").
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
	Context context.Context
}

func someThingCreationFormHandler(w http.ResponseWriter, r *http.Request) {
	targetOrg := activeOrgFromContext(r.Context())
	if targetOrg.ID == "" {
		redirToDefaultOrg(w, r)
		return
	}

	if err := Tmpl.ExecuteTemplate(w, "create-some-thing.html", someThingCreationPageData{
		Context: r.Context(),
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
	SomeThing models.SomeThing
	Context   context.Context
}

func someThingHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	p := models.SomeThing{}
	err := p.FindByID(r.Context(), vars["id"])
	if err != nil {
		errRes(w, r, 500, "A database error has occurred", err)
		return
	}

	org := orgFromContext(r.Context(), p.OrganisationID)

	if !can(r.Context(), org, "admin") {
		errRes(w, r, http.StatusForbidden, "You cannot create someThings for that organisation", nil)
		return
	}

	if err := Tmpl.ExecuteTemplate(w, "some-thing.html", someThingPageData{
		Context:   r.Context(),
		SomeThing: p,
	}); err != nil {
		errRes(w, r, http.StatusInternalServerError, "Templating error", err)
		return
	}
}

type someThingsPageData struct {
	SomeThings models.SomeThings
	Context    context.Context
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

	query := models.ByOrg{ID: targetOrg.ID}
	query.DefaultPageSize = 50
	query.Paginate(r.Form)

	if err := someThings.FindAll(r.Context(), query); err != nil {
		errRes(w, r, 500, "error fetching someThings", err)
		return
	}

	if err := Tmpl.ExecuteTemplate(w, "some-things.html", someThingsPageData{
		SomeThings: someThings,
		Context:    r.Context(),
	}); err != nil {
		errRes(w, r, http.StatusInternalServerError, "Templating error", err)
		return
	}
}
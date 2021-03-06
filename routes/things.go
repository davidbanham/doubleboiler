package routes

import (
	"context"
	"doubleboiler/models"
	m "doubleboiler/models"
	"net/http"

	"github.com/gorilla/mux"
)

func init() {
	r.Path("/things").
		Methods("POST").
		HandlerFunc(thingCreateOrUpdateHandler)

	r.Path("/things/{id}").
		Methods("POST").
		HandlerFunc(thingCreateOrUpdateHandler)

	r.Path("/create-thing").
		Methods("GET").
		HandlerFunc(thingCreationFormHandler)

	r.Path("/things").
		Methods("GET").
		HandlerFunc(thingsHandler)

	r.Path("/things/{id}").
		Methods("GET").
		HandlerFunc(thingHandler)
}

type thingCreationPageData struct {
	Context   context.Context
	ActiveOrg m.Organisation
}

func thingCreationFormHandler(w http.ResponseWriter, r *http.Request) {
	targetOrg := activeOrgFromContext(r.Context())
	if targetOrg.ID == "" {
		redirToDefaultOrg(w, r)
		return
	}

	if err := Tmpl.ExecuteTemplate(w, "create-thing.html", thingCreationPageData{
		Context:   r.Context(),
		ActiveOrg: orgFromContext(r.Context(), targetOrg.ID),
	}); err != nil {
		errRes(w, r, http.StatusInternalServerError, "Templating error", err)
		return
	}
}

func thingCreateOrUpdateHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	required := []string{
		"name",
		"organisationID",
	}
	okay := checkFormInput(required, r.Form, w, r)
	if !okay {
		return
	}

	org := orgFromContext(r.Context(), r.FormValue("organisationID"))

	if !can(r.Context(), org, "admin") {
		errRes(w, r, http.StatusForbidden, "You cannot create things for that organisation", nil)
		return
	}

	var thing m.Thing

	// Thing already exists. This is an update.
	if r.FormValue("id") != "" {
		err := thing.FindByID(r.Context(), r.FormValue("id"))
		if err != nil {
			errRes(w, r, 500, "Error looking up thing", err)
			return
		}

		thing.Name = r.FormValue("name")
	} else {
		// Thing doesn't exist. Let's create it.

		thing.New(
			r.FormValue("name"),
			r.FormValue("organisationID"),
		)
	}

	if err := thing.Save(r.Context()); err != nil {
		errRes(w, r, 500, "A database error has occurred", err)
		return
	}

	http.Redirect(w, r, "/things/"+thing.ID, 302)
}

type thingPageData struct {
	Thing     m.Thing
	Context   context.Context
	ActiveOrg m.Organisation
}

func thingHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	p := m.Thing{}
	err := p.FindByID(r.Context(), vars["id"])
	if err != nil {
		errRes(w, r, 500, "A database error has occurred", err)
		return
	}

	org := orgFromContext(r.Context(), p.OrganisationID)

	if !can(r.Context(), org, "admin") {
		errRes(w, r, http.StatusForbidden, "You cannot create things for that organisation", nil)
		return
	}

	if err := Tmpl.ExecuteTemplate(w, "thing.html", thingPageData{
		Context:   r.Context(),
		ActiveOrg: org,
		Thing:     p,
	}); err != nil {
		errRes(w, r, http.StatusInternalServerError, "Templating error", err)
		return
	}
}

type thingsPageData struct {
	Things    models.Things
	Context   context.Context
	ActiveOrg m.Organisation
}

func thingsHandler(w http.ResponseWriter, r *http.Request) {
	targetOrg := activeOrgFromContext(r.Context())

	if targetOrg.ID == "" {
		redirToDefaultOrg(w, r)
		return
	}

	if !can(r.Context(), targetOrg, "admin") {
		errRes(w, r, http.StatusForbidden, "You cannot list things for that organisation", nil)
		return
	}

	things := m.Things{}

	err := things.FindAll(r.Context(), m.ByOrg, targetOrg.ID)
	if err != nil {
		errRes(w, r, 500, "error fetching things", err)
		return
	}

	if err := Tmpl.ExecuteTemplate(w, "things.html", thingsPageData{
		Things:    things,
		Context:   r.Context(),
		ActiveOrg: targetOrg,
	}); err != nil {
		errRes(w, r, http.StatusInternalServerError, "Templating error", err)
		return
	}
}

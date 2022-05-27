package routes

import (
	"context"
	"doubleboiler/models"
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
	Context context.Context
}

func thingCreationFormHandler(w http.ResponseWriter, r *http.Request) {
	targetOrg := activeOrgFromContext(r.Context())
	if targetOrg.ID == "" {
		redirToDefaultOrg(w, r)
		return
	}

	if err := Tmpl.ExecuteTemplate(w, "create-thing.html", thingCreationPageData{
		Context: r.Context(),
	}); err != nil {
		errRes(w, r, http.StatusInternalServerError, "Templating error", err)
		return
	}
}

func thingCreateOrUpdateHandler(w http.ResponseWriter, r *http.Request) {
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
		errRes(w, r, http.StatusForbidden, "You cannot create things for that organisation", nil)
		return
	}

	var thing models.Thing

	// Thing already exists. This is an update.
	if r.FormValue("id") != "" {
		if err := thing.FindByID(r.Context(), r.FormValue("id")); err != nil {
			errRes(w, r, http.StatusInternalServerError, "Error looking up thing", err)
			return
		}

		if thing.Revision != r.FormValue("revision") {
			errRes(w, r, http.StatusBadRequest, models.ErrWrongRev.Message, nil)
			return
		}

		thing.Name = r.FormValue("name")
		thing.Description = r.FormValue("description")
	} else {
		// Thing doesn't exist. Let's create it.

		thing.New(
			r.FormValue("name"),
			r.FormValue("description"),
			r.FormValue("organisationID"),
		)
	}

	if err := thing.Save(r.Context()); err != nil {
		errRes(w, r, http.StatusInternalServerError, "A database error has occurred", err)
		return
	}

	http.Redirect(w, r, "/things/"+thing.ID, 302)
}

type thingPageData struct {
	Thing   models.Thing
	Context context.Context
}

func thingHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	thing := models.Thing{}
	if err := thing.FindByID(r.Context(), vars["id"]); err != nil {
		errRes(w, r, http.StatusInternalServerError, "A database error has occurred", err)
		return
	}

	org := orgFromContext(r.Context(), thing.OrganisationID)

	if !can(r.Context(), org, "admin") {
		errRes(w, r, http.StatusForbidden, "You cannot view things for that organisation", nil)
		return
	}

	if err := Tmpl.ExecuteTemplate(w, "thing.html", thingPageData{
		Context: r.Context(),
		Thing:   thing,
	}); err != nil {
		errRes(w, r, http.StatusInternalServerError, "Templating error", err)
		return
	}
}

type thingsPageData struct {
	Things  models.Things
	Context context.Context
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

	things := models.Things{}

	query := models.ByOrg{ID: targetOrg.ID}
	query.DefaultPageSize = 50
	query.Paginate(r.Form)

	if err := things.FindAll(r.Context(), query); err != nil {
		errRes(w, r, http.StatusInternalServerError, "error fetching things", err)
		return
	}

	if err := Tmpl.ExecuteTemplate(w, "things.html", thingsPageData{
		Things:  things,
		Context: r.Context(),
	}); err != nil {
		errRes(w, r, http.StatusInternalServerError, "Templating error", err)
		return
	}
}

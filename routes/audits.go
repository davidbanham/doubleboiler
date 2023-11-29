package routes

import (
	"doubleboiler/models"
	"net/http"

	"github.com/gorilla/mux"
)

func init() {
	r.Path("/audits").
		Methods("GET").
		HandlerFunc(auditsHandler)

	r.Path("/audits/{id}").
		Methods("GET").
		HandlerFunc(auditsHandler)
}

type auditsPageData struct {
	basePageData
	Audits models.Audits
}

func auditsHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	targetOrg := activeOrgFromContext(r.Context())

	if targetOrg.ID == "" {
		redirToDefaultOrg(w, r)
		return
	}

	if !can(r.Context(), targetOrg, "admin") {
		errRes(w, r, http.StatusForbidden, "You cannot list audits for that organisation", nil)
		return
	}

	audits := models.Audits{}

	var criteria models.Criteria

	if vars["id"] != "" {
		criteria.Query = &models.ByEntityID{EntityID: vars["id"]}
	} else {
		criteria.Query = &models.ByOrg{ID: targetOrg.ID}
	}

	criteria.Pagination.DefaultPageSize = 50
	criteria.Pagination.Paginate(r.Form)

	if err := audits.FindAll(r.Context(), criteria); err != nil {
		errRes(w, r, http.StatusInternalServerError, "error fetching audits", err)
		return
	}

	for _, audit := range audits.Data {
		if audit.OrganisationID != targetOrg.ID {
			errRes(w, r, http.StatusForbidden, "You cannot list audits for that organisation", nil)
			return
		}
	}

	if err := Tmpl.ExecuteTemplate(w, "audits.html", auditsPageData{
		Audits: audits,
		basePageData: basePageData{
			PageTitle: "DoubleBoiler - Audits",
			Context:   r.Context(),
		},
	}); err != nil {
		errRes(w, r, http.StatusInternalServerError, "Templating error", err)
		return
	}
}

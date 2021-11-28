package routes

import (
	"context"
	"doubleboiler/models"
	"net/http"
)

func init() {
	r.Path("/search").
		Methods("GET").
		HandlerFunc(searchHandler)
}

type searchResultPageData struct {
	Context context.Context
	Phrase  string
	Results models.SearchResults
}

func searchHandler(w http.ResponseWriter, r *http.Request) {
	targetOrg := activeOrgFromContext(r.Context())

	if targetOrg.ID == "" {
		redirToDefaultOrg(w, r)
		return
	}

	if !can(r.Context(), targetOrg, "admin") {
		errRes(w, r, http.StatusForbidden, "You cannot list things for that organisation", nil)
		return
	}

	results := models.SearchResults{}

	query := models.ByPhrase{
		OrgID:  targetOrg.ID,
		Phrase: r.FormValue("search_field"),
		User:   userFromContext(r.Context()),
	}
	query.DefaultPageSize = 50
	query.Paginate(r.Form)

	if err := results.FindAll(r.Context(), query); err != nil {
		errRes(w, r, 500, "error fetching results", err)
		return
	}

	if err := Tmpl.ExecuteTemplate(w, "searchresults.html", searchResultPageData{
		Results: results,
		Phrase:  r.FormValue("search_field"),
		Context: r.Context(),
	}); err != nil {
		errRes(w, r, http.StatusInternalServerError, "Templating error", err)
		return
	}
}

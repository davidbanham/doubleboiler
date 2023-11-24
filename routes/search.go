package routes

import (
	"doubleboiler/models"
	"net/http"
)

var searchableWhitelist map[string]models.Searchable

func init() {
	r.Path("/search").
		Methods("GET").
		HandlerFunc(searchHandler)
}

type searchResultPageData struct {
	basePageData
	Phrase           string
	EntitiesIncluded []string
	Results          models.SearchResults
}

func searchHandler(w http.ResponseWriter, r *http.Request) {
	targetOrg := activeOrgFromContext(r.Context())

	if targetOrg.ID == "" {
		redirToDefaultOrg(w, r)
		return
	}

	if !can(r.Context(), targetOrg, "admin") {
		errRes(w, r, http.StatusForbidden, "You cannot search for that organisation", nil)
		return
	}

	results := models.SearchResults{}

	query := models.ByPhrase{
		OrganisationID: targetOrg.ID,
		Phrase:         r.FormValue("search_field"),
	}

	entities := []string{}
	targets := models.SearchTargets
	if len(r.Form["entity-filter"]) != 0 {
		targets = targets.FilterByTableNames(r.Form["entity-filter"])
	}
	for _, target := range targets {
		entities = append(entities, target.Tablename)
	}

	criteria := models.SearchCriteria{
		Entities: entities,
		Query:    query,
		Pagination: models.Pagination{
			DefaultPageSize: 50,
		},
	}

	criteria.Pagination.Paginate(r.Form)
	roles := orgUserFromContext(r.Context(), targetOrg).Roles

	if err := results.FindAll(r.Context(), roles, criteria, models.SearchTargets); err != nil {
		errRes(w, r, 500, "error fetching results", err)
		return
	}

	if err := Tmpl.ExecuteTemplate(w, "searchresults.html", searchResultPageData{
		Results: results,
		basePageData: basePageData{
			PageTitle: "DoubleBoiler - Search Results",
			Context:   r.Context(),
		},
	}); err != nil {
		errRes(w, r, http.StatusInternalServerError, "Templating error", err)
		return
	}
}

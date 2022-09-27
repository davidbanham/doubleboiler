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

	searchableWhitelist = map[string]models.Searchable{}
	for _, searchable := range models.Searchables {
		searchableWhitelist[searchable.Label] = searchable
	}
}

type searchResultPageData struct {
	basePageData
	Phrase          string
	EntityFilterMap map[string]bool
	Results         models.SearchResults
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
		OrgID:        targetOrg.ID,
		Phrase:       r.FormValue("search_field"),
		User:         userFromContext(r.Context()),
		Roles:        orgUserFromContext(r.Context(), targetOrg).Roles,
		EntityFilter: map[string]bool{},
	}
	for _, label := range r.Form["entity-filter"] {
		if label != "" && searchableWhitelist[label].Label == label {
			query.EntityFilter[label] = true
		}
	}

	query.DefaultPageSize = 50
	query.Paginate(r.Form)

	if err := results.FindAll(r.Context(), query); err != nil {
		errRes(w, r, 500, "error fetching results", err)
		return
	}

	if err := Tmpl.ExecuteTemplate(w, "searchresults.html", searchResultPageData{
		Results:         results,
		Phrase:          r.FormValue("search_field"),
		EntityFilterMap: query.EntityFilter,
		basePageData: basePageData{
			PageTitle: "DoubleBoiler - Search Results",
			Context:   r.Context(),
		},
	}); err != nil {
		errRes(w, r, http.StatusInternalServerError, "Templating error", err)
		return
	}
}

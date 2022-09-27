package routes

import (
	"doubleboiler/changelog"
	"net/http"
)

func init() {
	r.Path("/changelog").
		Methods("GET").
		HandlerFunc(serveChangelog)
}

type changelogPageData struct {
	basePageData
	Changes []changelog.Change
}

func serveChangelog(w http.ResponseWriter, r *http.Request) {
	changes := changelog.Changes

	err := Tmpl.ExecuteTemplate(w, "changelog.html", changelogPageData{
		basePageData: basePageData{
			PageTitle: "DoubleBoiler - Changelog",
			Context:   r.Context(),
		},
		Changes: changes,
	})
	if err != nil {
		errRes(w, r, 500, "Problem with template", err)
		return
	}
}

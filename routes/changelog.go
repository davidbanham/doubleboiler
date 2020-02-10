package routes

import (
	"context"
	"doubleboiler/changelog"
	"net/http"
)

func init() {
	r.Path("/changelog").
		Methods("GET").
		HandlerFunc(serveChangelog)
}

type changelogPageData struct {
	Context context.Context
	Changes []changelog.Change
}

func serveChangelog(w http.ResponseWriter, r *http.Request) {
	changes := changelog.Changes

	err := Tmpl.ExecuteTemplate(w, "changelog.html", changelogPageData{
		Context: r.Context(),
		Changes: changes,
	})
	if err != nil {
		errRes(w, r, 500, "Problem with template", err)
		return
	}
}

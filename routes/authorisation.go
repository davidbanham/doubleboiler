package routes

import (
	"context"
	"doubleboiler/models"
	"net/http"
	"strings"
)

func formParsingMiddleware(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		h.ServeHTTP(w, r)
	})
}

func authFreeMiddleware(h http.Handler) http.Handler {
	var unAuthedPaths = []string{
		"",
		"reset-password",
		"verify",
		"login",
		"signup",
		"signup-successful",
		"css",
		"fonts",
		"img",
		"images",
		"js",
		"prospects",
		"health",
		"contact",
		"sales-enquiry",
		"trial-mode-upgrade",
		"webhooks",
		"features",
		"pricing",
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		noAuth := false
		for _, path := range unAuthedPaths {
			slice := strings.Split(r.URL.Path, "/")
			if len(slice) == 0 {
				noAuth = true
			} else {
				if slice[1] == path {
					noAuth = true
				}

				// Root favicon nonsense
				if strings.Contains(slice[1], "png") {
					noAuth = true
				}
				if strings.Contains(slice[1], "xml") {
					noAuth = true
				}
				if strings.Contains(slice[1], "ico") {
					noAuth = true
				}
				if strings.Contains(slice[1], "svg") {
					noAuth = true
				}
				if strings.Contains(slice[1], "json") {
					noAuth = true
				}
				if strings.Contains(slice[1], "xml") {
					noAuth = true
				}
				if strings.Contains(slice[1], "pdf") {
					noAuth = true
				}
			}
		}

		con := context.WithValue(r.Context(), "authFree", noAuth)
		h.ServeHTTP(w, r.WithContext(con))
	})
}

func isAuthFree(ctx context.Context) bool {
	authFree, ok := ctx.Value("authFree").(bool)
	return authFree && ok
}

func can(ctx context.Context, target models.Organisation, role string) bool {
	if isAppAdmin(ctx) {
		return true
	}

	unconv := ctx.Value("organisation_users")
	if unconv == nil {
		return false
	}
	orgUsers := unconv.(models.OrganisationUsers)

	for _, ou := range orgUsers.Data {
		if ou.OrganisationID == target.ID {
			for _, r := range ou.Roles {
				if r.Can(role) {
					return true
				}
			}
		}
	}

	return false
}

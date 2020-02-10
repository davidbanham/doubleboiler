package routes

import (
	"context"
	"doubleboiler/models"
	"net/http"
)

func orgMiddleware(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Context().Value("authFree").(bool) {
			h.ServeHTTP(w, r)
			return
		}

		organisations := models.Organisations{}
		organisationUsers := models.OrganisationUsers{}

		unconv := r.Context().Value("user")

		if unconv != nil {
			user := unconv.(models.User)

			query := models.OrganisationsContainingUser
			if user.Admin {
				query = models.All
			}
			if err := organisations.FindAll(r.Context(), query, user.ID); err != nil {
				errRes(w, r, 500, "error looking up organisations", err)
				return
			}

			if err := organisationUsers.FindAll(r.Context(), models.ByUser, user.ID); err != nil {
				errRes(w, r, 500, "error looking up organisation users", err)
				return
			}
		}

		con := context.WithValue(r.Context(), "organisations", organisations)
		con = context.WithValue(con, "organisation_users", organisationUsers)

		setOrgCookie := true

		if unconv != nil {
			targetOrg := r.URL.Query().Get("organisationid")
			if targetOrg == "" {
				cookieOrg := orgFromCookie(r)
				if cookieOrg != "" {
					setOrgCookie = false
					targetOrg = cookieOrg
				} else {
					if len(organisations) > 0 {
						targetOrg = organisations[0].ID
					}
				}
			}
			con = context.WithValue(con, "target_org", targetOrg)

			if setOrgCookie {
				encoded, err := secureCookie.Encode("doubleboiler-targetorg", map[string]string{
					"TargetOrg": targetOrg,
				})
				if err != nil {
					errRes(w, r, 500, "Error encoding cookie", nil)
					return
				}
				cookie := http.Cookie{
					Path:     "/",
					Name:     "doubleboiler-targetorg",
					Value:    encoded,
					Secure:   true,
					HttpOnly: true,
				}
				http.SetCookie(w, &cookie)
			}

			qs := r.URL.Query()
			qs.Set("organisationid", targetOrg)
			r.URL.RawQuery = qs.Encode()
		}

		h.ServeHTTP(w, r.WithContext(con))
	})
}

func orgFromCookie(r *http.Request) string {
	c, err := r.Cookie("doubleboiler-targetorg")
	if err != nil {
		return ""
	}

	cookieValue := make(map[string]string)
	if err := secureCookie.Decode("doubleboiler-targetorg", c.Value, &cookieValue); err != nil {
		return ""
	}

	return cookieValue["TargetOrg"]
}

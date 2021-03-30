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

			var query models.Query
			query = models.OrganisationsContainingUser{}
			if user.Admin {
				query = models.All{}
			}
			if err := organisations.FindAll(r.Context(), query, user.ID); err != nil {
				errRes(w, r, 500, "error looking up organisations", err)
				return
			}

			if err := organisationUsers.FindAll(r.Context(), models.ByUser{}, user.ID); err != nil {
				errRes(w, r, 500, "error looking up organisation users", err)
				return
			}
		}

		con := context.WithValue(r.Context(), "organisations", organisations)
		con = context.WithValue(con, "organisation_users", organisationUsers)

		if unconv != nil {
			qsOrg := r.URL.Query().Get("organisationid")
			if qsOrg != "" {
				encoded, err := secureCookie.Encode("doubleboiler-targetorg", map[string]string{
					"TargetOrg": qsOrg,
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

				q := r.URL.Query()
				q.Del("organisationid")
				r.URL.RawQuery = q.Encode()
				http.Redirect(w, r, r.URL.String(), http.StatusFound)
				return
			}

			targetOrg := orgFromCookie(r)

			if targetOrg == "" {
				if len(organisations.Data) > 0 {
					targetOrg = organisations.Data[0].ID
				}
			}

			con = context.WithValue(con, "target_org", targetOrg)

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

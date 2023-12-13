package routes

import (
	"bytes"
	"context"
	"database/sql"
	"doubleboiler/config"
	"doubleboiler/flashes"
	"doubleboiler/logger"
	"doubleboiler/models"
	"doubleboiler/util"
	"encoding/base64"
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"runtime/debug"
	"strconv"
	"strings"
	"time"
)

var templateFuncMap = template.FuncMap{
	"humanDate": func(t time.Time) string {
		return t.Format("02 Jan 2006")
	},
	"dateonly": func(in time.Time) string {
		return in.Format("2006-01-02")
	},
	"breakLines": func(in string) []string {
		return strings.Split(in, "\n")
	},
	"contains": func(str []string, target string) bool {
		for _, s := range str {
			if s == target {
				return true
			}
		}
		return false
	},
	"firstFiveChars":       util.FirstFiveChars,
	"loggedIn":             isLoggedIn,
	"user":                 userFromContext,
	"orgsFromContext":      orgsFromContext,
	"flashes":              flashesFromContext,
	"activeOrgFromContext": activeOrgFromContext,
	"can": func(ctx context.Context, role string) bool {
		org := activeOrgFromContext(ctx)
		return can(ctx, org, role)
	},
	"csrf": func(ctx context.Context) string {
		unconv := ctx.Value("user")
		if unconv == nil {
			return ""
		}

		user := unconv.(models.User)
		return util.CalcToken(config.SECRET, 0, user.ID).String()
	},
	"isLocal": func() bool { return config.LOCAL },
	"logoLink": func(ctx context.Context) string {
		if !isLoggedIn(ctx) {
			return "/"
		}
		unconv := ctx.Value("url")
		if unconv == nil {
			return "/"
		}
		url := unconv.(*url.URL)
		if strings.Contains(url.Path, "/dashboard") {
			return "/"
		}
		return "/dashboard"
	},
	"subComponent": func(name string, data interface{}) (template.HTML, error) {
		return Tmpl.Component(name, data)
	},
	"searchableEntities": func(ctx context.Context, query models.SearchQuery) models.Searchables {
		org := activeOrgFromContext(ctx)
		roles := orgUserFromContext(ctx, org).Roles

		return models.SearchTargets.FilterByRole(roles, query)
	},
	"base64": func(in bytes.Buffer) string {
		return base64.StdEncoding.EncodeToString(in.Bytes())
	},
}

func checkFormInput(required []string, form url.Values, w http.ResponseWriter, r *http.Request) bool {
	for _, val := range required {
		if len(form[val]) < 1 {
			errRes(w, r, 400, "Invalid "+val, nil)
			return false
		}
		if form[val][0] == "" {
			errRes(w, r, 400, "Invalid "+val, nil)
			return false
		}
	}
	return true
}

type errorPageData struct {
	basePageData
	Message string
	Context context.Context
}

func errRes(w http.ResponseWriter, r *http.Request, code int, message string, passedErr error) {
	sendErr := func() {
		if passedErr != nil && passedErr.Error() == "http2: stream closed" {
			return
		}

		if passedErr == nil {
			config.ReportError(errors.New(strconv.Itoa(code) + " " + message))
		} else {
			config.ReportError(passedErr)
		}

		if clientSafe, addendum := isClientSafe(passedErr); clientSafe {
			message += " " + addendum
		}

		logger.Log(r.Context(), logger.Warning, fmt.Sprintf("Sending Error Response: %+v, %+v, %+v, %+v", code, message, r.URL.String(), passedErr))
		if code == 500 {
			logger.Log(r.Context(), logger.Error, passedErr)
			logger.Log(r.Context(), logger.Debug, string(debug.Stack()))
		}

		w.WriteHeader(code)
		if r.Header.Get("Accept") == "application/json" {
			w.Write([]byte(fmt.Sprintf(`{"error": "%s"}`, message)))
			return
		} else {
			if err := Tmpl.ExecuteTemplate(w, "error.html", errorPageData{
				Message: message,
				Context: r.Context(),
			}); err != nil {
				config.ReportError(err)
				w.Write([]byte("Error rendering the error template. Oh dear."))
				return
			}
		}
	}

	tx := r.Context().Value("tx")
	switch v := tx.(type) {
	case *sql.Tx:
		rollbackErr := v.Rollback()
		if rollbackErr != nil {
			logger.Log(r.Context(), logger.Error, fmt.Sprintf("Error rolling back tx: %+v", rollbackErr))
		}
	default:
		//fmt.Printf("DEBUG no transaction on error\n")
	}
	sendErr()
}

func redirToDefaultOrg(w http.ResponseWriter, r *http.Request) {
	orgs := orgsFromContext(r.Context())
	if len(orgs.Data) < 1 {
		http.Redirect(w, r, "/organisations/create", http.StatusFound)
		return
	} else {
		query := r.URL.Query()
		query.Set("organisationid", orgs.Data[0].ID)
		r.URL.RawQuery = query.Encode()
	}

	http.Redirect(w, r, r.URL.String(), http.StatusFound)
}

func isClientSafe(err error) (bool, string) {
	type clientSafe interface {
		ClientSafeMessage() string
	}
	cse, ok := err.(clientSafe)
	if ok {
		return ok, cse.ClientSafeMessage()
	} else {
		return false, ""
	}
}

func userFromContext(ctx context.Context) models.User {
	if !isLoggedIn(ctx) {
		return models.User{}
	}
	return ctx.Value("user").(models.User)
}

func totpVerifiedFromContext(ctx context.Context) bool {
	unconv := ctx.Value("totp-verified")
	if unconv == nil {
		return false
	}
	return unconv.(bool)
}

func orgUserFromContext(ctx context.Context, org models.Organisation) models.OrganisationUser {
	if v, ok := ctx.Value("organisation_users").(models.OrganisationUsers); ok {
		for _, ou := range v.Data {
			if ou.OrganisationID == org.ID {
				return ou
			}
		}
	}
	return models.OrganisationUser{}
}

func flashesFromContext(ctx context.Context) flashes.Flashes {
	if ctx == nil {
		return flashes.Flashes{}
	}
	switch user := ctx.Value("user").(type) {
	case models.User:
		for _, flash := range user.Flashes {
			if !flash.Sticky {
				user.DeleteFlash(ctx, flash)
			}
		}
		return user.Flashes
	}
	return flashes.Flashes{}
}

var nextFlow = util.NextFlow

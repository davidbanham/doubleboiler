package routes

import (
	"doubleboiler/models"
	"doubleboiler/util"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
)

func init() {
	r.Path("/communications").
		Methods("GET").
		HandlerFunc(communicationsHandler)

	r.Path("/communications/{id}").
		Methods("GET").
		HandlerFunc(communicationHandler)
}

type communicationsPageData struct {
	basePageData
	Communications models.Communications
	ActiveOrg      models.Organisation
	Users          models.Users
}

func communicationsHandler(w http.ResponseWriter, r *http.Request) {
	targetOrg := activeOrgFromContext(r.Context())

	if targetOrg.ID == "" {
		redirToDefaultOrg(w, r)
		return
	}

	if !can(r.Context(), targetOrg, "admin") {
		errRes(w, r, http.StatusForbidden, "You cannot view communications for that organisation", nil)
		return
	}

	customFilters := models.Filters{}

	if r.FormValue("org-user-id") != "" {
		orgUser := models.OrganisationUser{}
		if err := orgUser.FindByID(r.Context(), r.FormValue("org-user-id")); err != nil {
			errRes(w, r, http.StatusInternalServerError, "Error looking up org user", err)
			return
		}

		cf := models.Custom{
			Key:         "user_id",
			Values:      []string{orgUser.UserID},
			CustomID:    fmt.Sprintf("orguser-is-%s", orgUser.ID),
			CustomLabel: orgUser.FullName(),
		}
		customFilters = append(customFilters, cf)
		r.Form.Add("custom-filter", cf.CustomID)
	}

	communications := models.Communications{}

	query := models.ByOrg{ID: targetOrg.ID}
	query.DefaultPageSize = 50
	query.Paginate(r.Form)
	if err := communications.FindAll(r.Context(), query); err != nil {
		errRes(w, r, 500, "error fetching communications", err)
		return
	}

	users, err := communications.Users(r.Context())
	if err != nil {
		errRes(w, r, 500, "error fetching users", err)
		return
	}

	if err := Tmpl.ExecuteTemplate(w, "communications.html", communicationsPageData{
		Communications: communications,
		basePageData: basePageData{
			PageTitle: "DoubleBoiler - Communications",
			Context:   r.Context(),
		},
		ActiveOrg: targetOrg,
		Users:     users,
	}); err != nil {
		errRes(w, r, http.StatusInternalServerError, "Templating error", err)
		return
	}
}

type communicationPageData struct {
	basePageData
	Communication    models.Communication
	ActiveOrg        models.Organisation
	OrganisationUser models.OrganisationUser
}

func communicationHandler(w http.ResponseWriter, r *http.Request) {
	targetOrg := activeOrgFromContext(r.Context())

	if targetOrg.ID == "" {
		redirToDefaultOrg(w, r)
		return
	}

	if !can(r.Context(), targetOrg, "admin") {
		errRes(w, r, http.StatusForbidden, "You cannot view communications for that organisation", nil)
		return
	}

	vars := mux.Vars(r)

	communication := models.Communication{}
	if err := communication.FindByID(r.Context(), vars["id"]); err != nil {
		errRes(w, r, 500, "error fetching communication", err)
		return
	}

	orgUser := models.OrganisationUser{}

	if communication.UserID.Valid {
		orgUsers := models.OrganisationUsers{}
		if err := orgUsers.FindAll(r.Context(), models.ByOrg{ID: targetOrg.ID}); err != nil {
			errRes(w, r, 500, "error fetching org users", err)
			return
		}
		for _, ou := range orgUsers.Data {
			if ou.UserID == communication.UserID.String {
				orgUser = ou
			}
		}
	}

	if err := Tmpl.ExecuteTemplate(w, "communication.html", communicationPageData{
		Communication: communication,
		basePageData: basePageData{
			PageTitle: "DoubleBoiler - Communication " + util.FirstFiveChars(communication.ID),
			Context:   r.Context(),
		},
		ActiveOrg:        targetOrg,
		OrganisationUser: orgUser,
	}); err != nil {
		errRes(w, r, http.StatusInternalServerError, "Templating error", err)
		return
	}
}

package routes

import (
	"context"
	"doubleboiler/models"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
)

func TestOrganisationUserCreateHandler(t *testing.T) {
	t.Parallel()
	ctx := getCtx(t)

	org := models.Organisation{}
	org.New(
		bandname(),
		"Australia",
		[]models.OrganisationUser{},
		"aud",
	)
	assert.Nil(t, org.Save(ctx))

	form := url.Values{
		"organisationID": {org.ID},
		"email":          {bandEmail()},
	}
	req := &http.Request{
		Method: "POST",
		URL:    &url.URL{Path: "/organisation-users"},
		Form:   form,
	}

	ctx = contextifyOrgAdmin(ctx, org)

	req = req.WithContext(ctx)

	user, _ := userFixture(ctx, t)
	req = contextify(user, req)

	rr := httptest.NewRecorder()
	organisationUserCreateHandler(rr, req)

	assert.Equal(t, http.StatusFound, rr.Code)

	closeTx(t, ctx)
}

func organisationUserFixture(ctx context.Context, t *testing.T) (i models.OrganisationUser) {
	org := models.Organisation{}
	org.New(
		bandname(),
		"Australia",
		[]models.OrganisationUser{},
		"aud",
	)
	assert.Nil(t, org.Save(ctx))

	user := models.User{}
	user.New(
		bandEmail(),
		bandname(),
	)
	assert.Nil(t, user.Save(ctx))

	i.New(
		user.ID,
		org.ID,
		models.Roles{"admin": true},
	)
	i.Save(ctx)
	return
}

func TestOrganisationUserDeletion(t *testing.T) {
	t.Parallel()
	ctx := getCtx(t)

	fix := organisationUserFixture(ctx, t)

	org := models.Organisation{}
	assert.Nil(t, org.FindByID(ctx, fix.OrganisationID))

	req, err := http.NewRequest("POST", "/organisations/remove-user/"+fix.ID, nil)
	assert.Nil(t, err)

	ctx = contextifyOrgAdmin(ctx, org)
	req = req.WithContext(ctx)

	u := models.User{}
	u.New(
		bandEmail(),
		bandname(),
	)

	req = req.WithContext(ctx)

	req = contextify(u, req)

	r := mux.NewRouter()

	r.HandleFunc("/organisations/remove-user/{id}", organisationUserDeletionHandler).Methods("POST")

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	assert.Equal(t, rr.Code, http.StatusFound)
	loc, err := rr.Result().Location()
	assert.Nil(t, err)

	updatedOrg := models.Organisation{}
	assert.Nil(t, updatedOrg.FindByID(ctx, org.ID))

	assert.Equal(t, "/organisations/"+fix.OrganisationID, loc.Path)

	r.HandleFunc("/organisations/{id}", organisationHandler).Methods("GET")
	req2, err := http.NewRequest("GET", loc.String(), nil)
	assert.Nil(t, err)

	ctx = context.WithValue(ctx, "user", u)
	ctx = contextifyOrgAdmin(ctx, updatedOrg)

	req2 = req2.WithContext(ctx)

	r2 := httptest.NewRecorder()
	r.ServeHTTP(r2, req2)

	assert.Equal(t, r2.Code, http.StatusOK)
	assert.NotContains(t, r2.Body.String(), fix.ID)

	closeTx(t, ctx)
}

func TestOrganisationUserCreateHandlerAddExistingUser(t *testing.T) {
	t.Parallel()
	ctx := getCtx(t)

	user := models.User{}
	user.New(bandEmail(), bandname())
	assert.Nil(t, user.Save(ctx))

	org := models.Organisation{}
	org.New(bandname(), "Australia", []models.OrganisationUser{}, "aud")
	assert.Nil(t, org.Save(ctx))

	form := url.Values{
		"email":          {user.Email},
		"organisationID": {org.ID},
	}
	req := &http.Request{
		Method: "POST",
		URL:    &url.URL{Path: "/organisation-users"},
		Form:   form,
	}

	ctx = contextifyOrgAdmin(ctx, org)

	req = req.WithContext(ctx)

	req = contextify(user, req)

	rr := httptest.NewRecorder()
	organisationUserCreateHandler(rr, req)

	assert.Equal(t, http.StatusFound, rr.Code)

	loc, err := rr.Result().Location()
	assert.Nil(t, err)

	req, err = http.NewRequest("GET", loc.String(), nil)
	assert.Nil(t, err)

	req = req.WithContext(ctx)
	req = contextify(user, req)

	r := mux.NewRouter()

	r.HandleFunc("/organisations/{id}", organisationHandler).Methods("GET")

	r2 := httptest.NewRecorder()
	r.ServeHTTP(r2, req)

	assert.Equal(t, http.StatusOK, r2.Code)
	assert.Contains(t, r2.Body.String(), user.Email, "User email not found")

	closeTx(t, ctx)
}

func TestOrganisationUserCreateHandlerAddNewUserByEmail(t *testing.T) {
	t.Parallel()
	ctx := getCtx(t)

	email := bandEmail()

	org := models.Organisation{}
	org.New(
		bandname(),
		"Australia",
		[]models.OrganisationUser{},
		"aud",
	)
	assert.Nil(t, org.Save(ctx))

	form := url.Values{
		"organisationID": {org.ID},
		"email":          {email},
	}
	req := &http.Request{
		Method: "POST",
		URL:    &url.URL{Path: "/organisation-users"},
		Form:   form,
	}

	ctx = contextifyOrgAdmin(ctx, org)

	req = req.WithContext(ctx)

	user, _ := userFixture(ctx, t)
	req = contextify(user, req)

	rr := httptest.NewRecorder()
	organisationUserCreateHandler(rr, req)

	assert.Equal(t, http.StatusFound, rr.Code)

	loc, err := rr.Result().Location()
	assert.Nil(t, err)

	updatedOrg := models.Organisation{}
	assert.Nil(t, updatedOrg.FindByID(ctx, org.ID))
	ctx = contextifyOrgAdmin(ctx, updatedOrg)

	req, err = http.NewRequest("GET", loc.String(), nil)
	assert.Nil(t, err)

	req = req.WithContext(ctx)

	req = contextify(user, req)

	r := mux.NewRouter()

	r.HandleFunc("/organisations/{id}", organisationHandler).Methods("GET")

	r2 := httptest.NewRecorder()
	r.ServeHTTP(r2, req)

	assert.Equal(t, http.StatusOK, r2.Code)
	assert.Contains(t, r2.Body.String(), email, "User email not found")

	createdUser := models.User{}
	err = createdUser.FindByColumn(ctx, "email", email)
	assert.Nil(t, err)

	assert.False(t, createdUser.Verified)

	closeTx(t, ctx)
}

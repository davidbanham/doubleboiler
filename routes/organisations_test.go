package routes

import (
	"context"
	"doubleboiler/config"
	"doubleboiler/models"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
)

func TestOrganisationCreateOrUpdateHandler(t *testing.T) {
	t.Parallel()
	ctx := getCtx(t)

	user := models.User{}
	user.New(bandEmail(), bandname())
	user.Save(ctx)

	form := url.Values{
		"name":         {bandname()},
		"users.userid": {user.ID},
	}

	req := &http.Request{
		Method: "POST",
		URL:    &url.URL{Path: "/organisations"},
		Form:   form,
	}

	u := models.User{}
	u.New(bandEmail(), bandname())
	u.Admin = true
	assert.Nil(t, u.Save(ctx))

	ctx = context.WithValue(ctx, "user", u)

	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	organisationCreateOrUpdateHandler(rr, req)

	assert.Equal(t, http.StatusFound, rr.Code)

	closeTx(t, ctx)
}

func TestOrganisationCreateOrUpdateHandlerUpdate(t *testing.T) {
	t.Parallel()
	ctx := getCtx(t)

	org := models.Organisation{}
	org.New(
		bandname(),
		"Australia",
	)
	org.Save(ctx)

	form := url.Values{
		"name":     {org.Name},
		"country":  {org.Country},
		"id":       {org.ID},
		"revision": {org.Revision},
	}
	req := &http.Request{
		Method: "POST",
		URL:    &url.URL{Path: "/organisations/" + org.ID},
		Form:   form,
	}

	u := models.User{}
	u.New(bandEmail(), bandname())
	assert.Nil(t, u.Save(ctx))

	ctx = context.WithValue(ctx, "user", u)

	req = req.WithContext(ctx)

	r := mux.NewRouter()

	r.HandleFunc("/organisations/{id}", organisationCreateOrUpdateHandler).Methods("POST")

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusFound, rr.Code)

	closeTx(t, ctx)
}

func TestOrganisationHandler(t *testing.T) {
	t.Parallel()
	ctx := getCtx(t)

	fixture := organisationFixture(ctx, t)

	req, err := http.NewRequest("GET", "/organisations/"+fixture.ID, nil)

	assert.Nil(t, err)

	u := models.User{}
	u.New(
		bandEmail(),
		bandname(),
	)

	ctx = context.WithValue(ctx, "user", u)
	ctx = contextifyOrgAdmin(ctx, fixture)

	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()

	r := mux.NewRouter()

	r.HandleFunc("/organisations/{id}", organisationHandler).Methods("GET")

	r.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), fixture.Name, "Name not found")

	closeTx(t, ctx)
}

func TestOrganisationsHandler(t *testing.T) {
	t.Parallel()
	ctx := getCtx(t)

	fixture := organisationFixture(ctx, t)

	req, err := http.NewRequest("GET", "/organisations", nil)
	assert.Nil(t, err)

	ctx = contextifyOrgAdmin(ctx, fixture)

	user, _ := userFixture(ctx, t)
	ctx = context.WithValue(ctx, "user", user)

	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()

	r := mux.NewRouter()

	r.HandleFunc("/organisations", organisationsHandler).Methods("GET")

	r.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), fixture.Name, "Name not found")

	closeTx(t, ctx)
}

func TestOrganisationsInvalidInputChecking(t *testing.T) {
	t.Parallel()

	ctx := getCtx(t)
	form := url.Values{
		"lol": {"wut"},
	}
	req := &http.Request{
		Method: "POST",
		URL:    &url.URL{Path: "/organisations"},
		Form:   form,
	}
	req = req.WithContext(ctx)

	u := models.User{}
	u.New(bandEmail(), bandname())

	ctx = context.WithValue(ctx, "user", u)

	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	organisationCreateOrUpdateHandler(rr, req)

	assert.Equal(t, rr.Code, http.StatusBadRequest)
	assert.Contains(t, rr.Body.String(), "Invalid name")

	closeTx(t, ctx)
}

func TestCopySampleOrgData(t *testing.T) {
	t.Parallel()
	ctx := getCtx(t)

	org := organisationFixture(ctx, t)
	config.SAMPLEORG_ID = org.ID

	newOrg := organisationFixture(ctx, t)

	assert.Nil(t, copySampleOrgData(ctx, newOrg))
}

func organisationFixture(ctx context.Context, t *testing.T) (i models.Organisation) {
	i.New(
		bandname(),
		"Australia",
	)
	err := i.Save(ctx)
	assert.Nil(t, err)
	return
}

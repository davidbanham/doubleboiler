package routes

import (
	"context"
	"doubleboiler/models"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
)

func TestSomeThingCreateHandler(t *testing.T) {
	t.Parallel()

	ctx := getCtx(t)
	org := organisationFixture(ctx, t)
	ctx = contextifyOrgAdmin(ctx, org)

	form := url.Values{
		"name":           {bandname()},
		"description":    {bandname()},
		"organisationID": {org.ID},
	}
	req := &http.Request{
		Method: "POST",
		URL:    &url.URL{Path: "/some-things"},
		Form:   form,
	}

	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	someThingCreateOrUpdateHandler(rr, req)

	assert.Equal(t, http.StatusFound, rr.Code)

	closeTx(t, ctx)
}

func TestSomeThingHandler(t *testing.T) {
	t.Parallel()

	ctx := getCtx(t)
	org := organisationFixture(ctx, t)
	ctx = contextifyOrgAdmin(ctx, org)

	fixture := someThingFixture(ctx, t, org)

	req, err := http.NewRequest("GET", "/some-things/"+fixture.ID, nil)
	req = req.WithContext(ctx)

	assert.Nil(t, err)

	rr := httptest.NewRecorder()

	r := mux.NewRouter()

	r.HandleFunc("/some-things/{id}", someThingHandler).Methods("GET")

	r.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), fixture.Name, "Name not found")

	closeTx(t, ctx)
}

func TestSomeThingDeleteHandler(t *testing.T) {
	t.Parallel()

	ctx := getCtx(t)
	org := organisationFixture(ctx, t)
	ctx = contextifyOrgAdmin(ctx, org)

	fixture := someThingFixture(ctx, t, org)

	req, err := http.NewRequest("DELETE", "/some-things/"+fixture.ID, nil)
	req = req.WithContext(ctx)

	assert.Nil(t, err)

	rr := httptest.NewRecorder()

	r := mux.NewRouter()

	r.HandleFunc("/some-things/{id}", someThingDeletionHandler).Methods("DELETE")

	r.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusFound, rr.Code)

	found := models.SomeThing{}
	assert.NotNil(t, found.FindByID(ctx, fixture.ID))

	closeTx(t, ctx)
}

func TestSomeThingsHandler(t *testing.T) {
	t.Parallel()

	ctx := getCtx(t)
	org := organisationFixture(ctx, t)
	ctx = contextifyOrgAdmin(ctx, org)

	fixture := someThingFixture(ctx, t, org)

	targetUrl := fmt.Sprintf("/some-things?organisationid=%s", org.ID)

	req, err := http.NewRequest("GET", targetUrl, nil)
	assert.Nil(t, err)

	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()

	r := mux.NewRouter()

	r.HandleFunc("/some-things", someThingsHandler).Methods("GET")

	r.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), fixture.Name, "Name not found")

	closeTx(t, ctx)
}

func someThingFixture(ctx context.Context, t *testing.T, org models.Organisation) (someThing models.SomeThing) {
	someThing.New(
		bandname(),
		bandname(),
		org.ID,
	)
	assert.Nil(t, someThing.Save(ctx))
	return someThing
}

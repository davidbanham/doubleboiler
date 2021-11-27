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

func TestThingCreateHandler(t *testing.T) {
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
		URL:    &url.URL{Path: "/things"},
		Form:   form,
	}

	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	thingCreateOrUpdateHandler(rr, req)

	assert.Equal(t, http.StatusFound, rr.Code)

	closeTx(t, ctx)
}

func TestThingHandler(t *testing.T) {
	t.Parallel()

	ctx := getCtx(t)
	org := organisationFixture(ctx, t)
	ctx = contextifyOrgAdmin(ctx, org)

	fixture := thingFixture(ctx, t, org)

	req, err := http.NewRequest("GET", "/things/"+fixture.ID, nil)
	req = req.WithContext(ctx)

	assert.Nil(t, err)

	rr := httptest.NewRecorder()

	r := mux.NewRouter()

	r.HandleFunc("/things/{id}", thingHandler).Methods("GET")

	r.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), fixture.Name, "Name not found")

	closeTx(t, ctx)
}

func TestThingsHandler(t *testing.T) {
	t.Parallel()

	ctx := getCtx(t)
	org := organisationFixture(ctx, t)
	ctx = contextifyOrgAdmin(ctx, org)

	fixture := thingFixture(ctx, t, org)

	targetUrl := fmt.Sprintf("/things?organisationid=%s", org.ID)

	req, err := http.NewRequest("GET", targetUrl, nil)
	assert.Nil(t, err)

	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()

	r := mux.NewRouter()

	r.HandleFunc("/things", thingsHandler).Methods("GET")

	r.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), fixture.Name, "Name not found")

	closeTx(t, ctx)
}

func thingFixture(ctx context.Context, t *testing.T, org models.Organisation) (thing models.Thing) {
	thing.New(
		bandname(),
		bandname(),
		org.ID,
	)
	err := thing.Save(ctx)
	assert.Nil(t, err)
	return thing
}

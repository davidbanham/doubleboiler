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

func bandEmail() string {
	return fmt.Sprintf("%s@example.com", bandname())
}

func TestUserCreateOrUpdateHandler(t *testing.T) {
	t.Parallel()
	ctx := getCtx(t)

	form := url.Values{
		"email":       {bandEmail()},
		"password":    {bandname()},
		"orgname":     {bandname()},
		"country":     {"Australia"},
		"orgcurrency": {"AUD"},
		"terms":       {"agreed"},
	}
	req := &http.Request{
		Method: "POST",
		URL:    &url.URL{Path: "/users"},
		Form:   form,
	}
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	userCreateOrUpdateHandler(rr, req)

	assert.Equal(t, http.StatusFound, rr.Code)

	// Check that test data is created

	org := models.Organisation{}
	assert.Nil(t, org.FindByColumn(ctx, "name", form.Get("orgname")))

	closeTx(t, ctx)
}

func TestUserHandler(t *testing.T) {
	t.Parallel()
	ctx := getCtx(t)

	fixture, _ := userFixture(ctx, t)

	ctx = context.WithValue(ctx, "user", fixture)

	req, err := http.NewRequest("GET", "/users/"+fixture.ID, nil)
	req = req.WithContext(ctx)

	assert.Nil(t, err)

	rr := httptest.NewRecorder()

	r := mux.NewRouter()

	r.HandleFunc("/users/{id}", userHandler).Methods("GET")

	r.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), fixture.Email, "Email not found")

	closeTx(t, ctx)
}

func TestUsersHandler(t *testing.T) {
	t.Parallel()
	ctx := getCtx(t)

	fixture, _ := userFixture(ctx, t)
	fixture.SuperAdmin = true

	req, err := http.NewRequest("GET", "/users", nil)
	req = req.WithContext(ctx)
	req = contextify(fixture, req)

	assert.Nil(t, err)

	rr := httptest.NewRecorder()

	r := mux.NewRouter()

	r.HandleFunc("/users", usersHandler).Methods("GET")

	r.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), fixture.Email, "Email not found")

	closeTx(t, ctx)
}

func TestUsersInvalidInputChecking(t *testing.T) {
	t.Parallel()
	ctx := getCtx(t)

	form := url.Values{
		"lol": {"wut"},
	}
	req := &http.Request{
		Method: "POST",
		URL:    &url.URL{Path: "/users"},
		Form:   form,
	}
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	userCreateOrUpdateHandler(rr, req)

	assert.Equal(t, rr.Code, http.StatusBadRequest)
	assert.Contains(t, rr.Body.String(), "Invalid email")

	closeTx(t, ctx)
}

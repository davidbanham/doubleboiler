package routes

import (
	"context"
	"doubleboiler/models"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoginHandler(t *testing.T) {
	t.Parallel()
	ctx := getCtx(t)

	fixture, password := userFixture(ctx, t)

	form := url.Values{
		"email":    {fixture.Email},
		"password": {password},
	}
	req := &http.Request{
		Method: "POST",
		URL:    &url.URL{Path: "/login"},
		Form:   form,
	}
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	loginHandler(rr, req)

	assert.Equal(t, http.StatusFound, rr.Code)
}

func TestServeLogin(t *testing.T) {
	t.Parallel()
	ctx := getCtx(t)

	req, err := http.NewRequest("GET", "/login", nil)
	req = req.WithContext(ctx)

	assert.Nil(t, err)

	rr := httptest.NewRecorder()

	serveLogin(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), "Email", "Email not found")
	assert.Contains(t, rr.Body.String(), "Password", "Password not found")
}

func userFixture(ctx context.Context, t *testing.T) (models.User, string) {
	rawpass := bandname()
	u := models.User{}
	u.New(
		bandname(),
		rawpass,
	)
	assert.Nil(t, u.Save(ctx))
	return u, rawpass
}

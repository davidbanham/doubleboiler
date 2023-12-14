package routes

import (
	"doubleboiler/config"
	"doubleboiler/models"
	"doubleboiler/util"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSignupFlow(t *testing.T) {
	t.Parallel()
	ctx := getCtx(t)

	email := bandname() + "@example.com"
	orgname := bandname()

	form := url.Values{
		"email":   {email},
		"orgname": {orgname},
		"next":    {"signup-success"},
		"terms":   {"agreed"},
	}

	req := &http.Request{
		Method: "POST",
		URL:    &url.URL{Path: "/users"},
		Form:   form,
	}
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	userCreateOrUpdateHandler(rr, req)

	assert.Equal(t, 302, rr.Code)

	u := models.User{}
	u.FindByColumn(ctx, "email", email)
	assert.False(t, u.Verified)

	orgs := models.Organisations{}
	criteria := models.Criteria{}
	models.AddCustomQuery(models.OrganisationsContainingUser{ID: u.ID}, &criteria)
	orgs.FindAll(ctx, criteria)
	orgfound := false
	for _, o := range orgs.Data {
		if o.Name == orgname {
			orgfound = true
		}
	}
	assert.True(t, orgfound, "Org not found in Contains query")

	closeTx(t, ctx)
}

func TestSignupFlowDuplicateEmail(t *testing.T) {
	t.Parallel()
	ctx := getCtx(t)
	defer closeTx(t, ctx)

	fix, _ := userFixture(ctx, t)

	pass := bandname()
	form := url.Values{
		"email":            {fix.Email},
		"password":         {pass},
		"confirm-password": {pass},
		"next":             {"signup"},
		"orgname":          {bandname()},
		"country":          {"Australia"},
		"orgcurrency":      {"AUD"},
		"terms":            {"agreed"},
	}

	req := &http.Request{
		Method: "POST",
		URL:    &url.URL{Path: "/users"},
		Form:   form,
	}
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	userCreateOrUpdateHandler(rr, req)

	assert.Equal(t, http.StatusConflict, rr.Code)
}

func TestVerificationHandlerValid(t *testing.T) {
	t.Parallel()
	ctx := getCtx(t)

	fix, _ := userFixture(ctx, t)

	// Simulate verification URL click
	token := util.CalcToken(config.SECRET, 1, fix.Email)
	escaped := url.QueryEscape(token.String())

	target := fmt.Sprintf("/verify?expiry=%s&uid=%s&token=%s", token.ExpiryString(), fix.ID, escaped)
	req := httptest.NewRequest("GET", target, nil)
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	verifyHandler(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), "Just choose your password")

	u := models.User{}
	u.FindByID(ctx, fix.ID)
	assert.False(t, u.Verified)

	// Set password for new user
	pass := bandname()
	form := url.Values{
		"id":               {fix.ID},
		"revision":         {fix.Revision},
		"email":            {bandname()},
		"password":         {pass},
		"confirm-password": {pass},
		"terms":            {"agreed"},
		"token":            {token.String()},
		"expiry":           {token.ExpiryString()},
	}
	req2 := &http.Request{
		Method: "POST",
		URL:    &url.URL{Path: "/users"},
		Form:   form,
	}
	req2 = req2.WithContext(ctx)
	req2 = contextify(fix, req2)

	rr2 := httptest.NewRecorder()
	userCreateOrUpdateHandler(rr2, req2)

	assert.Equal(t, http.StatusFound, rr2.Code)

	u.FindByID(ctx, fix.ID)
	assert.True(t, u.Verified)

	closeTx(t, ctx)
}

func TestVerificationHandlerInvalidToken(t *testing.T) {
	t.Parallel()
	ctx := getCtx(t)

	fix, _ := userFixture(ctx, t)
	expiry := util.CalcExpiry(1)

	target := fmt.Sprintf("/verify?expiry=%s&uid=%s&token=%s", expiry, fix.ID, "hackhackhack")

	req := httptest.NewRequest("GET", target, nil)
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	verifyHandler(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
	assert.Contains(t, rr.Body.String(), "Invalid token")

	u := models.User{}
	u.FindByID(ctx, u.ID)
	assert.False(t, u.Verified)

	closeTx(t, ctx)
}

func TestVerificationHandlerExpiredToken(t *testing.T) {
	t.Parallel()
	ctx := getCtx(t)

	fix, _ := userFixture(ctx, t)
	token := util.CalcToken(config.SECRET, -1, fix.Email)
	escaped := url.QueryEscape(token.String())

	target := fmt.Sprintf("/verify?expiry=%s&uid=%s&token=%s", token.ExpiryString(), fix.ID, escaped)

	req := httptest.NewRequest("GET", target, nil)
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	verifyHandler(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
	assert.Contains(t, rr.Body.String(), "Invalid token")

	u := models.User{}
	u.FindByID(ctx, u.ID)
	assert.False(t, u.Verified)

	closeTx(t, ctx)
}

func TestVerificationHandlerInvalidExpiry(t *testing.T) {
	t.Parallel()
	ctx := getCtx(t)

	fix, _ := userFixture(ctx, t)
	token := util.CalcToken(config.SECRET, 1, fix.Email)
	escaped := url.QueryEscape(token.String())

	target := fmt.Sprintf("/verify?expiry=%s&uid=%s&token=%s", "hackhackhack", fix.ID, escaped)

	req := httptest.NewRequest("GET", target, nil)
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	verifyHandler(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
	assert.Contains(t, rr.Body.String(), "Invalid token")

	u := models.User{}
	u.FindByID(ctx, u.ID)
	assert.False(t, u.Verified)

	closeTx(t, ctx)
}

func TestServeSignup(t *testing.T) {
	t.Parallel()
	ctx := getCtx(t)

	req, err := http.NewRequest("GET", "/signup", nil)
	req = req.WithContext(ctx)

	assert.Nil(t, err)

	rr := httptest.NewRecorder()

	serveSignup(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), "Email", "Email not found")
	assert.Contains(t, rr.Body.String(), "orgname")

	closeTx(t, ctx)
}

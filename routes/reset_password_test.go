package routes

import (
	"doubleboiler/config"
	"doubleboiler/util"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPasswordResetHandler(t *testing.T) {
	t.Parallel()
	ctx := getCtx(t)

	fix, _ := userFixture(ctx, t)

	fix.Verified = true
	err := fix.Save(ctx)
	assert.Nil(t, err)

	form := url.Values{
		"email": {fix.Email},
	}

	req := &http.Request{
		Method: "POST",
		URL:    &url.URL{Path: "/reset-password"},
		Form:   form,
	}
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	passwordResetHandler(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), "A password reset email has been sent")

	closeTx(t, ctx)
}

func TestServeResetPasswordGenerateToken(t *testing.T) {
	t.Parallel()
	ctx := getCtx(t)

	req, err := http.NewRequest("GET", "/reset-password", nil)
	req = req.WithContext(ctx)

	assert.Nil(t, err)

	rr := httptest.NewRecorder()

	serveResetPassword(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), "Reset Password")

	closeTx(t, ctx)
}

func TestServeResetPasswordWithToken(t *testing.T) {
	t.Parallel()
	ctx := getCtx(t)

	fix, _ := userFixture(ctx, t)

	fix.Verified = true
	err := fix.Save(ctx)
	assert.Nil(t, err)

	token := util.CalcToken(config.SECRET, 1, fix.Email)
	escaped := url.QueryEscape(token.String())
	req, err := http.NewRequest("GET", fmt.Sprintf("/reset-password?expiry=%s&uid=%s&token=%s", token.ExpiryString(), fix.ID, escaped), nil)
	req = req.WithContext(ctx)

	assert.Nil(t, err)

	rr := httptest.NewRecorder()

	serveResetPassword(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), "New Password")

	closeTx(t, ctx)
}

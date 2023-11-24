package routes

import (
	"doubleboiler/config"
	"doubleboiler/models"
	"doubleboiler/util"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCSRFMiddleware(t *testing.T) {
	t.Parallel()

	var middle = csrfMiddleware(final)

	var rr *httptest.ResponseRecorder
	var err error
	var req *http.Request
	var userFix models.User

	// It should deny a POST without a valid csrf token
	rr = httptest.NewRecorder()
	req, err = http.NewRequest("POST", "/stuff", nil)
	req = contextify(userFix, req)
	assert.Nil(t, err)

	middle.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusForbidden, rr.Code)

	// It shouldn't touch a GET
	rr = httptest.NewRecorder()
	req, err = http.NewRequest("GET", "/stuff", nil)
	assert.Nil(t, err)

	middle.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)

	// It should allow a POST with a valid csrf token
	rr = httptest.NewRecorder()
	values := url.Values{
		"csrf": {
			util.CalcToken(config.SECRET, 0, userFix.ID).String(),
		},
	}
	req, err = http.NewRequest("POST", "/stuff", strings.NewReader(values.Encode()))
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("Content-Length", strconv.Itoa(len(values)))
	req = contextify(userFix, req)
	assert.Nil(t, err)

	middle.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
}

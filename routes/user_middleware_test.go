package routes

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
)

var final = http.HandlerFunc(okHandler)

func TestUserMiddleware(t *testing.T) {
	t.Parallel()

	var middle = userMiddleware(loginMiddleware(final))

	var rr *httptest.ResponseRecorder
	var err error
	var req *http.Request
	var loc *url.URL
	var form url.Values

	// It should redirect a non-whitelisted url
	rr = httptest.NewRecorder()
	req, err = http.NewRequest("GET", "/hackhackhack", nil)
	assert.Nil(t, err)

	middle.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusFound, rr.Code)

	// It should serve a whitelisted url
	rr = httptest.NewRecorder()
	req, err = http.NewRequest("GET", "/img/some_image.jpg", nil)
	assert.Nil(t, err)

	middle.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), "ok")

	// It should reject an unauthed GET to /users
	rr = httptest.NewRecorder()
	req, err = http.NewRequest("GET", "/users", nil)
	assert.Nil(t, err)

	middle.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusFound, rr.Code)

	// It should accept a POST to /users with no id
	rr = httptest.NewRecorder()
	form = url.Values{
		"email":    {bandname()},
		"password": {bandname()},
	}
	req = &http.Request{
		Method: "POST",
		URL:    &url.URL{Path: "/users"},
		Form:   form,
	}

	middle.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)

	// It should reject a POST to /users with an ID but no auth
	rr = httptest.NewRecorder()
	form = url.Values{
		"id":       {uuid.NewV4().String()},
		"email":    {bandname()},
		"password": {bandname()},
	}
	req = &http.Request{
		Method: "POST",
		URL:    &url.URL{Path: "/users"},
		Form:   form,
	}

	middle.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusFound, rr.Code)
	loc, err = rr.Result().Location()
	assert.Nil(t, err)
	assert.Equal(t, "/login", loc.Path)

}

func okHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(200)
	w.Write([]byte("ok"))
}

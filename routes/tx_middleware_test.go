package routes

import (
	"doubleboiler/models"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func queryAssertingHandler(w http.ResponseWriter, r *http.Request) {
	tx := r.Context().Value("tx").(models.Querier)
	if tx != nil {
		w.WriteHeader(200)
		w.Write([]byte("ok"))
	}
}

func TestTXMiddleware(t *testing.T) {
	queryAsserter := http.HandlerFunc(queryAssertingHandler)

	t.Parallel()

	var middle = txMiddleware(queryAsserter)

	var rr *httptest.ResponseRecorder
	var err error
	var req *http.Request

	// It should populate a Querier into tx
	rr = httptest.NewRecorder()
	req, err = http.NewRequest("POST", "/stuff", nil)
	assert.Nil(t, err)

	middle.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
}

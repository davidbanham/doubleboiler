package routes

import (
	"context"
	"database/sql"
	"doubleboiler/models"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
)

func TestCommunicationsHandler(t *testing.T) {
	t.Parallel()

	ctx := getCtx(t)
	org := organisationFixture(ctx, t)
	user, _ := userFixture(ctx, t)
	ctx = contextifyOrgAdmin(ctx, org)

	fixture := communicationFixture(ctx, t, user, org)

	targetUrl := fmt.Sprintf("/communications?organisationid=%s", org.ID)

	req, err := http.NewRequest("GET", targetUrl, nil)
	assert.Nil(t, err)

	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()

	r := mux.NewRouter()

	r.HandleFunc("/communications", communicationsHandler).Methods("GET")

	r.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), fixture.Subject, "Subject not found")

	closeTx(t, ctx)
}

func communicationFixture(ctx context.Context, t *testing.T, user models.User, org models.Organisation) (communication models.Communication) {
	communication.New(org.ID, "email", bandname())
	communication.UserID = sql.NullString{
		Valid:  true,
		String: user.ID,
	}
	assert.Nil(t, communication.Save(ctx))
	return communication
}

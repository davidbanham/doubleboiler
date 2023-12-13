package routes

import (
	"net/http"

	"github.com/gorilla/mux"
)

func init() {
	r.Path("/flashes/{id}/delete").
		Methods("POST").
		HandlerFunc(flashDeleteHandler)
}

func flashDeleteHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	user := userFromContext(r.Context())
	if err := user.DeleteFlash(r.Context(), vars["id"]); err != nil {
		errRes(w, r, http.StatusInternalServerError, "A database error has occurred", err)
		return
	}
}

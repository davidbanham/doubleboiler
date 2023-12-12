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
	for _, flash := range user.Flashes {
		if flash.ID == vars["id"] {
			if err := user.DeleteFlash(r.Context(), flash); err != nil {
				errRes(w, r, http.StatusInternalServerError, "A database error has occurred", err)
				return
			}
		}
	}
}

package routes

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strings"

	kewpie "github.com/davidbanham/kewpie_go/v3"
	"github.com/davidbanham/notifications"
)

func init() {
	// NOTE by default we assume auth is handled by Cloud Run as per:
	// https://cloud.google.com/run/docs/tutorials/pubsub#integrating-pubsub
	// If you're not deploying to Cloud Run you must authenticate incoming messages yourself
	r.Path("/webhooks/send-email").
		Methods("POST").
		HandlerFunc(sendEmail)
}

func sendEmail(w http.ResponseWriter, r *http.Request) {
	task := kewpie.Task{}

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		errRes(w, r, http.StatusInternalServerError, "Error reading webhook body", err)
		return
	}

	if err := json.Unmarshal(body, &task); err != nil {
		errRes(w, r, http.StatusInternalServerError, "Error decoding webhook body", err)
		return
	}

	input := notifications.Email{}

	if err := task.Unmarshal(&input); err != nil {
		errRes(w, r, http.StatusInternalServerError, "Could not decode task body", err)
	}

	if input.To == "" {
		errRes(w, r, http.StatusBadRequest, "No email specified", err)
		return
	}

	if err := notifications.SendEmail(input); err != nil {
		if strings.Contains(err.Error(), "status code: 400") {
			errRes(w, r, http.StatusBadRequest, "Bad email payload", err)
			return
		}
		errRes(w, r, http.StatusInternalServerError, "Error communicating with email gateway", err)
		return
	}

	w.WriteHeader(200)
	w.Write([]byte("ok"))
	return
}

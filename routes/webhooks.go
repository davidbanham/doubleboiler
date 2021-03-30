package routes

import (
	"context"
	"doubleboiler/config"
	"doubleboiler/workers"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	kewpie "github.com/davidbanham/kewpie_go/v3"
	"github.com/davidbanham/notifications"
)

func init() {
	taskErrorHandler := func(ctx context.Context, httpErr kewpie.HTTPError) {
		fmt.Println("ERROR", httpErr.Error.Error())
	}

	for queueName, handler := range workers.Handlers {
		r.Path("/webhooks/tasks/" + queueName).
			Methods("POST").
			HandlerFunc(config.QUEUE.SubscribeHTTP(config.SECRET, handler, taskErrorHandler))
	}
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

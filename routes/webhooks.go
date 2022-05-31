package routes

import (
	"context"
	"doubleboiler/config"
	"doubleboiler/workers"
	"fmt"

	kewpie "github.com/davidbanham/kewpie_go/v3"
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

package workers

import (
	"doubleboiler/config"
	"doubleboiler/workers/send_email"

	kewpie "github.com/davidbanham/kewpie_go/v3"
)

type handler struct {
	path string
}

var queue kewpie.Kewpie

func Init() {
	send_email.Init()
}

var Handlers = map[string]kewpie.Handler{
	config.SEND_EMAIL_QUEUE_NAME: send_email.Handler{},
}

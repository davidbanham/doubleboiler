package send_email

import (
	"context"
	"doubleboiler/config"
	"fmt"
	"log"
	"strings"

	kewpie "github.com/davidbanham/kewpie_go/v3"
	"github.com/davidbanham/notifications"
)

func Init() {
	go func() {
		if err := config.QUEUE.Subscribe(context.Background(), config.SEND_EMAIL_QUEUE_NAME, Handler{}); err != nil {
			log.Println("ERROR Queue error", config.SEND_EMAIL_QUEUE_NAME, err)
		}
	}()
}

type Handler struct{}

func (h Handler) Handle(task kewpie.Task) (requeue bool, err error) {
	input := notifications.Email{}

	if err := task.Unmarshal(&input); err != nil {
		config.ReportError(err)
		return false, err
	}

	if input.To == "" {
		return false, fmt.Errorf("No To email specified")
	}

	if err := notifications.SendEmail(input); err != nil {
		if strings.Contains(err.Error(), "status code: 400") {
			return false, err
		}
		return true, err
	}

	return false, nil
}

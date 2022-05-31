package send_email

import (
	"context"
	"doubleboiler/config"
	"doubleboiler/logger"
	"doubleboiler/models"
	"doubleboiler/util"
	"fmt"
	"log"
	"strings"

	kewpie "github.com/davidbanham/kewpie_go/v3"
	"github.com/davidbanham/notifications"
)

func Init() {
	go func() {
		if err := config.QUEUE.Subscribe(context.Background(), config.SEND_EMAIL_QUEUE_NAME, Handler{}); err != nil {
			logger.Log(context.Background(), logger.Error, "Queue error", config.SEND_EMAIL_QUEUE_NAME, err)
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

	ctx, tx, err := util.GetTxCtx()
	if err != nil {
		util.RollbackTx(ctx)
		return true, err
	}
	wtf := task.Tags.Get("user_id")
	log.Printf("DEBUG wtf: %+v \n", wtf)

	if task.Tags.Get("user_id") != "" {
		user := models.User{}
		if err := user.FindByID(ctx, task.Tags.Get("user_id")); err != nil {
			util.RollbackTx(ctx)
			return true, err
		}

		subject := input.Subject
		if task.Tags.Get("communication_subject") != "" {
			subject = task.Tags.Get("communication_subject")
		}

		if err := models.LogUserCommunication(ctx, task.Tags.Get("organisation_id"), user, "email", subject); err != nil {
			util.RollbackTx(ctx)
			return false, err
		}
	}

	if err := notifications.SendEmail(input); err != nil {
		util.RollbackTx(ctx)
		if strings.Contains(err.Error(), "status code: 400") {
			return false, err
		}
		return true, err
	}

	if err := tx.Commit(); err != nil {
		config.ReportError(err)
		return true, err
	}

	return false, nil
}

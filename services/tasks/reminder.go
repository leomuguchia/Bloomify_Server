package tasks

import (
	"bloomify/models"
	"encoding/json"
	"time"

	"github.com/hibiken/asynq"
)

const TypeSendReminder = "reminder:send"

func NewReminderTask(payload models.ReminderPayload, fireAt time.Time) (*asynq.Task, []asynq.Option, error) {
	b, err := json.Marshal(payload)
	if err != nil {
		return nil, nil, err
	}
	task := asynq.NewTask(TypeSendReminder, b)
	opts := []asynq.Option{asynq.ProcessAt(fireAt)}

	return task, opts, nil
}

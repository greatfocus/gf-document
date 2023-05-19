package task

import (
	"context"
	"encoding/json"
	"math"
	"os"
	"time"

	"github.com/greatfocus/gf-document/models"
	"github.com/greatfocus/gf-document/repositories"
	"github.com/greatfocus/gf-document/services"
	broker "github.com/greatfocus/gf-sframe/broker"
	"github.com/greatfocus/gf-sframe/server"
	amqp "github.com/rabbitmq/amqp091-go"
)

// Tasks struct
type Tasks struct {
	fileRepository *repositories.FileRepository
	fileService    *services.FileService
	server         *server.Server
}

// Init required parameters
func (t *Tasks) Init(s *server.Server) {
	t.fileRepository = &repositories.FileRepository{}
	t.fileRepository.Init(s.Database, s.Cache)

	t.fileService = &services.FileService{}
	t.fileService.Init(s.Database, s.Cache, s.JWT, s.Logger)

	t.server = s
}

// RemoveUnTemporaryFile start the job to remove temp files
func (t *Tasks) RemoveTemporaryFile() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(t.server.Timeout)*time.Second)
	defer cancel()

	t.server.Logger.Info("Scheduler_RemoveUnTemporaryFile started")
	msgs, err := t.fileRepository.GetFilesByStatus(ctx, t.server.JWT.Secret(), "temp")
	if err != nil {
		t.server.Logger.Warn("Scheduler_RemoveUnTemporaryFile Error fetching files")
		return
	}

	for i := 0; i < len(msgs); i++ {
		// calculate total number of days
		duration := time.Since(msgs[i].CreatedOn)
		mins := int(duration.Minutes() / 24 * 60)
		if math.Abs(float64(mins)) > 60 {
			t.fileService.DeleteFromJob(ctx, t.server.JWT.Secret(), msgs[i].ID)
		}
	}

	t.server.Logger.Info("Scheduler_RemoveUnTemporaryFile ended")
}

// RemoveUnTemporaryFile start the job to remove temp files
func (t *Tasks) EventsListerner() {
	// broker.Publish("post.event.delete", "ped", uuid.New().String(), []byte(`{"id":"c45d75d7-276f-4f53-bffb-2b1b5a7119e9", "refId": "c45d75d7-276f-4f53-bffb-2b1b5a7119e9"}`))
	go func() {
		approval := broker.ConsumerParam{
			Handler:       t.approveDocument,
			QueueName:     "post.event.approved",
			AppId:         t.server.Name,
			ConnectionStr: os.Getenv("RABBITMQ_URL"),
		}
		for {
			time.Sleep(10 * time.Second)
			broker.Consumer(approval)
		}
	}()
	go func() {
		for {
			delete := broker.ConsumerParam{
				Handler:       t.deleteDocument,
				QueueName:     "post.event.delete",
				AppId:         t.server.Name,
				ConnectionStr: os.Getenv("RABBITMQ_URL"),
			}
			time.Sleep(10 * time.Second)
			broker.Consumer(delete)
		}
	}()
}

func (t *Tasks) approveDocument(d amqp.Delivery) error {
	if d.Body != nil {
		// validate if json object
		file := models.File{}
		err := json.Unmarshal(d.Body, &file)
		if err != nil {
			return err
		}
		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(t.server.Timeout)*time.Second)
		defer cancel()

		// validate payload rules
		err = file.ValidateFile("update")
		if err != nil {
			return err
		}

		// create token
		_, err = t.fileService.Update(ctx, t.server.JWT.Secret(), file)
		return err
	}
	return nil
}

func (t *Tasks) deleteDocument(d amqp.Delivery) error {
	if d.Body != nil {
		// validate if json object
		file := models.File{}
		err := json.Unmarshal(d.Body, &file)
		if err != nil {
			return err
		}
		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(t.server.Timeout)*time.Second)
		defer cancel()

		success, err := t.fileService.Delete(ctx, t.server.JWT.Secret(), file.ID)
		if !success || err != nil {
			return err
		}
		return nil
	}
	return nil
}

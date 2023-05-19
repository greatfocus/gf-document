package main

import (
	"time"

	"github.com/go-co-op/gocron"
	"github.com/greatfocus/gf-document/router"
	"github.com/greatfocus/gf-document/task"
	"github.com/greatfocus/gf-sframe/server"
	_ "github.com/lib/pq"
)

// Entry point to the solution
func main() {

	service := server.NewServer("gf-document", "document")
	service.Mux = router.LoadRouter(service)

	// background task
	tasks := task.Tasks{}
	tasks.Init(service)
	schedule := gocron.NewScheduler(time.UTC)
	schedule.Cron("0 0 * * *").Do(tasks.RemoveTemporaryFile) // every minute

	tasks.EventsListerner()

	service.Start()
}

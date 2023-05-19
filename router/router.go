package router

import (
	"fmt"
	"net/http"
	"time"

	"github.com/greatfocus/gf-document/services"

	"github.com/greatfocus/gf-document/handler"
	"github.com/greatfocus/gf-sframe/server"
)

// Router is exported and used in main.go
func LoadRouter(s *server.Server) *http.ServeMux {
	mux := http.NewServeMux()
	loadHandlers(mux, s)
	s.Logger.Info(fmt.Sprintln("Created routes with handler"))
	return mux
}

// documentRoute created all routes and handlers relating to document controller
func loadHandlers(mux *http.ServeMux, s *server.Server) {
	// initialize services
	fileService := services.FileService{}
	fileService.Init(s.Database, s.Cache, s.JWT, s.Logger)

	fileHandler := handler.File{}
	fileHandler.Init(s, &fileService)
	mux.Handle("/document/file", server.Use(fileHandler,
		server.SetHeaders(),
		server.CheckThrottle(),
		server.CheckCors(),
		server.CheckAllowedIPs(),
		server.ProcessTimeout(time.Duration(s.Timeout)),
		server.WithoutAuth()))
}

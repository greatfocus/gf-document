package handler

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/greatfocus/gf-document/services"
	server "github.com/greatfocus/gf-sframe/server"
)

// File struct
type File struct {
	FileHandler func(http.ResponseWriter, *http.Request)
	fileService *services.FileService
	server      *server.Server
}

// ServeHTTP checks if is valid method
func (f File) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		f.getFiles(w, r)
		return
	}
	if r.Method == http.MethodPost {
		f.upload(w, r)
		return
	}

	// catch all
	// if no method is satisfied return an error
	w.WriteHeader(http.StatusMethodNotAllowed)
	w.Header().Add("Allow", "GET, POST")
}

// ValidateRequest checks if request is valid
func (f File) ValidateRequest(w http.ResponseWriter, r *http.Request) (interface{}, error) {
	data, err := f.server.Request(w, r)
	if err != nil {
		return nil, err
	}

	return data, nil
}

// Init method
func (f *File) Init(s *server.Server, fileService *services.FileService) {
	f.fileService = fileService
	f.server = s
}

// uploadFile upload file
func (f *File) upload(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), time.Duration(f.server.Timeout)*time.Second)
	defer cancel()

	doc, err := f.fileService.Upload(ctx, f.server.JWT.Secret(), r)
	if err != nil {
		derr := errors.New("invalid payload request")
		f.server.Logger.Error(fmt.Sprintf("Error: %v\n", derr))
		f.server.Error(w, r, derr)
		return
	}
	w.WriteHeader(http.StatusOK)
	f.server.Success(w, r, doc)
}

// getFiles method
func (f *File) getFiles(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), time.Duration(f.server.Timeout)*time.Second)
	defer cancel()

	lastID := r.FormValue("lastId")
	id := r.FormValue("id")

	if id != "" {
		file, err := f.fileService.GetFileByID(ctx, f.server.JWT.Secret(), id)
		if err != nil {
			w.WriteHeader(http.StatusUnprocessableEntity)
			f.server.Error(w, r, err)
			return
		}
		w.WriteHeader(http.StatusOK)
		f.server.Success(w, r, file)
		return
	}

	files, err := f.fileService.GetFiles(ctx, f.server.JWT.Secret(), lastID)
	if err != nil {
		w.WriteHeader(http.StatusUnprocessableEntity)
		f.server.Error(w, r, err)
		return
	}
	w.WriteHeader(http.StatusOK)
	f.server.Success(w, r, files)
}

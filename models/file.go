package models

import (
	"errors"
	"strings"
	"time"
)

// File struct
type File struct {
	ID        string    `json:"id,omitempty"`
	RefID     string    `json:"refId,omitempty"`
	Name      string    `json:"name,omitempty"`
	Extension string    `json:"extension,omitempty"`
	Size      int64     `json:"size,omitempty"`
	Status    string    `json:"status,omitempty"`
	CreatedOn time.Time `json:"-"`
}

// ValidateFile check if request is valid
func (f *File) ValidateFile(action string) error {
	switch strings.ToLower(action) {
	case "add":
		if f.Name == "" {
			return errors.New("required Name")
		}
		if f.Extension == "" {
			return errors.New("required Extension")
		}
		if f.Size == 0 {
			return errors.New("required Size")
		}
		return nil
	case "update":
		if f.RefID == "" {
			return errors.New("required RefID")
		}
		if f.ID == "" {
			return errors.New("required ID")
		}
		return nil
	default:
		return errors.New("invalid validation operation")
	}
}

// PrepareFileOutput initiliazes the file request object
func (f *File) PrepareFileOutput(file File) {
	f.ID = file.ID
	f.Status = file.Status
	f.Name = file.Name
}

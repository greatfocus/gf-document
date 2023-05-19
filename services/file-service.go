package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"
	"time"

	"github.com/greatfocus/gf-document/models"
	"github.com/greatfocus/gf-document/repositories"
	"github.com/greatfocus/gf-sframe/database"
	"github.com/greatfocus/gf-sframe/logger"
	"github.com/greatfocus/gf-sframe/server"
	cache "github.com/patrickmn/go-cache"
)

var uploadPath = "./upload"

// FileService struct
type FileService struct {
	fileRepository *repositories.FileRepository
	jwt            server.JWT
	logger         logger.Logger
}

// Init method
func (f *FileService) Init(database database.Database, cache *cache.Cache, jwt server.JWT, logger logger.Logger) {
	f.fileRepository = &repositories.FileRepository{}
	f.fileRepository.Init(database, cache)
	f.jwt = jwt
	f.logger = logger
}

// createFolder make dir
func createFolder(path string) bool {
	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		errDir := os.MkdirAll(path, 0755)
		return errDir == nil
	}
	return true
}

// FileExists check file exist
func fileExists(filename string) bool {
	path := uploadPath + "/Temp"
	createdFolder := createFolder(path)
	if !createdFolder {
		return false
	}

	info, err := os.Stat(path + "/" + filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

// dropFile check file exist
func dropFile(filename string) {
	path := uploadPath + "/Temp"
	os.Remove(path + "/" + filename)
}

// MoveFile cut file after success save
func moveFile(filename string) bool {
	oldLocation := uploadPath + "/Temp"
	newLocation := uploadPath

	createdFolder := createFolder(oldLocation)
	if !createdFolder {
		return false
	}

	createdFolder = createFolder(newLocation)
	if !createdFolder {
		return false
	}

	err := os.Rename(oldLocation+"/"+filename, newLocation+"/"+filename)
	return err == nil
}

// Upload file function
func (f *FileService) Upload(ctx context.Context, enKey string, r *http.Request) (models.File, error) {
	doc := models.File{}
	// Parse our multipart form, 10 << 20 specifies a maximum
	// upload of 10 MB files.
	r.ParseMultipartForm(10 << 20)
	// FormFile returns the first file for the given key `myFile`
	// it also returns the FileHeader so we can get the Filename,
	// the Header and the size of the file
	file, handler, err := r.FormFile("image")
	if err != nil {
		derr := errors.New("cannot create file")
		return doc, derr
	}
	defer file.Close()
	f.logger.Info(fmt.Sprintf("Uploaded File: %+v\n", handler.Filename))
	f.logger.Info(fmt.Sprintf("File Size: %+v\n", handler.Size))
	f.logger.Info(fmt.Sprintf("MIME Header: %+v\n", handler.Header))

	doc.Size = handler.Size
	doc.Status = "new"

	if uploadPath == "" {
		err := errors.New("Upload PATH is not set")
		derr := errors.New("cannot create file")
		f.logger.Error(fmt.Sprintf("Error: %v\n", err))
		return doc, derr
	}

	// Create a temporary file within our temp-images directory that follows
	// a particular naming pattern
	createdFolder := createFolder(uploadPath + "/Temp")
	if !createdFolder {
		return doc, err
	}

	tempFile, err := ioutil.TempFile(uploadPath+"/Temp", "image-*.png")
	re := regexp.MustCompile(`^(.*/)?(?:$|(.+?)(?:(\.[^.]*$)|$))`)
	match2 := re.FindStringSubmatch(tempFile.Name())
	fileName := match2[2] + match2[3]
	doc.Name = fileName
	doc.Extension = match2[3]
	if err != nil {
		return doc, err
	}
	defer tempFile.Close()

	// read all of the contents of our uploaded file into a
	// byte array
	fileBytes, err := ioutil.ReadAll(file)
	if err != nil {
		return doc, err
	}
	// write this byte array to our temporary file
	tempFile.Write(fileBytes)
	// return that we have successfully uploaded our file!
	f.logger.Info(fmt.Sprintf("Successfully Uploaded File: %+v\n", tempFile))

	created, err := f.createFile(ctx, enKey, r, doc)
	if err != nil {
		return doc, err
	}
	return created, nil
}

// CreateFile method
func (f *FileService) createFile(ctx context.Context, enKey string, r *http.Request, file models.File) (models.File, error) {
	// validate token
	file.CreatedOn = time.Now()

	// validate payload rules
	err := file.ValidateFile("add")
	if err != nil {
		return file, err
	}

	fileFound := fileExists(file.Name)
	if !fileFound {
		derr := errors.New("kindly upload choose and upload file")
		f.logger.Error(fmt.Sprintf("Error: %v\n", derr))
		return file, derr
	}

	// insert file
	created, err := f.fileRepository.Create(ctx, enKey, file)
	if err != nil {
		derr := errors.New("failed to upload image")
		f.logger.Error(fmt.Sprintf("Error: %v\n", derr))
		f.fileRepository.Delete(ctx, enKey, created.ID)
		return file, derr
	}

	result := models.File{}
	result.PrepareFileOutput(created)
	return result, nil
}

// GetFiles method gets file by lastID
func (f *FileService) GetFiles(ctx context.Context, enKey string, lastID string) ([]models.File, error) {
	files, err := f.fileRepository.GetFiles(ctx, enKey, lastID)
	if err != nil {
		return files, err
	}
	return files, nil
}

// GetFileByID method gets file by ID
func (f *FileService) GetFileByID(ctx context.Context, enKey string, id string) (models.File, error) {
	file, err := f.fileRepository.GetFileByID(ctx, enKey, id)
	if err != nil && err != sql.ErrNoRows {
		return file, err
	}
	return file, nil
}

// Update method updates the file record
func (f *FileService) Update(ctx context.Context, enKey string, file models.File) (models.File, error) {
	// forensic should be done
	foundFile, err := f.fileRepository.GetFileByID(ctx, enKey, file.ID)
	if err != nil {
		return file, err
	}

	// updated File
	file.Status = "approved"
	err = f.fileRepository.Update(ctx, enKey, file)
	if err != nil {
		derr := errors.New("failed to update File")
		f.logger.Error(fmt.Sprintf("Error: %v\n", derr))
		return file, derr
	}

	// move the file from temp
	moveFile(foundFile.Name)

	result := models.File{}
	result.PrepareFileOutput(file)
	return result, nil
}

// Delete method delete the file record
func (f *FileService) Delete(ctx context.Context, enKey string, id string) (bool, error) {
	// payment should doen before verification
	insertedFile, err := f.fileRepository.GetFileByID(ctx, enKey, id)
	if err != nil {
		return false, errors.New("record does not exist")
	}

	if insertedFile.Status == "approved" {
		return false, errors.New("you are not allowed to delete file")
	}

	err = f.fileRepository.Delete(ctx, enKey, id)
	if err != nil {
		derr := errors.New("failed to delete File")
		f.logger.Error(fmt.Sprintf("Error: %v\n", derr))
		return false, derr
	}
	dropFile(insertedFile.Name)

	result := models.File{}
	result.PrepareFileOutput(insertedFile)
	return true, nil
}

// DeleteFromJob method delete the file record
func (f *FileService) DeleteFromJob(ctx context.Context, enKey string, id string) (bool, error) {
	// payment should doen before verification
	insertedFile, err := f.fileRepository.GetFileByID(ctx, enKey, id)
	if err != nil {
		return false, errors.New("record does not exist")
	}

	if insertedFile.Status == "approved" {
		return false, errors.New("you are not allowed to delete file")
	}

	err = f.fileRepository.Delete(ctx, enKey, id)
	if err != nil {
		derr := errors.New("failed to delete File")
		f.logger.Error(fmt.Sprintf("Error: %v\n", derr))
		return false, derr
	}
	dropFile(insertedFile.Name)

	result := models.File{}
	result.PrepareFileOutput(insertedFile)
	return true, nil
}

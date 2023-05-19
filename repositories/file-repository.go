package repositories

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/greatfocus/gf-document/models"
	"github.com/greatfocus/gf-sframe/database"
	cache "github.com/patrickmn/go-cache"
)

// fileRepositoryCacheKeys array
var fileRepositoryCacheKeys = []string{}

// FileRepository struct
type FileRepository struct {
	db    database.Database
	cache *cache.Cache
}

// Init method
func (repo *FileRepository) Init(database database.Database, cache *cache.Cache) {
	repo.db = database
	repo.cache = cache
}

// Create method
func (repo *FileRepository) Create(ctx context.Context, enKey string, doc models.File) (models.File, error) {
	var id = uuid.New().String()
	statement := `
    insert into files (id, name, extension, size, status)
    values ($1, PGP_SYM_ENCRYPT($2, '` + enKey + `'), $3, $4, $5)
    returning id
  	`
	_, inserted := repo.db.Insert(ctx, statement, id, doc.Name, doc.Extension, doc.Size, doc.Status)
	if !inserted {
		return doc, errors.New("create doc failed")
	}
	doc.ID = id
	repo.deleteCache()
	return doc, nil
}

// GetFileByID method
func (repo *FileRepository) GetFileByID(ctx context.Context, enKey string, id string) (models.File, error) {
	// get data from cache
	var key = "FileRepository.GetFileByID." + id
	found, cache := repo.getFileCache(key)
	if found {
		return cache, nil
	}

	query := `
	select id, pgp_sym_decrypt(name::bytea, '` + enKey + `'), status, createdOn
	from files
	where id = $1
	`

	row := repo.db.Select(ctx, query, id)
	file := models.File{}
	err := row.Scan(&file.ID, &file.Name, &file.Status, &file.CreatedOn)
	switch err {
	case sql.ErrNoRows:
		return file, err
	case nil:
		// update cache
		repo.setFileCache(key, file)
		return file, nil
	default:
		return file, err
	}
}

// GetFiles method
func (repo *FileRepository) GetFiles(ctx context.Context, enKey string, lastID string) ([]models.File, error) {
	// get data from cache
	var key = "FileRepository.GetFiles" + lastID
	found, cache := repo.getFilesCache(key)
	if found {
		return cache, nil
	}

	var query string
	var rows *sql.Rows
	var err error
	if lastID != "" {
		query = `
		select id, pgp_sym_decrypt(name::bytea, '` + enKey + `'), extension, size, status, createdOn
		from files
		where id >= $1
		order BY createdOn DESC limit 20
		`
		rows, err = repo.db.Query(ctx, query, lastID)
	} else {
		query = `
		select id, pgp_sym_decrypt(name::bytea, '` + enKey + `'), extension, size, status, createdOn
		from files
		order BY createdOn DESC limit 20
		`
		rows, err = repo.db.Query(ctx, query)
	}

	if err != nil {
		return nil, err
	}
	defer func() {
		_ = rows.Close()
	}()
	result, err := getFilesFromRows(rows)
	if err != nil {
		return nil, err
	}

	// update cache
	repo.setFilesCache(key, result)
	return result, nil
}

// Update method update file
func (repo *FileRepository) Update(ctx context.Context, enKey string, file models.File) error {
	statement := `
    update files
	set 
	 	refId=$2,
		status=$3
    where id=$1
  	`
	updated := repo.db.Update(ctx, statement, file.ID, file.RefID, file.Status)
	if !updated {
		return errors.New("update file failed")
	}

	repo.deleteCache()
	return nil
}

// Delete method
func (repo *FileRepository) Delete(ctx context.Context, enKey string, id string) error {
	query := `
    delete from files
    where id=$1
  	`
	deleted := repo.db.Delete(ctx, query, id)
	if !deleted {
		return errors.New("update file failed")
	}

	repo.deleteCache()
	return nil
}

// getFileCache method get cache for files
func (repo *FileRepository) getFileCache(key string) (bool, models.File) {
	var data models.File
	if x, found := repo.cache.Get(key); found {
		data = x.(models.File)
		return found, data
	}
	return false, data
}

// setFileCache method set cache for file
func (repo *FileRepository) setFileCache(key string, file models.File) {
	if file != (models.File{}) {
		fileRepositoryCacheKeys = append(fileRepositoryCacheKeys, key)
		repo.cache.Set(key, file, 5*time.Minute)
	}
}

// getFileCache method get cache for files
func (repo *FileRepository) getFilesCache(key string) (bool, []models.File) {
	var data []models.File
	if x, found := repo.cache.Get(key); found {
		data = x.([]models.File)
		return found, data
	}
	return false, data
}

// setFileCache method set cache for files
func (repo *FileRepository) setFilesCache(key string, files []models.File) {
	if len(files) > 0 {
		fileRepositoryCacheKeys = append(fileRepositoryCacheKeys, key)
		repo.cache.Set(key, files, 10*time.Minute)
	}
}

// deleteCache method to delete
func (repo *FileRepository) deleteCache() {
	if len(fileRepositoryCacheKeys) > 0 {
		for i := 0; i < len(fileRepositoryCacheKeys); i++ {
			repo.cache.Delete(fileRepositoryCacheKeys[i])
		}
		fileRepositoryCacheKeys = []string{}
	}
}

// prepare files row
func getFilesFromRows(rows *sql.Rows) ([]models.File, error) {
	files := []models.File{}
	for rows.Next() {
		var file models.File
		err := rows.Scan(&file.ID, &file.Name, &file.Extension,
			&file.Size, &file.Status, &file.CreatedOn)
		if err != nil {
			return nil, err
		}
		files = append(files, file)
	}

	return files, nil
}

// GetFilesByStatus method
func (repo *FileRepository) GetFilesByStatus(ctx context.Context, enKey string, status string) ([]models.File, error) {
	query := `
	select id, typeId, pgp_sym_decrypt(name::bytea, '` + enKey + `'), extension, size, status, createdOn
	from files
	where status = $1
	`

	rows, err := repo.db.Query(ctx, query, status)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = rows.Close()
	}()
	result, err := getFilesFromRows(rows)
	if err != nil {
		return nil, err
	}
	return result, nil
}

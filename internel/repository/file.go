package repository

import (
	"context"
)

type FileRepository interface {
	SaveFileRecord(ctx context.Context, userId string, filePath string) error
}

type fileRepository struct {
	//dao dao.UserDAO
}

func (f fileRepository) SaveFileRecord(ctx context.Context, userId string, filePath string) error {
	//TODO implement me
	panic("implement me")
}

func NewFileRepository() FileRepository {
	return &fileRepository{
		//dao: dao,
	}
}

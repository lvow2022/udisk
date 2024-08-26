package service

import (
	"github.com/lvow2022/udisk/internel/domain"
	"github.com/lvow2022/udisk/internel/service/file"
)

type ValidateUploadRequest struct {
	Metadata domain.FileMetadata `json:"metadata"`
}
type ValidateUploadResponse struct {
	TaskDetails *file.TaskDetails `json:"task_details"`
}

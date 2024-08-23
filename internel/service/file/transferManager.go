package file

import (
	"github.com/lvow2022/udisk/internel/domain"
	"github.com/spf13/afero"
)

const (
	LargeFileMaxSize = 100 * 1024 * 1024 // 中等文件最大值为 500MB (500 * 1024 * 1024 字节)
)

// 文件大小类型
type FileSizeCategory int

const (
	RegularFile FileSizeCategory = iota
	LargeFile
)

type TransferManager interface {
	GenUploadTask(osFs afero.Fs, memFs afero.Fs, metadata domain.FileMetadata) (taskId string, err error)
	GenDownloadTask(osFs afero.Fs, memFs afero.Fs, metadata domain.FileMetadata) (taskId string, err error)
	Process(taskId string) (filePath string, fileName string, err error)
}

type transferManager struct {
	tc *TaskCenter
}

func (t *transferManager) GenDownloadTask(osFs afero.Fs, memFs afero.Fs, metadata domain.FileMetadata) (taskId string, err error) {
	// 根据文件大小生成对应 task
	var taskType TaskType
	if metadata.Size < LargeFileMaxSize {
		taskType = RegularFileDownload
	} else {
		taskType = LargeFileDownload
	}

	return t.tc.Gen(taskType, osFs, memFs, metadata)
}

func (t *transferManager) GenUploadTask(osFs afero.Fs, memFs afero.Fs, metadata domain.FileMetadata) (taskId string, err error) {
	// 根据文件大小生成对应 task
	var taskType TaskType
	if metadata.Size < LargeFileMaxSize {
		taskType = RegularFileUpload
	} else {
		taskType = LargeFileUpload
	}
	return t.tc.Gen(taskType, osFs, memFs, metadata)
}

func (t *transferManager) Process(taskId string) (filePath string, fileName string, err error) {
	return t.tc.Do(taskId)
}

func NewTransferManager() TransferManager {
	tc := NewTaskCenter()
	return &transferManager{
		tc: tc,
	}
}

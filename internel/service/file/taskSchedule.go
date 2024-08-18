package file

import (
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/lvow2022/udisk/internel/domain"
	"github.com/spf13/afero"
	"sync"
)

type TaskStatus int

const (
	TaskPending TaskStatus = iota
	TaskInProgress
	TaskCompleted
	TaskFailed
)

type TaskType int

const (
	SmallFileUpload TaskType = iota
	RegularFileUpload
	LargeFileUpload
	SmallFileDownload
	RegularFileDownload
	LargeFileDownload
)

type TaskSchedule interface {
	Gen(t TaskType, osFs afero.Fs, memFs afero.Fs, metadata domain.FileMetadata) (string, error)
	Get(taskId string) (*Task, error)
}

type Task struct {
	ID       string
	Status   TaskStatus
	Metadata domain.FileMetadata
	Error    error
	Type     TaskType
	osFs     afero.Fs
	memFs    afero.Fs
}

type taskSchedule struct {
	taskMap sync.Map
	taskCh  chan *Task
	wg      sync.WaitGroup
}

func NewTaskSchedule() TaskSchedule {
	ts := &taskSchedule{
		taskCh: make(chan *Task),
	}

	return ts
}
func (t *taskSchedule) Gen(taskType TaskType, osFs afero.Fs, memFs afero.Fs, metadata domain.FileMetadata) (string, error) {
	taskID, _ := generateTaskID() // 生成唯一任务 ID
	task := &Task{
		ID:       taskID,
		Status:   TaskPending,
		Metadata: metadata,
		Type:     taskType,
		osFs:     osFs,
		memFs:    memFs,
	}

	t.taskMap.Store(taskID, task)

	return taskID, nil
}

func (t *taskSchedule) Get(taskId string) (*Task, error) {
	value, ok := t.taskMap.Load(taskId)
	if !ok {
		return nil, errors.New("task not found")
	}
	return value.(*Task), nil
}

//func (t *taskSchedule) handleSmallFileUpload(task *Task) error {
//	// 处理小文件上传逻辑
//	err := afero.WriteFile(task.memFs, task.Metadata.Path, []byte(task.Metadata.MD5), 0644)
//	if err != nil {
//		return fmt.Errorf("failed to create file: %v", err)
//	}
//
//	// 将文件写入 OS 文件系统
//	err = afero.WriteFile(task.osFs, task.Metadata.Path, task.Content, 0644)
//	if err != nil {
//		return fmt.Errorf("failed to write file to OS filesystem: %v", err)
//	}
//	return nil
//}
//
//func (t *taskSchedule) handleRegularFileUpload(task *Task) error {
//	// 处理普通文件上传逻辑
//
//	// 文件不存在时：afero.WriteFile 会自动创建该文件，并写入数据。
//	// 目录不存在时：afero.WriteFile 不会创建目录，会返回一个错误。
//	// 因此，在调用 afero.WriteFile 前，确保文件路径中的所有目录已经存在，
//	// 如果目录不存在，可以先调用 afero.MkdirAll 创建目录。
//	err := afero.WriteFile(task.memFs, task.Metadata.Path, []byte(task.Metadata.MD5), 0644)
//	if err != nil {
//		return fmt.Errorf("failed to create file: %v", err)
//	}
//
//	// 将文件写入 OS 文件系统
//	err = afero.WriteFile(task.osFs, task.Metadata.Path, task.Content, 0644)
//	if err != nil {
//		return fmt.Errorf("failed to write file to OS filesystem: %v", err)
//	}
//	return nil
//}
//
//func (t *taskSchedule) handleLargeFileUpload(task *Task) {
//	// 处理大文件上传逻辑
//}
//
//func (t *taskSchedule) handleSmallFileDownload(task *Task) {
//	// 处理小文件下载逻辑
//
//}
//
//func (t *taskSchedule) handleRegularFileDownload(task *Task) {
//	// 处理普通文件下载逻辑
//
//}
//
//func (t *taskSchedule) handleLargeFileDownload(task *Task) {
//	// 处理大文件下载逻辑
//}

func generateTaskID() (string, error) {
	// 生成一个新的 UUID
	taskID, err := uuid.NewUUID()
	if err != nil {
		return "", fmt.Errorf("failed to generate task ID: %v", err)
	}
	return taskID.String(), nil
}

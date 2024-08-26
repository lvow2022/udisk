package file

import (
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/lvow2022/udisk/internel/domain"
	"github.com/spf13/afero"
	"path/filepath"
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
	RegularFileUpload TaskType = iota
	LargeFileUpload
	RegularFileDownload
	LargeFileDownload
)

type Task struct {
	Status   TaskStatus
	Metadata domain.FileMetadata
	Error    error
	//osFs    afero.Fs
	memFs   afero.Fs
	Details *TaskDetails
}

type TaskCenter struct {
	taskMap sync.Map
}

func NewTaskCenter() *TaskCenter {
	ts := &TaskCenter{}
	return ts
}

type TaskDetails struct {
	TaskId   string      `json:"task_id"`
	Type     TaskType    `json:"type"`
	ChunkNum int         `json:"chunk_num"`
	Chunks   []ChunkInfo `json:"chunks"`
}

type ChunkInfo struct {
	Index int `json:"index"`
	Start int `json:"start"`
	End   int `json:"end"`
}

func ceilDiv(a, b int) int {
	if b == 0 {
		panic("division by zero")
	}
	return (a + b - 1) / b
}

func (t *TaskCenter) Gen(taskType TaskType, osFs afero.Fs, memFs afero.Fs, metadata domain.FileMetadata) (taskDetails *TaskDetails, err error) {
	taskId, _ := generateTaskID() // 生成唯一任务 ID

	if taskType == LargeFileUpload {
		// 上传大文件，需要告知客户端文件分多少片，每一分片对应的字节范围，
		// 生成分片信息
		n := ceilDiv(metadata.Size, FileChunkSize)
		taskDetails = &TaskDetails{
			TaskId:   taskId,
			Type:     LargeFileUpload,
			ChunkNum: n,
			Chunks:   make([]ChunkInfo, n),
		}

		var start int
		var end int
		for i := 0; i < n-1; i++ {
			start = i * FileChunkSize
			end = start + FileChunkSize - 1
			taskDetails.Chunks[i] = ChunkInfo{i, start, end}
		}

		start = (n - 1) * FileChunkSize
		end = metadata.Size - 1
		taskDetails.Chunks[n-1] = ChunkInfo{n, start, end}

	} else if taskType == LargeFileDownload {
		// 大文件以目录+块文件的形式保存
		// 所有这里只需要返回文件块的路径
	}

	task := &Task{
		Status:   TaskPending,
		Metadata: metadata,
		memFs:    memFs,
		Details:  taskDetails,
	}

	t.taskMap.Store(taskId, task)
	return
}

func (t *TaskCenter) Do(taskId string) (filePath string, fileName string, err error) {
	task, err := t.get(taskId)
	if err != nil {
		return "", "", errors.New("no such task")
	}

	switch task.Details.Type {

	case RegularFileUpload:
		filePath, _, err = t.handleRegularFileUpload(task)
	case LargeFileUpload:
		filePath, _, err = t.handleLargeFileUpload(task)

	case RegularFileDownload:
		filePath, fileName, err = t.handleRegularFileDownload(task)
	case LargeFileDownload:
		filePath, fileName, err = t.handleLargeFileDownload(task)

	default:

	}
	return filePath, fileName, err
}

func (t *TaskCenter) get(taskId string) (*Task, error) {
	value, ok := t.taskMap.Load(taskId)
	if !ok {
		return nil, errors.New("task not found")
	}
	return value.(*Task), nil
}

// 1.	afero.Fs
// afero.Fs 是 afero 中最核心的接口，它定义了操作文件系统的基本方法。它包括以下方法：
//   - Create(name string) (afero.File, error)
//   - 创建一个新文件。如果文件已经存在，它会被截断为 0 字节。
//   - Open(name string) (afero.File, error)
//   - 打开一个现有文件。
//   - OpenFile(name string, flag int, perm os.FileMode) (afero.File, error)
//   - 以指定的标志和权限打开一个文件。
//   - Remove(name string) error
//   - 删除一个文件。
//   - Rename(oldname, newname string) error
//   - 重命名文件。
//   - Mkdir(name string, perm os.FileMode) error
//   - 创建一个目录。
//   - MkdirAll(path string, perm os.FileMode) error
//   - 递归创建目录。
//   - Stat(name string) (os.FileInfo, error)
//   - 获取文件或目录的状态信息。
//   - Chmod(name string, mode os.FileMode) error
//   - 更改文件或目录的权限。
//   - Chtimes(name string, atime time.Time, mtime time.Time) error
//   - 更改文件的访问时间和修改时间。
//   - ReadDir(dirname string) ([]os.DirEntry, error)
//   - 读取目录中的条目。
//     2.	afero.File
//
// afero.File 是对文件的抽象，定义了文件操作的方法，包括：
//   - Stat() (os.FileInfo, error)
//   - 获取文件的状态信息。
//   - Close() error
//   - 关闭文件。
//   - Read(p []byte) (int, error)
//   - 从文件中读取数据。
//   - ReadAt(p []byte, off int64) (int, error)
//   - 从文件的指定位置读取数据。
//   - Write(p []byte) (int, error)
//   - 向文件中写入数据。
//   - WriteAt(p []byte, off int64) (int, error)
//   - 向文件的指定位置写入数据。
//   - Seek(offset int64, whence int) (int64, error)
//   - 更改文件的当前读取/写入位置。
//   - Truncate(size int64) error
//   - 将文件的长度截断为指定大小。
//
// 文件不存在时：afero.WriteFile 会自动创建该文件，并写入数据。
// 目录不存在时：afero.WriteFile 不会创建目录，会返回一个错误。
// 因此，在调用 afero.WriteFile 前，确保文件路径中的所有目录已经存在，
// 如果目录不存在，可以先调用 afero.MkdirAll 创建目录。

func (t *TaskCenter) handleRegularFileUpload(task *Task) (filePath string, fileName string, err error) {
	// 将文件添加到 memfs,这里文件内容记录的是 md5
	err = afero.WriteFile(task.memFs, task.Metadata.Path, []byte(task.Metadata.MD5), 0644)
	// 实际的路径
	filePath = filepath.Join("./tmp", task.Metadata.MD5)
	return filePath, "", err
}

func (t *TaskCenter) handleLargeFileUpload(task *Task) (filePath string, fileName string, err error) {
	// 处理大文件上传逻辑
	return "", "", nil
}

func (t *TaskCenter) handleRegularFileDownload(task *Task) (filePath string, fileName string, err error) {
	md5Byte, err := afero.ReadFile(task.memFs, task.Metadata.Path)
	// 实际的路径
	filePath = filepath.Join("./tmp", string(md5Byte))
	return filePath, "", err
}

func (t *TaskCenter) handleLargeFileDownload(task *Task) (filePath string, fileName string, err error) {
	// 处理大文件下载逻辑
	return "", "", nil
}

func generateTaskID() (string, error) {
	// 生成一个新的 UUID
	taskID, err := uuid.NewUUID()
	if err != nil {
		return "", fmt.Errorf("failed to generate task ID: %v", err)
	}
	return taskID.String(), nil
}

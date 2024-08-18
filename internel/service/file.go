package service

import (
	"context"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/lvow2022/udisk/internel/domain"
	"github.com/lvow2022/udisk/internel/repository"
	"github.com/lvow2022/udisk/internel/service/file"
	"os"
	"sync"

	"github.com/spf13/afero"
)

type FileService interface {
	Upload(ctx context.Context, userId string, taskId string, content []byte) error
	Download(ctx context.Context, userId string, path string) (string, string, error)
	ListDirectory(ctx context.Context, userId string, path string) ([]os.FileInfo, error)
	MakeDirectory(ctx context.Context, path string) error
	FileStat(ctx context.Context, path string) (os.FileInfo, error)
	ValidateUpload(ctx context.Context, userId string, metadata domain.FileMetadata) (string, error)
	ValidateDownload(ctx context.Context, userId string, metadata domain.FileMetadata) (string, error)
	AddUser(ctx context.Context, userId string) error
}

type fileService struct {
	mu       sync.RWMutex
	memFsMap sync.Map          // 每个用户拥有一个独立的内存文件系统
	osFs     afero.Fs          // 所有用户共享 os 文件系统
	ts       file.TaskSchedule // 文件任务调度中心,控制文件上传下载，实现并发能力
	repo     repository.FileRepository
}

func (f *fileService) MakeDirectory(ctx context.Context, path string) error {
	//TODO implement me
	panic("implement me")
}

// AddUser 添加新用户，分配内存文件系统
func (f *fileService) AddUser(ctx context.Context, userId string) error {
	// 检查用户是否已经存在
	_, ok := f.memFsMap.Load(userId)
	if ok {
		return fmt.Errorf("user %s already exists", userId)
	}

	// 创建新的内存文件系统并存储到用户映射中
	f.memFsMap.Store(userId, afero.NewMemMapFs())
	return nil
}

// Validate 上传文件前需要验证文件的元数据，生成任务并返回任务 ID
func (f *fileService) ValidateUpload(ctx context.Context, userId string, metadata domain.FileMetadata) (string, error) {
	// 这里可以加入元数据验证逻辑
	// 验证是否存在同名文件
	memFs, err := f.getMemFs(userId)
	if err != nil {
		return "", err
	}
	info, err := memFs.Stat(metadata.Path)
	if info != nil {
		return "", errors.New("file exists")
	}
	// 根据文件大小生成对应 task
	category := GetFileSizeCategory(metadata.Size)
	var taskType file.TaskType
	switch category {
	case SmallFile:
		taskType = file.SmallFileUpload
	case MediumFile:
		taskType = file.RegularFileUpload
	case LargeFile:
		taskType = file.LargeFileUpload
	}
	taskID, err := f.ts.Gen(taskType, f.osFs, memFs, metadata)
	if err != nil {
		return "", fmt.Errorf("failed to generate task: %v", err)
	}

	// 返回生成的任务 ID
	return taskID, nil
}

func (f *fileService) ValidateDownload(ctx context.Context, userId string, metadata domain.FileMetadata) (string, error) {
	// 这里可以加入元数据验证逻辑
	// 验证是否存在同名文件
	memFs, err := f.getMemFs(userId)
	if err != nil {
		return "", err
	}
	info, err := memFs.Stat(metadata.Path)
	if err != nil {
		return "", errors.New("no such file ")
	}
	// 根据文件大小生成对应 task
	category := GetFileSizeCategory(info.Size())
	var taskType file.TaskType
	switch category {
	case SmallFile:
		taskType = file.SmallFileDownload
	case MediumFile:
		taskType = file.RegularFileDownload
	case LargeFile:
		taskType = file.LargeFileDownload
	}
	taskID, err := f.ts.Gen(taskType, f.osFs, memFs, metadata)
	if err != nil {
		return "", fmt.Errorf("failed to generate task: %v", err)
	}

	// 返回生成的任务 ID
	return taskID, nil
}

// Upload 上传文件并持久化到 OS 文件系统
func (f *fileService) Upload(ctx context.Context, userId string, taskId string, content []byte) error {
	task, err := f.ts.Get(taskId)
	if err != nil {
		return err
	}
	memFs, err := f.getMemFs(userId)
	if err != nil {
		return err
	}

	switch task.Type {
	case file.SmallFileUpload:
		err = f.handleSmallFileUpload(memFs, task, content)
	case file.RegularFileUpload:
		err = f.handleRegularFileUpload(memFs, task, content)
	case file.LargeFileUpload:
		err = f.handleLargeFileUpload(memFs, task, content)

	default:
		task.Status = file.TaskFailed
		task.Error = fmt.Errorf("unknown task type")
	}

	// 处理完成后更新任务状态
	if task.Error == nil {
		task.Status = file.TaskCompleted
	} else {
		task.Status = file.TaskFailed
	}
	return nil
}

// Download 从 OS 文件系统下载文件
func (f *fileService) Download(ctx context.Context, userId string, taskId string) (string, string, error) {
	task, err := f.ts.Get(taskId)
	if err != nil {
		fmt.Println("no such task")
		return "", "", nil
	}
	memFs, err := f.getMemFs(userId)
	if err != nil {
		fmt.Println("no such task")
		return "", "", nil
	}

	var filePath string
	var fileName string

	switch task.Type {
	case file.SmallFileDownload:
		fileName, filePath, err = f.handleSmallFileDownload(memFs, task)
	case file.RegularFileDownload:
		fileName, filePath, err = f.handleRegularFileDownload(memFs, task)
	case file.LargeFileDownload:
		fileName, filePath, err = f.handleLargeFileDownload(memFs, task)
	default:
		task.Status = file.TaskFailed
		task.Error = fmt.Errorf("unknown task type")
		return "", "", task.Error // 返回错误时避免空变量的使用
	}

	return fileName, filePath, err

}

// ListDirectory 列出目录内容
func (f *fileService) ListDirectory(ctx context.Context, userId string, path string) ([]os.FileInfo, error) {
	// 从用户的内存文件系统中获取 md5
	memFs, err := f.getMemFs(userId)
	if err != nil {
		return nil, err
	}
	files, err := afero.ReadDir(memFs, path)
	if err != nil {
		return nil, fmt.Errorf("failed to list directory: %v", err)
	}

	for _, v := range files {
		fmt.Println("File:", v.Name())
	}

	return files, nil
}

// FileStat 获取文件状态
func (f *fileService) FileStat(ctx context.Context, path string) (os.FileInfo, error) {
	info, err := f.osFs.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("failed to get file status: %v", err)
	}

	fmt.Println("File info:", info.Name(), info.Size(), info.ModTime())
	return info, nil
}

// NewFileService 创建新的文件服务
func NewFileService(ts file.TaskSchedule, repo repository.FileRepository) FileService {
	// 使用 os 文件系统作为基础文件系统
	baseFs := afero.NewOsFs()
	return &fileService{
		osFs: afero.NewBasePathFs(baseFs, "./tmp"),
		ts:   ts,
		repo: repo,
	}
}

// getMemFs 获取用户的内存文件系统
func (f *fileService) getMemFs(userId string) (afero.Fs, error) {
	value, ok := f.memFsMap.Load(userId)
	if !ok {
		return nil, fmt.Errorf("memory filesystem for user %s not found", userId)
	}
	return value.(afero.Fs), nil
}

const (
	SmallFileMaxSize  = 100 * 1024        // 小文件最大值为 100KB (100 * 1024 字节)
	MediumFileMaxSize = 500 * 1024 * 1024 // 中等文件最大值为 500MB (500 * 1024 * 1024 字节)
)

// 文件大小类型
type FileSizeCategory int

const (
	SmallFile FileSizeCategory = iota
	MediumFile
	LargeFile
)

// 根据文件大小获取文件类别
func GetFileSizeCategory(size int64) FileSizeCategory {
	switch {
	case size <= SmallFileMaxSize:
		return SmallFile
	case size <= MediumFileMaxSize:
		return MediumFile
	default:
		return LargeFile
	}
}

func (f *fileService) handleSmallFileUpload(memFs afero.Fs, task *file.Task, content []byte) error {
	// 处理小文件上传逻辑
	// 在 memFs 中记录 md5
	err := afero.WriteFile(memFs, task.Metadata.Path, []byte(task.Metadata.MD5), 0644)
	if err != nil {
		return fmt.Errorf("failed to create file: %v", err)
	}

	// 在 osFs 中写入文件内容
	err = afero.WriteFile(f.osFs, task.Metadata.MD5, content, 0644)
	if err != nil {
		return fmt.Errorf("failed to write file to OS filesystem: %v", err)
	}
	return nil
}

func (f *fileService) handleRegularFileUpload(memFs afero.Fs, task *file.Task, content []byte) error {
	// 处理普通文件上传逻辑

	// 文件不存在时：afero.WriteFile 会自动创建该文件，并写入数据。
	// 目录不存在时：afero.WriteFile 不会创建目录，会返回一个错误。
	// 因此，在调用 afero.WriteFile 前，确保文件路径中的所有目录已经存在，
	// 如果目录不存在，可以先调用 afero.MkdirAll 创建目录。
	// 在 memFs 中创建文件，文件路径为上传路径，文件内容为 md5

	// todo  md5校验
	err := afero.WriteFile(memFs, task.Metadata.Path, []byte(task.Metadata.MD5), 0644)
	if err != nil {
		return fmt.Errorf("failed to create file: %v", err)
	}

	// 在 osFs 中创建文件，文件路径为固定路径，文件名为 md5
	err = afero.WriteFile(f.osFs, task.Metadata.MD5, content, 0644)
	if err != nil {
		return fmt.Errorf("failed to write file to OS filesystem: %v", err)
	}
	return nil
}

func (f *fileService) handleLargeFileUpload(memFs afero.Fs, task *file.Task, content []byte) error {
	// 处理大文件上传逻辑
	return nil
}

func (f *fileService) handleSmallFileDownload(memFs afero.Fs, task *file.Task) (string, string, error) {
	// 处理小文件下载逻辑
	// 从用户的内存文件系统中获取 md5
	md5Byte, _ := afero.ReadFile(memFs, task.Metadata.Path)
	fileName := task.Metadata.Path
	filePath := "./tmp/" + string(md5Byte)
	fmt.Println("File Path:", filePath)
	return fileName, filePath, nil
}

func (t *fileService) handleRegularFileDownload(memFs afero.Fs, task *file.Task) (string, string, error) {
	// 处理普通文件下载逻辑
	return "", "", nil
}

func (t *fileService) handleLargeFileDownload(memFs afero.Fs, task *file.Task) (string, string, error) {
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

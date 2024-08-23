package service

import (
	"context"
	"errors"
	"fmt"
	"github.com/lvow2022/udisk/internel/domain"
	"github.com/lvow2022/udisk/internel/repository"
	"github.com/lvow2022/udisk/internel/service/file"
	"os"
	"sync"

	"github.com/spf13/afero"
)

type FileService interface {
	Upload(ctx context.Context, taskId string) (filePath string, err error)
	Download(ctx context.Context, taskId string) (filePath string, fileName string, err error)
	ListDirectory(ctx context.Context, userId string, path string) ([]os.FileInfo, error)
	MakeDirectory(ctx context.Context, path string) error
	FileStat(ctx context.Context, path string) (os.FileInfo, error)
	ValidateUpload(ctx context.Context, userId string, metadata domain.FileMetadata) (string, error)
	ValidateDownload(ctx context.Context, userId string, metadata domain.FileMetadata) (string, error)
	AddUser(ctx context.Context, userId string) error
}

type fileService struct {
	mu       sync.RWMutex
	memFsMap sync.Map             // 每个用户拥有一个独立的内存文件系统
	osFs     afero.Fs             // 所有用户共享 os 文件系统
	tm       file.TransferManager // 文件任务调度中心,控制文件上传下载，实现并发能力
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

	taskId, err := f.tm.GenUploadTask(f.osFs, memFs, metadata)
	if err != nil {
		return "", fmt.Errorf("failed to generate task: %v", err)
	}

	// 返回生成的任务 ID
	return taskId, nil
}

func (f *fileService) ValidateDownload(ctx context.Context, userId string, metadata domain.FileMetadata) (string, error) {
	// 这里可以加入元数据验证逻辑
	// 验证是否存在同名文件
	memFs, err := f.getMemFs(userId)
	if err != nil {
		return "", err
	}
	_, err = memFs.Stat(metadata.Path)
	if err != nil {
		return "", errors.New("no such file ")
	}

	taskId, err := f.tm.GenDownloadTask(f.osFs, memFs, metadata)
	if err != nil {
		return "", fmt.Errorf("failed to generate task: %v", err)
	}

	// 返回生成的任务 ID
	return taskId, nil
}

// Upload 上传文件并持久化到 OS 文件系统
func (f *fileService) Upload(ctx context.Context, taskId string) (filePath string, err error) {
	filePath, _, err = f.tm.Process(taskId)
	if err != nil {
		return "", err
	}
	return filePath, nil
}

// Download 从 OS 文件系统下载文件
func (f *fileService) Download(ctx context.Context, taskId string) (filePath string, fileName string, err error) {
	filePath, fileName, err = f.tm.Process(taskId)
	if err != nil {
		return "", "", err
	}
	return filePath, fileName, err
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
func NewFileService(tm file.TransferManager, repo repository.FileRepository) FileService {
	// 使用 os 文件系统作为基础文件系统
	baseFs := afero.NewOsFs()
	return &fileService{
		osFs: afero.NewBasePathFs(baseFs, "./tmp"),
		tm:   tm,
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

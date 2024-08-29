package service

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/lvow2022/udisk/internel/repository"
	"io"
	"os"
	"path/filepath"
	"sync"

	"github.com/spf13/afero"
)

const ChunkSize = 5 * 1024 * 1024

type FileService interface {
	Upload(ctx *gin.Context, chunkIndex int, chunkMd5, fileMd5 string) error
	Download(ctx *gin.Context, filePath, chunkIndex string) (path string, err error)
	CompleteUpload(ctx *gin.Context, fileMd5 string, totalChunks int) error
	ListDirectory(ctx context.Context, userId string, path string) ([]os.FileInfo, error)
	MakeDirectory(ctx context.Context, path string) error
	FileStat(ctx context.Context, path string) (os.FileInfo, error)
	ValidateDownload(ctx *gin.Context, userId string, src, dst string) (md5 string, chunkCount int, err error)
	ValidateUpload(ctx context.Context, userId string, src, dst string) (chunkSize int, err error)
	AddUser(ctx context.Context, userId string) error
}

type fileService struct {
	mu       sync.RWMutex
	memFsMap sync.Map // 每个用户拥有一个独立的内存文件系统
	repo     repository.FileRepository
	osFs     afero.Fs
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

func (f *fileService) ValidateDownload(ctx *gin.Context, userId string, src, dst string) (md5 string, chunkCount int, err error) {
	// 检查 src 是否存在
	md5, err = f.CheckIfFileExists(userId, src)
	if err != nil {
		// 处理错误
		return "", 0, err
	}

	// 查找真实路径
	osPath := filepath.Join("./tmp", md5)
	chunkCount, err = f.countFilesInDirectory(osPath)
	if err != nil {
		return "", 0, err
	}
	// 不存在同名文件
	return md5, chunkCount, nil
}

func (f *fileService) ValidateUpload(ctx context.Context, userId string, src, dst string) (chunkSize int, err error) {
	//path := filepath.Join(dst, filepath.Base(src))
	//md5, err := f.CheckIfFileExists(userId, path)
	//if md5 != "" {
	//	return 0, fmt.Errorf("存在同名文件")
	//}

	return ChunkSize, err

}

// Upload 上传文件并持久化到 OS 文件系统
func (f *fileService) Upload(ctx *gin.Context, chunkIndex int, chunkMd5, fileMd5 string) error {
	filePath := fmt.Sprintf("./tmp/%s/%d", fileMd5, chunkIndex)

	// 创建目录
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		fmt.Println("Failed to create directory:", err)
		return err
	}

	// 创建文件
	file, err := os.Create(filePath)
	if err != nil {
		fmt.Println("Failed to create file:", err)
		return err
	}
	defer file.Close()

	// 创建 MD5 哈希计算器
	hash := md5.New()

	// 将请求体中的数据复制到文件中，同时计算 MD5
	writer := io.MultiWriter(file, hash)
	if _, err := io.Copy(writer, ctx.Request.Body); err != nil {
		fmt.Println("Failed to copy data:", err)
		return err
	}

	// 计算最终的 MD5 哈希值
	calculatedMd5 := hex.EncodeToString(hash.Sum(nil))

	// 比较计算的 MD5 值与传入的 MD5 值
	if calculatedMd5 != chunkMd5 {
		// 如果 MD5 校验失败，删除刚刚写入的文件
		if err := os.Remove(filePath); err != nil {
			fmt.Println("Failed to remove file:", err)
		}
		return fmt.Errorf("MD5 mismatch: calculated %s, expected %s", calculatedMd5, chunkMd5)
	}

	return nil
}
func (f *fileService) CompleteUpload(ctx *gin.Context, fileMd5 string, totalChunks int) error {
	// 指定存储分片文件的目录和合并后的文件路径
	directory := fmt.Sprintf("./tmp/%s", fileMd5)
	outputFile := fmt.Sprintf("./all/%s", fileMd5)

	// 合并分片
	if err := mergeChunks(directory, outputFile, totalChunks); err != nil {
		fmt.Println("Failed to merge chunks:", err)
		return err
	}

	fmt.Println("File merge completed successfully:", outputFile)
	return nil
}

// Download 从 OS 文件系统下载文件
func (f *fileService) Download(ctx *gin.Context, filePath, chunkIndex string) (path string, err error) {

	// 这里从 user memfs 从获取文件路径
	path = fmt.Sprintf("./tmp/12312321b3jh12gf321td312hd3j12h/part%s", chunkIndex)
	return path, err
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
func NewFileService(repo repository.FileRepository) FileService {
	// 使用 os 文件系统作为基础文件系统
	baseFs := afero.NewOsFs()
	return &fileService{
		osFs: afero.NewBasePathFs(baseFs, "./tmp"),
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

// CheckIfFileExists 检查用户目录下是否存在指定路径的文件,如果存在返回文件 md5
func (f *fileService) CheckIfFileExists(userId string, path string) (md5 string, err error) {
	// 验证 src 路径并获取 md5
	memFs, err := f.getMemFs(userId)
	if err != nil {
		return "", err
	}

	file, err := memFs.OpenFile(path, os.O_RDONLY, os.ModePerm)
	if err != nil {
		return "", err
	}
	defer file.Close()

	bytes, err := io.ReadAll(file)
	if err != nil {
		return "", err
	}

	return string(bytes), nil
}

func (f *fileService) countFilesInDirectory(directory string) (int, error) {
	// 打开目录
	dir, err := os.Open(directory)
	if err != nil {
		return 0, err
	}
	defer dir.Close()

	// 读取目录中的所有文件和子目录
	files, err := dir.Readdir(-1)
	if err != nil {
		return 0, err
	}

	// 初始化文件计数器
	fileCount := 0

	// 遍历目录项
	for _, fileInfo := range files {
		if !fileInfo.IsDir() {
			// 如果是文件，计数加一
			fileCount++
		}
	}

	return fileCount, nil
}

func mergeChunks(directory, outputFile string, totalChunks int) error {
	// 创建目标文件
	outFile, err := os.Create(outputFile)
	if err != nil {
		return fmt.Errorf("Failed to create output file: %v", err)
	}
	defer outFile.Close()

	// 逐个读取分片文件并写入目标文件
	for i := 0; i < totalChunks; i++ {
		chunkFilePath := fmt.Sprintf("%s/%d", directory, i)
		chunkFile, err := os.Open(chunkFilePath)
		if err != nil {
			return fmt.Errorf("Failed to open chunk file %s: %v", chunkFilePath, err)
		}

		// 将分片文件的内容复制到目标文件中
		if _, err := io.Copy(outFile, chunkFile); err != nil {
			chunkFile.Close()
			return fmt.Errorf("Failed to copy chunk file %s: %v", chunkFilePath, err)
		}
		chunkFile.Close()
	}

	return nil
}

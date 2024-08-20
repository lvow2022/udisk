package web

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/lvow2022/udisk/internel/domain"
	"github.com/lvow2022/udisk/internel/pkg/code"
	"github.com/lvow2022/udisk/internel/service"
	"github.com/lvow2022/udisk/pkg/ginx"
	"github.com/lvow2022/udisk/pkg/ginx/errors"
	"io/ioutil"
	"net/http"
	"os"
)

type FileHandler struct {
	fileSvc service.FileService
}

func NewFileHandler(fileSvc service.FileService) *FileHandler {
	return &FileHandler{
		fileSvc: fileSvc,
	}
}

func (h *FileHandler) RegisterRoutes(server *gin.Engine) {
	g := server.Group("/file")
	g.POST("/validate/upload", h.ValidateUpload)
	g.POST("/validate/download", h.ValidateDownload)
	g.POST("/upload", h.Upload)
	g.POST("/download", h.Download)
	g.POST("/adduser", h.AddUser)
}

type ValidateUploadRequest struct {
	domain.FileMetadata
}

func (h *FileHandler) ValidateUpload(ctx *gin.Context) {
	var req ValidateUploadRequest
	if err := ctx.Bind(&req); err != nil {
		ginx.WriteResponse(ctx, errors.WithCode(code.ErrBind, err.Error()), nil)
		return
	}

	metadata := domain.FileMetadata{
		Name:    req.Name,
		Size:    req.Size,
		Type:    req.Type,
		OwnerID: req.OwnerID,
		MD5:     req.MD5,
		Path:    req.Path,
	}
	taskId, err := h.fileSvc.ValidateUpload(ctx, "123", metadata)
	if err != nil {
		ginx.WriteResponse(ctx, err, nil)
	}

	type TaskDetails struct {
		TaskID string `json:"task_id"`
		//FileName string `json:"fileName"`
		//Status   string `json:"status"`
	}

	ginx.WriteResponse(ctx, nil, TaskDetails{taskId})
	return
}

type ValidateDownloadRequest struct {
}

func (h *FileHandler) ValidateDownload(ctx *gin.Context) {
	type request struct {
		domain.FileMetadata
	}

	var req request
	if err := ctx.Bind(&req); err != nil {
		return
	}

	metadata := domain.FileMetadata{
		Name:    req.Name,
		Size:    req.Size,
		Type:    req.Type,
		OwnerID: req.OwnerID,
		MD5:     req.MD5,
		Path:    req.Path,
	}
	taskId, err := h.fileSvc.ValidateDownload(ctx, "123", metadata)
	if err != nil {
		ginx.WriteResponse(ctx, err, nil)
		return
	}

	type TaskDetails struct {
		TaskID string `json:"task_id"`
		//FileName string `json:"fileName"`
		//Status   string `json:"status"`
	}

	ginx.WriteResponse(ctx, nil, TaskDetails{taskId})

	return
}

func (h *FileHandler) Upload(ctx *gin.Context) {
	file, err := ctx.FormFile("file")
	if err != nil {
		ctx.String(http.StatusBadRequest, fmt.Sprintf("Failed to get file: %s", err.Error()))
		return
	}

	// 打开上传的文件
	openedFile, err := file.Open()
	if err != nil {
		ctx.String(http.StatusInternalServerError, fmt.Sprintf("Failed to open file: %s", err.Error()))
		return
	}
	defer openedFile.Close()

	// 读取文件内容
	content, err := ioutil.ReadAll(openedFile)
	if err != nil {
		ctx.String(http.StatusInternalServerError, fmt.Sprintf("Failed to read file content: %s", err.Error()))
		return
	}
	// 打印文件内容，或者处理文件内容
	fmt.Println("File content:", string(content))
	userId := ctx.PostForm("user_id")
	taskId := ctx.PostForm("task_id")
	err = h.fileSvc.Upload(ctx, userId, taskId, content)
	ginx.WriteResponse(ctx, err, nil)
	return
}

func (h *FileHandler) Download(ctx *gin.Context) {
	type request struct {
		UserId string `json:"user_id"`
		TaskId string `json:"task_id"`
	}
	var req request
	if err := ctx.Bind(&req); err != nil {
		return
	}

	fileName, filePath, err := h.fileSvc.Download(ctx, req.UserId, req.TaskId)
	// 打开文件
	file, err := os.Open(filePath)
	if err != nil {
		ctx.String(http.StatusNotFound, fmt.Sprintf("File not found: %s", err.Error()))
		return
	}
	defer file.Close()

	// Content-Disposition 决定了下载行为和文件名，而 Content-Type 确保文件内容作为二进制文件处理，不被浏览器尝试解析。
	ctx.Header("Content-Disposition", "attachment; filename="+fileName)
	ctx.Header("Content-Type", "application/octet-stream")
	// 获取文件的修改时间
	fileInfo, err := file.Stat()
	if err != nil {
		ctx.String(http.StatusInternalServerError, fmt.Sprintf("Failed to get file info: %s", err.Error()))
		return
	}
	// 读取文件并发送到客户端
	http.ServeContent(ctx.Writer, ctx.Request, fileName, fileInfo.ModTime(), file)
}

func (h *FileHandler) AddUser(ctx *gin.Context) {
	type request struct {
		UserId string `json:"user_id"`
	}
	var req request
	if err := ctx.Bind(&req); err != nil {
		return
	}

	err := h.fileSvc.AddUser(ctx, req.UserId)
	if err != nil {
		ginx.WriteResponse(ctx, err, nil)
		return
	}

	ginx.WriteResponse(ctx, nil, nil)

	return
}

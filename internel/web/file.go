package web

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/lvow2022/udisk/internel/domain"
	"github.com/lvow2022/udisk/internel/pkg/code"
	"github.com/lvow2022/udisk/internel/service"
	"github.com/lvow2022/udisk/pkg/ginx"
	"github.com/lvow2022/udisk/pkg/ginx/errors"
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
	g.GET("/download", h.Download)
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
	taskId := ctx.PostForm("task_id")
	if err != nil || taskId == "" {
		ctx.String(http.StatusBadRequest, fmt.Sprintf("获取上传文件失败: %s", err.Error()))
		return
	}

	filePath, err := h.fileSvc.Upload(ctx, taskId)
	if err != nil {
		ctx.String(http.StatusOK, fmt.Sprintf("获取上传文件失败: %s", err.Error()))
		return
	}
	if err = ctx.SaveUploadedFile(file, filePath); err != nil {
		ctx.String(http.StatusInternalServerError, "保存文件失败: %s", err.Error())
	}
	ginx.WriteResponse(ctx, err, nil)
	return
}

func (h *FileHandler) Download(ctx *gin.Context) {
	taskId := ctx.Query("task_id")
	if taskId == "" {
		ctx.String(http.StatusBadRequest, "缺少 query param")
		return
	}
	filePath, fileName, err := h.fileSvc.Download(ctx, taskId)
	// 打开文件
	file, err := os.Open(filePath)
	if err != nil {
		ctx.String(http.StatusNotFound, fmt.Sprintf("File not found: %s", err.Error()))
		return
	}
	defer file.Close()

	// Content-Disposition 决定了下载行为和文件名，而 Content-Type 确保文件内容作为二进制文件处理，不被浏览器尝试解析。
	ctx.Header("Content-Disposition", "attachment; filename=example.txt")
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

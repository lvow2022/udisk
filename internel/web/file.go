package web

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/lvow2022/udisk/internel/service"
	"github.com/lvow2022/udisk/pkg/ginx"
	"github.com/lvow2022/udisk/pkg/log"
	"net/http"
	"strconv"
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
	g.POST("/complete", h.Complete)
}

// download from remote_src to local_dst
func (h *FileHandler) ValidateDownload(ctx *gin.Context) {
	src := ctx.Query("src")
	dst := ctx.Query("dst")

	md5, chunkCount, err := h.fileSvc.ValidateDownload(ctx, "123", src, dst)
	if err != nil {
		ginx.WriteResponse(ctx, err, nil)
		return // 确保在错误时返回
	}
	// Go 会自动解引用指针，因此可以直接传递结构体
	ginx.WriteResponse(ctx, nil, gin.H{
		"md5":        md5,
		"chunkCount": chunkCount,
	})
}

// upload from local_src to remote_dst
func (h *FileHandler) ValidateUpload(ctx *gin.Context) {
	src := ctx.Query("src")
	dst := ctx.Query("dst")

	chunkSize, err := h.fileSvc.ValidateUpload(ctx, "123", src, dst)

	type Response struct {
		ChunkSize int `json:"chunk_size"`
	}
	response := Response{
		ChunkSize: chunkSize, // Example chunk size
	}
	ginx.WriteResponse(ctx, err, response)
}

func (h *FileHandler) Upload(ctx *gin.Context) {
	chunkIndex := ctx.GetHeader("Chunk-Index")
	ChunkMd5 := ctx.GetHeader("Chunk-Md5")
	FileMd5 := ctx.GetHeader("File-Md5")

	index, err := strconv.Atoi(chunkIndex)

	err = h.fileSvc.Upload(ctx, index, ChunkMd5, FileMd5)
	if err != nil {
		ctx.String(http.StatusOK, fmt.Sprintf("获取上传文件失败: %s", err.Error()))
		return
	}

	ginx.WriteResponse(ctx, err, nil)
	return
}

func (h *FileHandler) Complete(ctx *gin.Context) {
	fileMd5 := ctx.Query("file_md5")
	chunk_num := ctx.Query("chunk_num")
	totalChunks, err := strconv.Atoi(chunk_num)
	if err != nil {
		log.Logger.Debug("param chunk_num atoi fail")
	}
	err = h.fileSvc.CompleteUpload(ctx, fileMd5, totalChunks)
	if err != nil {
		ctx.String(http.StatusOK, fmt.Sprintf("文件合并失败: %s", err.Error()))
		return
	}

	ginx.WriteResponse(ctx, err, nil)
	return
}

func (h *FileHandler) Download(ctx *gin.Context) {
	filePath := ctx.GetHeader("File-Path")
	chunkIndex := ctx.GetHeader("Chunk-Index")
	path, err := h.fileSvc.Download(ctx, filePath, chunkIndex)
	if err != nil {
	}
	ctx.File(path)
	//ginx.WriteResponse(ctx, nil, nil)
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

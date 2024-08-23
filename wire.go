//go:build wireinject

package main

import (
	"github.com/gin-gonic/gin"
	"github.com/google/wire"
	"github.com/lvow2022/udisk/internel/repository"
	"github.com/lvow2022/udisk/internel/repository/dao"
	"github.com/lvow2022/udisk/internel/service"
	"github.com/lvow2022/udisk/internel/service/file"
	"github.com/lvow2022/udisk/internel/web"
	"github.com/lvow2022/udisk/ioc"
)

func InitWebServer() *gin.Engine {
	wire.Build(
		// 第三方依赖
		ioc.InitDB,

		// dao
		dao.NewUserDAO,

		// repo
		repository.NewUserRepository,
		repository.NewFileRepository,

		file.NewTransferManager,
		// service
		service.NewUserService,
		service.NewFileService,

		// controller
		web.NewUserHandler,
		web.NewFileHandler,

		// app
		ioc.InitGinMiddlewares,
		ioc.InitWebServer,
	)
	return nil
}

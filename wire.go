//go:build wireinject

package main

import (
	"github.com/gin-gonic/gin"
	"github.com/google/wire"
	"github.com/lvow2022/udisk/internel/repository"
	"github.com/lvow2022/udisk/internel/repository/dao"
	"github.com/lvow2022/udisk/internel/service"
	"github.com/lvow2022/udisk/internel/web"
	ijwt "github.com/lvow2022/udisk/internel/web/jwt"
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

		// service
		service.NewUserService,
		service.NewFileService,

		// controller
		web.NewUserHandler,
		web.NewFileHandler,

		// app
		ijwt.NewLocalJWTHandler,
		ioc.InitGinMiddlewares,
		ioc.InitWebServer,
	)
	return nil
}

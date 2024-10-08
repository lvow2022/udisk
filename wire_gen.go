// Code generated by Wire. DO NOT EDIT.

//go:generate go run -mod=mod github.com/google/wire/cmd/wire
//go:build !wireinject
// +build !wireinject

package main

import (
	"github.com/gin-gonic/gin"
	"github.com/lvow2022/udisk/internel/repository"
	"github.com/lvow2022/udisk/internel/repository/dao"
	"github.com/lvow2022/udisk/internel/service"
	"github.com/lvow2022/udisk/internel/web"
	"github.com/lvow2022/udisk/ioc"
)

// Injectors from wire.go:

func InitWebServer() *gin.Engine {
	v := ioc.InitGinMiddlewares()
	db := ioc.InitDB()
	userDAO := dao.NewUserDAO(db)
	userRepository := repository.NewUserRepository(userDAO)
	userService := service.NewUserService(userRepository)
	userHandler := web.NewUserHandler(userService)
	fileRepository := repository.NewFileRepository()
	fileService := service.NewFileService(fileRepository)
	fileHandler := web.NewFileHandler(fileService)
	engine := ioc.InitWebServer(v, userHandler, fileHandler)
	return engine
}

package main

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

func main() {
	// 创建一个默认的 Gin 路由引擎
	r := gin.Default()

	// 定义一个 GET 请求的路由
	r.GET("/", func(c *gin.Context) {
		c.String(http.StatusOK, "Hello, World!")
	})

	// 定义一个 POST 请求的路由
	r.POST("/post", func(c *gin.Context) {
		name := c.PostForm("name")
		age := c.DefaultPostForm("age", "unknown")
		c.JSON(http.StatusOK, gin.H{
			"name": name,
			"age":  age,
		})
	})

	// 启动服务器，监听 8080 端口
	r.Run(":8080")
}

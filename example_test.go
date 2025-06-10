package graceful_test

import (
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lgyong511/graceful"
)

func TestGracefulServer(t *testing.T) {
	// 初始化配置
	config := graceful.Config{
		Addr:         ":8080", // 监听所有接口的8080端口
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  30 * time.Second,
	}

	// 创建Gin控制器
	gc := graceful.New(config)

	// 创建Gin引擎
	router := gin.Default()
	router.GET("/", func(c *gin.Context) {
		c.String(200, "Hello, World!")
	})

	// 启动服务
	gc.Start(router)

	// 模拟运行一段时间后重启
	time.Sleep(10 * time.Second)

	// 重启服务并修改端口为8081
	newConfig := config
	newConfig.Addr = ":8081"
	gc.Restart(&newConfig, nil) // 保持原有引擎，只修改配置

	// 让服务继续运行
	select {}
}

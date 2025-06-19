package graceful_test

import (
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lgyong511/graceful"
)

func TestGracefulServer(t *testing.T) {

	// 创建Gin控制器
	gc := graceful.New(":8080")

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
	gc.Restart(":8081", nil) // 保持原有引擎，只修改配置

	// 模拟运行一段时间后重启
	time.Sleep(10 * time.Second)

	gc.Restart(":2580", nil)

	// 模拟运行一段时间后停止
	time.Sleep(10 * time.Second)
	gc.Stop()

	// 让服务继续运行
	select {}
}

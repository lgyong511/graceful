package graceful

import (
	"context"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// GinController 控制gin的结构体
type GinController struct {
	mu     sync.Mutex         // 保护并发访问
	srv    *http.Server       // HTTP服务器实例
	sig    chan controlSignal // 控制信号通道
	wg     sync.WaitGroup     // 用于等待所有goroutine完成
	addr   string             // 端口
	engine *gin.Engine        // gin引擎实例
}

// controlSignal 控制信号
type controlSignal struct {
	action actionType  // 动作类型
	addr   string      // 可选的配置更新(用于重启时修改配置)
	engine *gin.Engine // 可选的引擎更新(用于重启时更换引擎)
}

type actionType uint

const (
	startAction   actionType = iota // 启动动作
	stopAction                      // 停止动作
	restartAction                   // 重启动作
)

// New 创建新的GinController实例
func New(addr string) *GinController {
	gc := &GinController{
		sig:  make(chan controlSignal, 1),
		addr: addr,
	}

	// 启动控制循环
	go gc.controlLoop()
	return gc
}

// Start 启动服务
func (gc *GinController) Start(engine *gin.Engine) {
	gc.mu.Lock()
	gc.engine = engine
	gc.mu.Unlock()

	gc.sig <- controlSignal{action: startAction}
}

// Stop 停止服务
func (gc *GinController) Stop() {
	gc.sig <- controlSignal{action: stopAction}
	gc.wg.Wait() // 等待所有操作完成
}

// Restart 重启服务，可选的更新配置和引擎
func (gc *GinController) Restart(addr string, newEngine *gin.Engine) {
	gc.mu.Lock()
	if len(addr) != 0 {
		gc.addr = addr
	}
	if newEngine != nil {
		gc.engine = newEngine
	}
	gc.mu.Unlock()

	gc.sig <- controlSignal{
		action: restartAction,
		addr:   addr,
		engine: newEngine,
	}
}

// controlLoop 控制循环，处理所有控制信号
func (gc *GinController) controlLoop() {
	gc.wg.Add(1)
	defer gc.wg.Done()

	for signal := range gc.sig {
		switch signal.action {
		case startAction:
			gc.startServer()

		case stopAction:
			gc.stopServer(context.Background())
			return // 退出控制循环

		case restartAction:
			gc.restartServer(context.Background())
		}
	}
}

// startServer 启动HTTP服务器
func (gc *GinController) startServer() {
	gc.mu.Lock()
	defer gc.mu.Unlock()

	if gc.engine == nil {
		logrus.Error("gin Engine未初始化，无法启动服务")
		return
	}

	if gc.srv != nil {
		logrus.Warn("服务已在运行中")
		return
	}

	gc.srv = &http.Server{
		Addr:         gc.addr,
		Handler:      gc.engine,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  30 * time.Second,
	}

	gc.wg.Add(1)
	go func() {
		defer gc.wg.Done()
		logrus.Infof("服务启动，监听地址: %s", gc.addr)
		if err := gc.srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logrus.WithError(err).Error("服务启动失败")
			os.Exit(1)
		}
	}()
}

// stopServer 停止HTTP服务器
func (gc *GinController) stopServer(ctx context.Context) {
	gc.mu.Lock()
	defer gc.mu.Unlock()

	if gc.srv == nil {
		logrus.Warn("服务未启动，无需停止")
		return
	}

	shutdownCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := gc.srv.Shutdown(shutdownCtx); err != nil {
		logrus.WithError(err).Error("服务停止失败")
		return
	}

	gc.srv = nil
	logrus.Info("服务已优雅停止")
	close(gc.sig) // 关闭信号通道
}

// restartServer 重启HTTP服务器
func (gc *GinController) restartServer(ctx context.Context) {
	gc.mu.Lock()
	defer gc.mu.Unlock()

	if gc.engine == nil {
		logrus.Error("gin Engine未初始化，无法重启服务")
		return
	}

	// 如果服务正在运行，先停止
	if gc.srv != nil {
		shutdownCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()

		if err := gc.srv.Shutdown(shutdownCtx); err != nil {
			logrus.WithError(err).Error("服务停止失败，无法重启")
			return
		}
		gc.srv = nil
		logrus.Info("服务已停止，准备重启")
	}

	// 启动新实例
	gc.srv = &http.Server{
		Addr:         gc.addr,
		Handler:      gc.engine,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  30 * time.Second,
	}

	gc.wg.Add(1)
	go func() {
		defer gc.wg.Done()
		logrus.Infof("服务重启完成，监听地址: %s", gc.addr)
		if err := gc.srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logrus.WithError(err).Error("服务重启失败")
			os.Exit(1)
		}
	}()
}

// ============================================================================
// 【文件导读】HTTP 服务的启动与优雅关闭。
//
// 这是整个 Go 后端最值得精读的 60 行。它浓缩了 Go 服务端开发的标准骨架：
//
//	goroutine 跑监听 -> 主协程阻塞等信号 -> 收到信号后带超时地关闭服务。
//
// 为什么必须"优雅"关闭？直接 kill 掉进程，正在处理中的请求会被腰斩，
// 客户端收到连接重置，未提交的事务丢失。Shutdown() 会：
//  1. 立即停止接收新连接；
//  2. 等待已有请求处理完；
//  3. 超过 ctx 超时时间仍没处理完的，才强制断开。
//
// 【本文件涉及的 Go 语法】
//   - interface 的隐式实现（无需 implements 关键字）
//   - goroutine（go 关键字）
//   - channel（make(chan T, 容量) + <- 取值阻塞）
//   - context.WithTimeout + defer cancel（超时控制与资源释放）
//   - errors 判等：err != http.ErrServerClosed
//
// ============================================================================
package core

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// [Go] 这就是 Go 的接口定义：一组方法签名的集合。
//
// Go 的接口是"隐式实现"的——没有任何类型声明过 implements server，
// 只要它恰好拥有这两个方法，它就自动满足了这个接口。
// 对比 Java/C# 的显式 implements，这是 Go 最核心的思维差异之一：
// 接口由使用方（这里是我们）定义，而不是由实现方声明。
//
// [注意] 这个接口在当前文件里其实没被用到（下面直接用 *http.Server 了），
// 属于框架里的历史残留。看到"定义了却没人用"的代码不要浪费时间去猜它的用意，
// 全局搜一下确认没引用就跳过。
type server interface {
	ListenAndServe() error
	Shutdown(context.Context) error
}

// initServer 启动服务并实现优雅关闭
func initServer(address string, router *gin.Engine, readTimeout, writeTimeout time.Duration) {
	// 创建服务
	// [Go] &http.Server{...} 取结构体字面量的地址，得到一个 *http.Server。
	// 字段名: 值 的写法叫"命名字段初始化"，可以只写部分字段，其余自动零值。
	srv := &http.Server{
		Addr:         address,
		Handler:      router,
		ReadTimeout:  readTimeout,
		WriteTimeout: writeTimeout,
		// [Go] 1 << 20 是位运算，等于 1048576，即 1MB。Go 里常用移位表示容量。
		MaxHeaderBytes: 1 << 20,
	}

	// 在goroutine中启动服务
	//
	// [Go] go 关键字启动一个 goroutine（由 Go 运行时调度的轻量级线程）。
	// 为什么必须放进 goroutine？因为 ListenAndServe() 是一个"永不返回"的阻塞调用，
	// 它内部是个 for 死循环在 accept 连接。如果直接在主协程调用，主协程就永远卡在
	// 这一行，下面等待信号的代码根本没机会执行。
	//
	// [Go] 末尾的 () 是立即调用这个匿名函数——写 `go func(){...}` 而不加 () 是语法错误。
	go func() {
		// [注意] 这里的 err != http.ErrServerClosed 判断不能省。
		// 主动调用 Shutdown() 时，ListenAndServe 会返回 ErrServerClosed，
		// 这是"正常的关闭"，不是错误。不排除它的话，每次优雅重启都会误报启动失败。
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Printf("listen: %s\n", err)
			zap.L().Error("server启动失败", zap.Error(err))
			// [注意] os.Exit 会立刻终止整个进程，defer 不会执行。
			// 监听失败（通常是端口被占用）是致命错误，没有继续跑的必要。
			os.Exit(1)
		}
	}()

	// 等待中断信号以优雅地关闭服务器

	// [Go] make(chan os.Signal, 1) 创建一个带缓冲区的 channel，容量为 1。
	// 容量为 1 的意义：信号到达时即使主协程还没读到，也能先存一个在缓冲区里，
	// 不会被丢弃（无缓冲 channel 要求收发双方同时就绪）。
	quit := make(chan os.Signal, 1)
	// kill (无参数) 默认发送 syscall.SIGTERM
	// kill -2 发送 syscall.SIGINT   （也就是 Ctrl+C）
	// kill -9 发送 syscall.SIGKILL，但是无法被捕获，所以不需要添加
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// [Go] `<-quit` 是从 channel 接收数据。这一行会一直阻塞，
	// 直到上面那个 channel 里被塞进一个信号。
	// 关键点：主协程因此"挂起"而不是"退出"。如果把这一行删掉，
	// main() 会立刻执行完返回，进程直接退出，服务就起不来了。
	<-quit
	zap.L().Info("关闭WEB服务...")

	// 设置5秒的超时时间
	//
	// [Go] context 是 Go 传递"取消信号 + 超时"的标准机制。
	// WithTimeout 返回一个派生 context 和一个 cancel 函数。
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)

	// [Go] defer 把 cancel() 推迟到本函数返回时执行，这是固定搭配。
	// 不调用 cancel 会造成 context 泄漏（它内部的定时器和引用不会被释放）。
	// Go 官方检查工具 go vet 会专门提醒 context.WithXxx 的 cancel 是否被调用。
	defer cancel()

	// 把 ctx 交给 Shutdown：最多等 5 秒，超时就强制断开剩余连接。
	if err := srv.Shutdown(ctx); err != nil {
		zap.L().Fatal("WEB服务关闭异常", zap.Error(err))
	}

	zap.L().Info("WEB服务已关闭")
}

package main

import (
	"go.uber.org/zap/zapcore"
	"kit.golaxy.org/golaxy"
	"kit.golaxy.org/golaxy/plugin"
	"kit.golaxy.org/golaxy/pt"
	"kit.golaxy.org/golaxy/service"
	gtp_gate "kit.golaxy.org/plugins/gate/gtp"
	zap_logger "kit.golaxy.org/plugins/logger/zap"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	if len(os.Args) < 2 {
		panic("missing endpoints")
	}

	// 创建实体库，注册实体原型
	entityLib := pt.NewEntityLib()
	entityLib.Register("demo", []string{
		defineDemoComp.Implementation,
	})

	// 创建插件包
	pluginBundle := plugin.NewPluginBundle()

	// 安装日志插件
	zapLogger, _ := zap_logger.NewConsoleZapLogger(zapcore.DebugLevel, "\t", "", 0, true, true)
	zap_logger.Install(pluginBundle, zap_logger.Option{}.ZapLogger(zapLogger), zap_logger.Option{}.Fields(0))

	// 安装网关插件
	gtp_gate.Install(pluginBundle,
		gtp_gate.Option{}.Endpoints(os.Args[1:]...),
		gtp_gate.Option{}.IOTimeout(10*time.Minute),
		gtp_gate.Option{}.IORetryTimes(1000),
		gtp_gate.Option{}.AgreeClientEncryptionProposal(true),
		gtp_gate.Option{}.AgreeClientCompressionProposal(true),
		gtp_gate.Option{}.CompressedSize(128),
		gtp_gate.Option{}.SessionStateChangedHandlers(SessionStateChangedHandler),
	)

	// 创建服务上下文与服务，并开始运行
	<-golaxy.NewService(service.NewContext(
		service.Option{}.EntityLib(entityLib),
		service.Option{}.PluginBundle(pluginBundle),
		service.Option{}.Name("demo_gate"),
		service.Option{}.RunningHandler(func(ctx service.Context, state service.RunningState) {
			if state != service.RunningState_Started {
				return
			}

			// 监听退出信号
			sigChan := make(chan os.Signal, 1)
			signal.Notify(sigChan, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

			go func() {
				<-sigChan
				ctx.GetCancelFunc()()
			}()
		}),
	)).Run()
}
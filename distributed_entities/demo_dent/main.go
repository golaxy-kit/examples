package main

import (
	"fmt"
	"git.golaxy.org/core"
	"git.golaxy.org/core/ec"
	"git.golaxy.org/core/plugin"
	"git.golaxy.org/core/pt"
	"git.golaxy.org/core/runtime"
	"git.golaxy.org/core/service"
	"git.golaxy.org/core/util/generic"
	"git.golaxy.org/core/util/uid"
	"git.golaxy.org/framework/plugins/broker/nats_broker"
	"git.golaxy.org/framework/plugins/dent"
	"git.golaxy.org/framework/plugins/dentq"
	"git.golaxy.org/framework/plugins/discovery/etcd_discovery"
	"git.golaxy.org/framework/plugins/dserv"
	"git.golaxy.org/framework/plugins/dsync/etcd_dsync"
	"git.golaxy.org/framework/plugins/log"
	"git.golaxy.org/framework/plugins/log/console_log"
	"git.golaxy.org/framework/plugins/rpc"
	"github.com/segmentio/ksuid"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	entIdList := []ksuid.KSUID{ksuid.New()}
	var servChanList []<-chan struct{}
	total := 6

	for i := 0; i < total; i++ {
		// 创建实体库，注册实体原型
		entityLib := pt.NewEntityLib(pt.DefaultComponentLib())
		entityLib.Register("demo", pt.CompNamePair(DemoComp{}, "DemoComp"))

		// 创建插件包，安装插件
		pluginBundle := plugin.NewPluginBundle()
		console_log.Install(pluginBundle, console_log.Option{}.Level(log.DebugLevel), console_log.Option{}.ServiceInfo(true))
		nats_broker.Install(pluginBundle, nats_broker.Option{}.CustomAddresses("127.0.0.1:4222"))
		etcd_discovery.Install(pluginBundle, etcd_discovery.Option{}.CustomAddresses("127.0.0.1:12379"))
		etcd_dsync.Install(pluginBundle, etcd_dsync.Option{}.CustomAddresses("127.0.0.1:12379"))
		dserv.Install(pluginBundle, dserv.Option{}.FutureTimeout(time.Minute))
		rpc.Install(pluginBundle)
		dentq.Install(pluginBundle, dentq.Option{}.CustomAddresses("127.0.0.1:12379"))

		// 创建服务上下文与服务，并开始运行
		serv := core.NewService(service.NewContext(
			service.Option{}.EntityLib(entityLib),
			service.Option{}.PluginBundle(pluginBundle),
			service.Option{}.Name(fmt.Sprintf("demo_dent_%d", i%(total/2))),
			service.Option{}.RunningHandler(generic.CastDelegateAction2(func(ctx service.Context, state service.RunningState) {
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

				// 创建运行时上下文与运行时，安装插件并开始运行
				rtCtx := runtime.NewContext(ctx,
					runtime.Option{}.Context.RunningHandler(generic.CastDelegateAction2(func(_ runtime.Context, state runtime.RunningState) {
						if state != runtime.RunningState_Terminated {
							return
						}
						ctx.GetCancelFunc()()
					})),
				)
				console_log.Install(rtCtx, console_log.Option{}.Level(log.DebugLevel), console_log.Option{}.ServiceInfo(true))
				dent.Install(rtCtx, dent.Option{}.CustomAddresses("127.0.0.1:12379"))

				rt := core.NewRuntime(rtCtx, core.Option{}.Runtime.AutoRun(true))

				// 在运行时线程环境中，创建实体
				for i := range entIdList {
					entId := entIdList[i]

					core.AsyncVoid(rt, func(ctx runtime.Context, _ ...any) {
						entity, err := core.CreateEntity(ctx,
							core.Option{}.EntityCreator.Prototype("demo"),
							core.Option{}.EntityCreator.Scope(ec.Scope_Global),
							core.Option{}.EntityCreator.PersistId(uid.Id(entId.String())),
						).Spawn()
						if err != nil {
							log.Panic(service.Current(ctx), err)
						}
						log.Infof(service.Current(ctx), "create entity %q finish", entity)
					})
				}
			})),
		))

		servChanList = append(servChanList, serv.Run())
	}

	for _, servChan := range servChanList {
		<-servChan
	}
}
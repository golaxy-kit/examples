package main

import (
	"git.golaxy.org/core"
	"git.golaxy.org/core/ec"
	"git.golaxy.org/core/plugin"
	"git.golaxy.org/core/pt"
	"git.golaxy.org/core/runtime"
	"git.golaxy.org/core/service"
	"git.golaxy.org/core/util/generic"
	"git.golaxy.org/framework/plugins/log"
	"git.golaxy.org/framework/plugins/log/console_log"
)

func main() {
	// 创建实体库，注册实体原型
	entityLib := pt.NewEntityLib(pt.DefaultComponentLib())
	entityLib.Declare("demo", DemoComp{})

	// 创建插件包，安装插件
	pluginBundle := plugin.NewPluginBundle()
	console_log.Install(pluginBundle, console_log.With.Level(log.DebugLevel))

	// 创建服务上下文与服务，并开始运行
	<-core.NewService(service.NewContext(
		service.With.EntityLib(entityLib),
		service.With.PluginBundle(pluginBundle),
		service.With.Name("demo_ec"),
		service.With.RunningHandler(generic.MakeDelegateAction2(func(ctx service.Context, state service.RunningState) {
			if state != service.RunningState_Started {
				return
			}

			// 创建运行时上下文与运行时，并开始运行
			rt := core.NewRuntime(
				runtime.NewContext(ctx,
					runtime.With.Context.RunningHandler(generic.MakeDelegateAction2(func(_ runtime.Context, state runtime.RunningState) {
						if state != runtime.RunningState_Terminated {
							return
						}
						ctx.GetCancelFunc()()
					})),
				),
				core.With.Runtime.Frame(runtime.NewFrame(runtime.With.Frame.TotalFrames(100))),
				core.With.Runtime.AutoRun(true),
			)

			// 在运行时线程环境中，创建实体
			core.AsyncVoid(rt, func(ctx runtime.Context, _ ...any) {
				entity1, err := core.CreateEntity(ctx).
					Prototype("demo").
					Scope(ec.Scope_Global).
					Spawn()
				if err != nil {
					log.Panic(service.Current(ctx), err)
				}

				log.Infof(service.Current(ctx), "create entity %q finish", entity1)

				entity2, err := core.CreateEntity(ctx).
					Prototype("demo").
					Scope(ec.Scope_Global).
					ParentId(entity1.GetId()).
					Spawn()
				if err != nil {
					log.Panic(service.Current(ctx), err)
				}

				log.Infof(service.Current(ctx), "create entity %q finish", entity2)

				entity3, err := core.CreateEntity(ctx).
					Prototype("demo").
					Scope(ec.Scope_Global).
					ParentId(entity1.GetId()).
					Spawn()
				if err != nil {
					log.Panic(service.Current(ctx), err)
				}

				log.Infof(service.Current(ctx), "create entity %q finish", entity3)

				entity4, err := core.CreateEntity(ctx).
					Prototype("demo").
					Scope(ec.Scope_Global).
					ParentId(entity2.GetId()).
					Spawn()
				if err != nil {
					log.Panic(service.Current(ctx), err)
				}

				log.Infof(service.Current(ctx), "create entity %q finish", entity4)

				ctx.GetEntityTree().RangeChildren(entity1.GetId(), func(child ec.Entity) bool {
					log.Infof(service.Current(ctx), "child: %s <- %s", entity1.GetId(), child.GetId())
					return true
				})

				ctx.GetEntityTree().RangeChildren(entity2.GetId(), func(child ec.Entity) bool {
					log.Infof(service.Current(ctx), "child: %s <- %s", entity2.GetId(), child.GetId())
					return true
				})

				ctx.GetEntityTree().ChangeParentNode(entity2.GetId(), entity3.GetId())

				ctx.GetEntityTree().RangeChildren(entity1.GetId(), func(child ec.Entity) bool {
					log.Infof(service.Current(ctx), "child: %s <- %s", entity1.GetId(), child.GetId())
					return true
				})

				ctx.GetEntityTree().PruningNode(entity3.GetId())

				entity3.DestroySelf()
			})
		})),
	)).Run()
}

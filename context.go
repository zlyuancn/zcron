package zcron

type Handler func(ctx IContext) (err error)

type IContext interface {
	// 获取task
	Task() ITask
	// 获取handler
	Handler() Handler
	// 获取元数据
	Meta() interface{}
	// 设置元数据
	SetMeta(meta interface{})
}

type Context struct {
	task    ITask
	handler Handler
	meta    interface{}
}

func newContext(task ITask) IContext {
	return &Context{
		task:    task,
		handler: task.Handler(),
		meta:    nil,
	}
}

func (ctx *Context) Task() ITask {
	return ctx.task
}

func (ctx *Context) Handler() Handler {
	return ctx.handler
}

func (ctx *Context) Meta() interface{} {
	return ctx.meta
}
func (ctx *Context) SetMeta(meta interface{}) {
	ctx.meta = meta
}

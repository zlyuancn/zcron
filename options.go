package zcron

import (
	"github.com/zlyuancn/zgpool"
	"github.com/zlyuancn/zlog"
)

type Options struct {
	log   zlog.Loger   // 日志
	gpool *zgpool.Pool // 协程池
}

func newOptions() *Options {
	return &Options{
		log: zlog.DefaultLogger,
	}
}

type Option func(o *Options)

// 设置日志组件
func WithLogger(log zlog.Loger) Option {
	return func(o *Options) {
		o.log = log
	}
}

// 设置同时处理job的最大goroutine数和job队列大小
// 产生任务时如果队列已满会抛弃掉
// 如果threadCount为0, 所有的job都会开启一个goroutine
func WithGoroutinePool(threadCount int, jobQueueSize int) Option {
	return func(o *Options) {
		if threadCount == 0 {
			o.gpool = nil
			return
		}
		o.gpool = zgpool.NewPool(threadCount, jobQueueSize)
	}
}

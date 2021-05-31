package zcron

import (
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"github.com/zlyuancn/zutils"
)

var OutOfMaxConcurrentExecuteCount = errors.New("超出最大并发执行数")

// 错误回调, 只有会被重试时才会调用
type ErrCallback func(ctx IContext, err error)

type IExecutor interface {
	// 执行
	Do(ctx IContext, errCallback ErrCallback) error
	// 等待任务执行完毕
	Wait()
	// 返回是否正在执行任务
	IsRunning() bool
	// 执行器信息
	ExecutorInfo() *ExecutorInfo
}

type ExecutorInfo struct {
	MaxConcurrentExecuteCount int64         // 最大并发执行数
	ConcurrentExecuteCount    int64         // 当前并发执行数
	MaxRetryCount             int64         // 最大重试数
	RetryInterval             time.Duration // 重试间隔
}

type Executor struct {
	maxConcurrentExecuteCount int64         // 最大并发执行数
	concurrentExecuteCount    int64         // 当前并发执行数
	maxRetryCount             int64         // 重试次数
	retryInterval             time.Duration // 重试间隔
	wg                        sync.WaitGroup
}

// 创建一个执行器, 任务失败会重试
//
// maxRetryCount: 任务失败重试次数
// retryInterval: 失败重试间隔时间
// maxConcurrentExecuteCount: 最大并发执行任务数, 如果为0则不限制
func NewExecutor(retryCount int64, retryInterval time.Duration, maxSyncExecuteCount int64) IExecutor {
	return &Executor{
		maxConcurrentExecuteCount: maxSyncExecuteCount,
		concurrentExecuteCount:    0,
		maxRetryCount:             retryCount,
		retryInterval:             retryInterval,
	}
}

func (w *Executor) ExecutorInfo() *ExecutorInfo {
	return &ExecutorInfo{
		MaxConcurrentExecuteCount: w.maxConcurrentExecuteCount,
		ConcurrentExecuteCount:    atomic.LoadInt64(&w.concurrentExecuteCount),
		MaxRetryCount:             w.maxRetryCount,
		RetryInterval:             w.retryInterval,
	}
}

// 执行, 如果已经达到最大并发执行任务数则会返回错误
func (w *Executor) Do(ctx IContext, errCallback ErrCallback) error {
	if w.maxConcurrentExecuteCount > 0 && atomic.LoadInt64(&w.concurrentExecuteCount) >= w.maxConcurrentExecuteCount {
		return OutOfMaxConcurrentExecuteCount
	}

	w.wg.Add(1)
	atomic.AddInt64(&w.concurrentExecuteCount, 1)

	err := w.doRetry(ctx, w.retryInterval, w.maxRetryCount, errCallback)

	atomic.AddInt64(&w.concurrentExecuteCount, -1)
	w.wg.Done()
	return err
}

// 等待所有任务执行完毕
func (w *Executor) Wait() {
	w.wg.Wait()
}

// 返回是否正在执行任务
func (w *Executor) IsRunning() bool {
	return atomic.LoadInt64(&w.concurrentExecuteCount) > 0
}

// 执行一个函数
func (w *Executor) doRetry(ctx IContext, interval time.Duration, retryCount int64, errCallback ErrCallback) (err error) {
	for {
		err = zutils.Recover.WrapCall(func() error {
			return ctx.Handler()(ctx)
		})
		if err == nil || retryCount == 0 {
			// 这里不需要错误回调, 如果有err交给调用者处理
			return
		}

		retryCount--

		if errCallback != nil {
			errCallback(ctx, err)
		}

		if interval > 0 {
			time.Sleep(interval)
		}
	}
}

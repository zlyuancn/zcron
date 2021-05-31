package zcron

import (
	"sync"
	"sync/atomic"
	"time"

	jsoniter "github.com/json-iterator/go"
	"github.com/zlyuancn/zutils"
)

type ITask interface {
	// 返回任务名
	Name() string
	// 获取handler
	Handler() Handler
	// 返回启用状态
	IsEnable() bool
	// 任务信息
	TaskInfo() *TaskInfo
	// 输出
	String() string

	// 设置启用
	Enable(enable ...bool)
	// 获取触发时间
	TriggerTime() time.Time
	// 生成下次触发时间, 如果返回了 false 表示没有下一次了, 返回的时间一定>t
	MakeNextTriggerTime(t time.Time) (time.Time, bool)
	// 立即触发执行, 阻塞等待执行结束
	Trigger(callback ErrCallback) *ExecuteInfo
	// 重置定时, 发生在被定时器添加任务时和重新设为启用时
	ResetClock()

	// 修改触发器
	ChangeTrigger(trigger ITrigger)
	// 修改执行器
	ChangeExecutor(executor IExecutor)

	// 设置堆索引
	setHeapIndex(index int)
	// 获取堆索引
	getHeapIndex() int
}

// 任务信息
type TaskInfo struct {
	// 任务名
	Name string
	// 是否启用
	Enable bool
	// 执行成功数
	ExecutedSuccessNum uint64
	// 执行失败数
	ExecuteFailureNum uint64
	// 最后一次执行信息
	LastExecuteInfo *ExecuteInfo

	// 触发器信息
	TriggerInfo *TriggerInfo
	// 执行器信息
	ExecutorInfo *ExecutorInfo
}

// 执行信息
type ExecuteInfo struct {
	// 执行开始时间
	ExecuteStartTime time.Time
	// 执行结束时间
	ExecuteEndTime time.Time
	// 是否执行成功
	ExecuteSuccess bool
	// 执行的错误, 如果最后一次执行没有错误那么值为nil
	ExecuteErr error
}

type Task struct {
	name       string
	successNum uint64
	failureNum uint64

	handler         Handler
	lastExecuteInfo *ExecuteInfo

	triggerTime time.Time
	trigger     ITrigger
	executor    IExecutor

	enable int32
	mx     sync.Mutex // 用于锁 triggerTime, lastExecuteInfo, trigger, executor

	heapIndex int // 堆索引
}

type TaskConfig struct {
	Trigger  ITrigger
	Executor IExecutor
	Handler  Handler
	Enable   bool
}

// 创建一个任务
func NewTask(name string, expression string, enable bool, handler Handler) ITask {
	trigger := NewCronTrigger(expression)
	executor := NewExecutor(0, 0, 1)
	return NewTaskOfConfig(name, TaskConfig{
		Trigger:  trigger,
		Executor: executor,
		Handler:  handler,
		Enable:   enable,
	})
}

// 根据任务配置创建一个任务
func NewTaskOfConfig(name string, config TaskConfig) ITask {
	t := &Task{
		name:     name,
		trigger:  config.Trigger,
		executor: config.Executor,
		handler:  config.Handler,
	}
	t.Enable(config.Enable)
	return t
}

func (t *Task) Name() string {
	return t.name
}
func (t *Task) Handler() Handler {
	return t.handler
}
func (t *Task) IsEnable() bool {
	return atomic.LoadInt32(&t.enable) == 1
}
func (t *Task) TaskInfo() *TaskInfo {
	t.mx.Lock()
	lastInfo := t.lastExecuteInfo
	trigger := t.trigger
	executor := t.executor
	t.mx.Unlock()

	info := &TaskInfo{
		Name:               t.name,
		Enable:             t.IsEnable(),
		ExecutedSuccessNum: atomic.LoadUint64(&t.successNum),
		ExecuteFailureNum:  atomic.LoadUint64(&t.failureNum),
		LastExecuteInfo:    nil,

		TriggerInfo:  trigger.TriggerInfo(),
		ExecutorInfo: executor.ExecutorInfo(),
	}

	if lastInfo != nil {
		v := *lastInfo
		info.LastExecuteInfo = &v
	}

	return info
}
func (t *Task) String() string {
	taskInfo := t.TaskInfo()
	triggerInfo := taskInfo.TriggerInfo
	executorInfo := taskInfo.ExecutorInfo
	info := map[string]interface{}{
		"name":                taskInfo.Name,
		"enable":              taskInfo.Enable,
		"execute_success_num": taskInfo.ExecutedSuccessNum,
		"execute_failure_num": taskInfo.ExecuteFailureNum,
		"last_execute_info":   nil,

		"trigger_info": map[string]interface{}{
			"trigger_type":            triggerInfo.TriggerType,
			"expression":              triggerInfo.Expression,
			"next_trigger_time":       zutils.Time.TimeToText(triggerInfo.NextTriggerTime),
			"next_trigger_time_stamp": triggerInfo.NextTriggerTime.Unix(),
		},

		"executor_info": map[string]interface{}{
			"max_concurrent_execute_count": executorInfo.MaxConcurrentExecuteCount,
			"concurrent_execute_count":     executorInfo.ConcurrentExecuteCount,
			"max_retry_count":              executorInfo.MaxRetryCount,
			"retry_interval":               executorInfo.RetryInterval.String(),
		},
	}

	if taskInfo.LastExecuteInfo != nil {
		var errMsg string
		if taskInfo.LastExecuteInfo.ExecuteErr != nil {
			errMsg = taskInfo.LastExecuteInfo.ExecuteErr.Error()
		}
		info["last_execute_info"] = map[string]interface{}{
			"execute_start_time":       zutils.Time.TimeToText(taskInfo.LastExecuteInfo.ExecuteStartTime),
			"execute_start_time_stamp": taskInfo.LastExecuteInfo.ExecuteStartTime.Unix(),
			"execute_end_time":         zutils.Time.TimeToText(taskInfo.LastExecuteInfo.ExecuteEndTime),
			"execute_end_time_stamp":   taskInfo.LastExecuteInfo.ExecuteEndTime.Unix(),
			"execute_success":          taskInfo.LastExecuteInfo.ExecuteSuccess,
			"err_message":              errMsg,
		}
	}

	text, _ := jsoniter.ConfigCompatibleWithStandardLibrary.MarshalToString(info)
	return text
}

func (t *Task) Enable(enable ...bool) {
	if len(enable) == 0 || enable[0] {
		atomic.StoreInt32(&t.enable, 1)
		t.ResetClock()
	} else {
		atomic.StoreInt32(&t.enable, 0)
	}
}
func (t *Task) TriggerTime() time.Time {
	t.mx.Lock()
	tt := t.triggerTime
	t.mx.Unlock()
	return tt
}
func (t *Task) MakeNextTriggerTime(tt time.Time) (time.Time, bool) {
	tt, ok := t.trigger.MakeNextTriggerTime(tt)
	t.mx.Lock()
	t.triggerTime = tt
	t.mx.Unlock()
	return tt, ok
}
func (t *Task) Trigger(callback ErrCallback) *ExecuteInfo {
	ctx := newContext(t)
	info := &ExecuteInfo{
		ExecuteStartTime: time.Now(),
	}

	err := t.execute(ctx, callback)
	info.ExecuteEndTime = time.Now()
	if err != nil {
		info.ExecuteErr = err
		atomic.AddUint64(&t.failureNum, 1)
	} else {
		info.ExecuteSuccess = true
		atomic.AddUint64(&t.successNum, 1)
	}

	t.mx.Lock()
	t.lastExecuteInfo = info
	t.mx.Unlock()
	return info
}
func (t *Task) ResetClock() {
	t.mx.Lock()
	trigger := t.trigger
	t.mx.Unlock()
	trigger.ResetClock()
}

// 执行
func (t *Task) execute(ctx IContext, errCallback ErrCallback) error {
	t.mx.Lock()
	executor := t.executor
	t.mx.Unlock()
	return executor.Do(ctx, errCallback)
}

func (t *Task) ChangeTrigger(trigger ITrigger) {
	t.mx.Lock()
	trigger.ResetClock()
	t.trigger = trigger
	t.mx.Unlock()
}
func (t *Task) ChangeExecutor(executor IExecutor) {
	t.mx.Lock()
	t.executor = executor
	t.mx.Unlock()
}

func (t *Task) setHeapIndex(index int) {
	t.heapIndex = index
}
func (t *Task) getHeapIndex() int {
	return t.heapIndex
}

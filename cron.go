package zcron

import (
	"fmt"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	jsoniter "github.com/json-iterator/go"
	"github.com/zlyuancn/zlog"
	"github.com/zlyuancn/zutils"
	"go.uber.org/zap"
)

type ICron interface {
	// 运行状态
	RunState() RunState
	// 定时器信息
	CronInfo() *CronInfo
	// 输出
	String() string

	// 启动
	Start()
	// 结束
	Stop()
	// 暂停所有任务
	Pause()
	// 恢复所有任务
	Resume()

	// 添加任务, 如果任务名重复返回 false
	AddTask(task ITask) bool
	// 移除任务
	RemoveTask(name string)
	// 启用任务
	EnableTask(task ITask, enable bool)
	// 获取任务名列表, 按名称排序
	TaskNames() []string
	// 获取任务列表, 按名称排序
	Tasks() []ITask
	// 获取任务, 如果不存在返回nil
	GetTask(name string) ITask
}

// 运行状态
type RunState int32

const (
	// 已停止
	StoppedState RunState = iota
	// 启动中
	StartingState
	// 暂停
	PausedState
	// 恢复中
	ResumingState
	// 已启动
	StartedState
	// 停止中
	StoppingState
)

func (r RunState) String() string {
	switch r {
	case StoppedState:
		return "stopped"
	case StartingState:
		return "starting"
	case PausedState:
		return "paused"
	case ResumingState:
		return "resuming"
	case StartedState:
		return "started"
	case StoppingState:
		return "stopping"
	}
	return fmt.Sprintf("undefined state: %d", r)
}

// 定时器信息
type CronInfo struct {
	// 运行状态
	RunState RunState
	// 任务数
	TaskNum uint32
	// 执行成功数
	ExecutedSuccessNum uint64
	// 执行失败数
	ExecuteFailureNum uint64
}

// =================================

const heapsCount = 64 // 任务堆数量

type Cron struct {
	*Options

	tasks map[string]ITask // 任务
	heaps []ITaskHeap      // 任务堆列表, 根据触发时间取模将任务分配到不同的任务堆

	runState  RunState
	closeChan chan struct{}

	taskNum    uint32
	successNum uint64
	failureNum uint64

	mx sync.Mutex // 锁 tasks, heaps
}

func (s *Cron) RunState() RunState {
	return RunState(atomic.LoadInt32((*int32)(&s.runState)))
}

func (s *Cron) CronInfo() *CronInfo {
	info := &CronInfo{
		RunState:           s.RunState(),
		TaskNum:            atomic.LoadUint32(&s.taskNum),
		ExecutedSuccessNum: atomic.LoadUint64(&s.successNum),
		ExecuteFailureNum:  atomic.LoadUint64(&s.failureNum),
	}
	return info
}

func (s *Cron) String() string {
	cronInfo := s.CronInfo()
	info := map[string]interface{}{
		"run_state":           cronInfo.RunState,
		"run_state_text":      cronInfo.RunState.String(),
		"task_num":            cronInfo.TaskNum,
		"execute_success_num": cronInfo.ExecutedSuccessNum,
		"execute_failure_num": cronInfo.ExecuteFailureNum,
	}
	text, _ := jsoniter.ConfigCompatibleWithStandardLibrary.MarshalToString(info)
	return text
}

func (s *Cron) Start() {
	if !atomic.CompareAndSwapInt32((*int32)(&s.runState), int32(StoppedState), int32(StartingState)) {
		return
	}
	s.log.Debug("正在启动定时器")

	s.resetClock()
	go s.start()

	atomic.StoreInt32((*int32)(&s.runState), int32(StartedState))
	s.log.Info("已启动定时器")
}

func (s *Cron) Stop() {
	state := atomic.LoadInt32((*int32)(&s.runState))
	if RunState(state) == StoppingState || RunState(state) == StoppedState {
		return
	}

	if !atomic.CompareAndSwapInt32((*int32)(&s.runState), state, int32(StoppingState)) {
		return
	}

	s.log.Warn("正在关闭定时器")
	s.closeChan <- struct{}{}
	<-s.closeChan

	atomic.StoreInt32((*int32)(&s.runState), int32(StoppedState))
	s.log.Warn("已关闭定时器")
}

func (s *Cron) Pause() {
	if s.RunState() != StartedState {
		return
	}

	if atomic.CompareAndSwapInt32((*int32)(&s.runState), int32(StartedState), int32(PausedState)) {
		s.log.Warn("暂停定时器")
	}
}

func (s *Cron) Resume() {
	if s.RunState() != StartedState {
		return
	}

	if atomic.CompareAndSwapInt32((*int32)(&s.runState), int32(PausedState), int32(ResumingState)) {
		s.resetClock()

		// 设为启动状态(恢复完成)
		// 这里使用cas是为了防止这个时候用户调用了Stop()+Start()后状态被更改
		if atomic.CompareAndSwapInt32((*int32)(&s.runState), int32(ResumingState), int32(StartedState)) {
			s.log.Info("恢复定时器")
		}
	}
}

func (s *Cron) AddTask(task ITask) bool {
	s.mx.Lock()
	if _, ok := s.tasks[task.Name()]; ok { // 已存在
		s.mx.Unlock()
		return false
	}

	s.tasks[task.Name()] = task

	if task.IsEnable() && s.RunState() == StartedState {
		task.resetClock()
		_, ok := task.MakeNextTriggerTime(time.Now())
		if ok {
			s.pushTaskToHeap(task)
		}
	}

	s.mx.Unlock()
	return true
}

func (s *Cron) RemoveTask(name string) {
	s.mx.Lock()
	task, ok := s.tasks[name]
	if !ok {
		s.mx.Unlock()
		return
	}

	delete(s.tasks, name)

	heap := s.getHeapOfTime(task.TriggerTime().Unix())
	heap.Remove(task)

	s.mx.Unlock()
}

func (s *Cron) EnableTask(task ITask, enable bool) {
	s.mx.Lock()
	rawTask, ok := s.tasks[task.Name()]
	if !ok || rawTask != task {
		s.mx.Unlock()
		return
	}

	heap := s.getHeapOfTime(task.TriggerTime().Unix())
	heap.Remove(task)

	task.setEnable(enable)
	if enable && s.RunState() == StartedState {
		task.resetClock()
		_, ok := task.MakeNextTriggerTime(time.Now())
		if ok {
			s.pushTaskToHeap(task)
		}
	}
	s.mx.Unlock()
}

func (s *Cron) TaskNames() []string {
	s.mx.Lock()
	names := make([]string, len(s.tasks))
	var index int
	for name := range s.tasks {
		names[index] = name
		index++
	}
	s.mx.Unlock()

	sort.Strings(names)
	return names
}

func (s *Cron) Tasks() []ITask {
	s.mx.Lock()
	tasks := make([]ITask, len(s.tasks))
	var index int
	for _, task := range s.tasks {
		tasks[index] = task
		index++
	}
	s.mx.Unlock()

	sort.Slice(tasks, func(i, j int) bool {
		return tasks[i].Name() < tasks[j].Name()
	})
	return tasks
}

func (s *Cron) GetTask(name string) ITask {
	s.mx.Lock()
	task := s.tasks[name]
	s.mx.Unlock()
	return task
}

// 开始
func (s *Cron) start() {
	timer := time.NewTicker(time.Second)
	for {
		select {
		case t := <-timer.C:
			if s.isStarted() {
				go s.heartBeat(t)
			}
		case <-s.closeChan:
			timer.Stop()
			s.closeChan <- struct{}{}
			return
		}
	}
}

// 是否已开始
func (s *Cron) isStarted() bool {
	return atomic.LoadInt32((*int32)(&s.runState)) == int32(StartedState)
}

// 构建时间堆
func (s *Cron) remakeHeaps() {
	heaps := make([]ITaskHeap, heapsCount)
	for i := 0; i < heapsCount; i++ {
		heaps[i] = NewTaskHeap()
	}
	s.heaps = heaps
}

// 根据时间获取任务堆
func (s *Cron) getHeapOfTime(sec int64) ITaskHeap {
	bucket := sec & (heapsCount - 1)
	return s.heaps[bucket]
}

// 将任务放入任务堆中
func (s *Cron) pushTaskToHeap(task ITask) {
	heap := s.getHeapOfTime(task.TriggerTime().Unix())
	heap.Push(task)
}

// 心跳
func (s *Cron) heartBeat(t time.Time) {
	heap := s.getHeapOfTime(t.Unix())

	s.mx.Lock()
	defer s.mx.Unlock()

	for {
		if len(heap.Tasks()) == 0 { // 没有任务
			return
		}

		task := heap.Tasks()[0]
		if task.TriggerTime().After(t) { // 时间未到
			return
		}

		task = heap.Pop()
		s.triggerTask(task) // 触发

		// 获取下一次触发时间
		_, ok := task.MakeNextTriggerTime(t)
		if ok {
			s.pushTaskToHeap(task)
		}
	}
}

// 触发一个任务
func (s *Cron) triggerTask(t ITask) {
	if s.gpool == nil {
		go s.execute(t)
		return
	}

	add := s.gpool.TryAddJob(func() {
		s.execute(t)
	})
	if !add {
		s.log.Warn("任务生成失败, 因为队列已满", zap.String("name", t.Name()))
	}
}

// 执行一个任务
func (s *Cron) execute(task ITask) {
	if !task.IsEnable() {
		return
	}

	s.log.Debug("开始执行任务", zap.String("name", task.Name()))

	executeInfo := task.Trigger(func(ctx IContext, err error) {
		s.log.Warn("任务执行失败, 即将重试", zap.String("name", task.Name()), zap.String("err", zutils.Recover.GetRecoverErrorDetail(err)))
	})
	if executeInfo.ExecuteSuccess {
		atomic.AddUint64(&s.successNum, 1)
		s.log.Debug("任务执行成功", zap.String("name", task.Name()))
	} else {
		atomic.AddUint64(&s.failureNum, 1)
		s.log.Error("任务执行失败", zap.String("name", task.Name()), zap.String("err", zutils.Recover.GetRecoverErrorDetail(executeInfo.ExecuteErr)))
	}
}

// 重置定时器
//
// 会重新创建任务堆列表并重新将所有任务加入堆中.
// 这里不要做任何耗时操作, 否则可能会错过下一秒的时间导致任务会延迟64秒后执行
func (s *Cron) resetClock() {
	s.mx.Lock()
	s.remakeHeaps()

	now := time.Now()
	for _, task := range s.tasks {
		if !task.IsEnable() {
			continue
		}

		task.resetClock()
		_, ok := task.MakeNextTriggerTime(now)
		if ok {
			s.pushTaskToHeap(task)
		}
	}
	s.mx.Unlock()
}

func NewCron(opts ...Option) ICron {
	s := &Cron{
		Options:   newOptions(),
		tasks:     make(map[string]ITask),
		runState:  StoppedState,
		closeChan: make(chan struct{}),
	}
	s.remakeHeaps()

	for _, o := range opts {
		o(s.Options)
	}
	if s.log == nil {
		s.log = zlog.DefaultLogger
	}
	if s.gpool != nil {
		s.gpool.Start()
	}
	return s
}

package zcron

import (
	"fmt"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
	"github.com/zlyuancn/zutils"
)

// 触发器类型
type TriggerType int

const (
	// Cron触发器
	CronTriggerType TriggerType = iota
	// 一次性触发器
	OnceTriggerType
)

func (t TriggerType) String() string {
	switch t {
	case CronTriggerType:
		return "cron"
	case OnceTriggerType:
		return "once"
	}
	return fmt.Sprintf("undefined trigger type: %d", t)
}

type ITrigger interface {
	// 触发器类型
	TriggerType() TriggerType
	// 返回触发器表达式
	Expression() string
	// 返回触发器信息
	TriggerInfo() *TriggerInfo

	// 重置定时器
	ResetClock()
	// 生成下次触发时间, 如果返回了 false 表示没有下一次了, 返回的时间一定>t
	MakeNextTriggerTime(t time.Time) (time.Time, bool)
}

type TriggerInfo struct {
	// 触发器类型
	TriggerType TriggerType
	// 表达式
	Expression string
	// 下一次触发时间
	NextTriggerTime time.Time
}

// -------------- cron触发器 --------------------

// cron触发器
type CronTrigger struct {
	expression      string
	schedule        cron.Schedule
	nextExecuteTime time.Time
	mx              sync.Mutex // 用于锁 nextExecuteTime
}

// 创建一个cron触发器
func NewCronTrigger(expression string) ITrigger {
	schedule, err := cron.ParseStandard(expression)
	if err != nil {
		panic(fmt.Errorf("expression syntax error, %s", err))
	}

	return &CronTrigger{
		expression:      expression,
		schedule:        schedule,
		nextExecuteTime: schedule.Next(time.Now()),
	}
}

func (c *CronTrigger) TriggerType() TriggerType {
	return CronTriggerType
}
func (c *CronTrigger) Expression() string {
	return c.expression
}
func (c *CronTrigger) TriggerInfo() *TriggerInfo {
	c.mx.Lock()
	nextExecuteTime := c.nextExecuteTime
	c.mx.Unlock()

	return &TriggerInfo{
		TriggerType:     CronTriggerType,
		Expression:      c.expression,
		NextTriggerTime: nextExecuteTime,
	}
}

func (c *CronTrigger) ResetClock() {
	c.mx.Lock()
	c.nextExecuteTime = c.schedule.Next(time.Now())
	c.mx.Unlock()
}
func (c *CronTrigger) MakeNextTriggerTime(t time.Time) (time.Time, bool) {
	c.mx.Lock()
	for t.Unix() >= c.nextExecuteTime.Unix() {
		c.nextExecuteTime = c.schedule.Next(c.nextExecuteTime)
	}
	t = c.nextExecuteTime
	c.mx.Unlock()
	return t, true
}

// --------------- 一次性触发器 -------------------

// 一次性触发器
type OnceTrigger struct {
	expression  string
	executeTime time.Time
}

func NewOnceTrigger(t time.Time) ITrigger {
	o := &OnceTrigger{
		expression:  t.Format(zutils.Time.Layout),
		executeTime: t,
	}
	return o
}

func (o *OnceTrigger) TriggerType() TriggerType {
	return OnceTriggerType
}
func (o *OnceTrigger) Expression() string {
	return o.expression
}
func (o *OnceTrigger) TriggerInfo() *TriggerInfo {
	return &TriggerInfo{
		TriggerType:     OnceTriggerType,
		Expression:      o.expression,
		NextTriggerTime: o.executeTime,
	}
}

func (o *OnceTrigger) ResetClock() {
}
func (o *OnceTrigger) MakeNextTriggerTime(t time.Time) (time.Time, bool) {
	if t.Unix() < o.executeTime.Unix() {
		return o.executeTime, true
	}
	return t, false
}

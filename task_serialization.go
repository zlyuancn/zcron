/*
-------------------------------------------------
   Author :       Zhang Fan
   date：         2020/10/28
   Description :
-------------------------------------------------
*/

package zcron

import (
	"fmt"
	"time"

	jsoniter "github.com/json-iterator/go"
	"github.com/robfig/cron/v3"
)

const TimeLayout = "2006-01-02 15:04:05"

type serializeTaskConfig struct {
	Name    string `json:"name"`
	Trigger struct {
		TriggerType string `json:"trigger_type"` // 触发器类型
		Expression  string `json:"expression"`   // 触发器表达式
	} `json:"trigger"` // 触发器
	Executor struct {
		MaxConcurrentExecuteCount int64 `json:"max_concurrent_count"` // 最大并发执行数
		MaxRetryCount             int64 `json:"max_retry_count"`      // 最大重试数
		RetryInterval             int64 `json:"retry_interval"`       // 重试间隔(毫秒)
	} `json:"executor"` // 执行器
}

// 序列化Task
func SerializeTask(task ITask) string {
	taskInfo := task.TaskInfo()
	stc := &serializeTaskConfig{
		Name: taskInfo.Name,
	}

	stc.Trigger.TriggerType = taskInfo.TriggerInfo.TriggerType.String()
	stc.Trigger.Expression = taskInfo.TriggerInfo.Expression

	stc.Executor.MaxConcurrentExecuteCount = taskInfo.ExecutorInfo.MaxConcurrentExecuteCount
	stc.Executor.MaxRetryCount = taskInfo.ExecutorInfo.MaxRetryCount
	stc.Executor.RetryInterval = int64(taskInfo.ExecutorInfo.RetryInterval / time.Millisecond)

	text, _ := jsoniter.MarshalToString(stc)
	return text
}

// 从序列化文本中加载Task
func NewTaskOfSerializeText(configText string, enable bool, handler Handler) (ITask, error) {
	stc := new(serializeTaskConfig)
	err := jsoniter.UnmarshalFromString(configText, stc)
	if err != nil {
		return nil, fmt.Errorf("串行化数据解析失败, %s", err)
	}

	var trigger ITrigger
	switch stc.Trigger.TriggerType {
	case CronTriggerType.String():
		_, err := cron.ParseStandard(stc.Trigger.Expression)
		if err != nil {
			return nil, fmt.Errorf("trigger expression syntax error, %s", err)
		}
		trigger = NewCronTrigger(stc.Trigger.Expression)
	case OnceTriggerType.String():
		t, err := time.ParseInLocation(TimeLayout, stc.Trigger.Expression, time.Local)
		if err != nil {
			return nil, fmt.Errorf("trigger expression syntax error, %s", err)
		}
		trigger = NewOnceTrigger(t)
	default:
		return nil, fmt.Errorf("undefined trigger type: %s", stc.Trigger.TriggerType)
	}

	executor := NewExecutor(stc.Executor.MaxRetryCount, time.Duration(stc.Executor.RetryInterval)*time.Millisecond, stc.Executor.MaxConcurrentExecuteCount)

	taskConfig := TaskConfig{
		Trigger:  trigger,
		Executor: executor,
		Handler:  handler,
		Enable:   enable,
	}
	return NewTaskOfConfig(stc.Name, taskConfig), nil
}

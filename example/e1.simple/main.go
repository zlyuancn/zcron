/*
-------------------------------------------------
   Author :       Zhang Fan
   date：         2020/11/3
   Description :
-------------------------------------------------
*/

package main

import (
	"fmt"
	"time"

	"github.com/zlyuancn/zcron"
)

func main() {
	cron := zcron.NewCron() // 创建一个定时器

	// 创建一个任务
	task := zcron.NewTask("task", "@every 1s", true, func(ctx zcron.IContext) (err error) {
		fmt.Println("执行")
		return nil
	})

	cron.AddTask(task) // 将任务添加到定时器
	cron.Start()       // 启动定时器
	cron.RemoveTask(task.Name())

	<-time.After(5 * time.Second) // 等待5秒后结束
	cron.Stop()
}

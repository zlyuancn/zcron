# 朴实无华的定时器

---

[toc]

---

# 功能列表

+ task

- cron定时器
- 一次性定时器
- 重试
- 作业panic拦截
- 并发执行数控制(线程数限制)
- 详细的执行信息
- 详细的task信息
- 可暂停恢复
- 序列化

+ cron

- 全局并发执行数控制(线程数限制)
- 任务列表
- 详细的cron信息
- 全局可暂停恢复
- 日志记录

+ 其他

# 获得

`go get -u github.com/zlyuancn/zcron`

# 概念说明

## 上下文(context)

> 任务在执行时的上下文环境, 每次执行会生成一个新的上下文

## 执行器(executor)

> 执行器控制如何去执行, 每一个task有一个自己的执行器, 他可以控制任务执行并发数限制, 它还捕获了任务panic导致的错误

## 触发器(trigger)

> 用于决定task在什么时候执行, 每一个task有一个自己的触发器

## 任务(task)

> 任务实体, 表示一个完整的工作内容

## 定时器(cron)

> 通常一个应用只有一个定时器, 用于管理多个task.
> 定时器内部使用 64 个最小堆保存任务列表, 每一个刻度会映射到最小堆上.

# 使用说明

```go
cron := zcron.NewCron() // 创建一个定时器

// 创建一个任务
task := zcron.NewTask("task", "@every 1s", true, func(ctx zcron.IContext) (err error) {
    fmt.Println("执行")
    return nil
})

cron.AddTask(task) // 将任务添加到定时器
cron.Start() // 启动定时器

<-time.After(5 * time.Second) // 等待5秒后结束
cron.Stop()
```

# 教程

+ [1.简单示例](example/e1.simple/main.go)
+ 待续...

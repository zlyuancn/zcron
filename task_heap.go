package zcron

type ITaskHeap interface {
	// 堆排序
	Sort()
	// 弹出第一个
	Pop() (task ITask)
	// 压入
	Push(task ITask)
	// 移除
	Remove(task ITask)
	// 获取任务列表
	Tasks() []ITask
}

type TaskHeap struct {
	tasks []ITask
}

func NewTaskHeap(tasks ...ITask) ITaskHeap {
	return &TaskHeap{tasks}
}

func (t *TaskHeap) Sort() {
	for i, task := range t.tasks {
		task.setHeapIndex(i)
	}

	n := len(t.tasks)
	for i := n/2 - 1; i >= 0; i-- {
		t.down(i, n)
	}
}

func (t *TaskHeap) Pop() (task ITask) {
	n := len(t.tasks) - 1
	t.swap(0, n)
	t.down(0, n)

	t.tasks, task = t.tasks[:n], t.tasks[n]
	return
}

func (t *TaskHeap) Push(task ITask) {
	task.setHeapIndex(len(t.tasks))
	t.tasks = append(t.tasks, task)
	t.up(len(t.tasks) - 1)
}

func (t *TaskHeap) Remove(task ITask) {
	index := task.getHeapIndex()
	if t.tasks[index] != task { // 检查
		return
	}

	n := len(t.tasks) - 1
	if n != index {
		t.swap(index, n)
		if !t.down(index, n) {
			t.up(index)
		}
	}
	t.tasks = t.tasks[:n]
}

func (t *TaskHeap) Tasks() []ITask {
	return t.tasks
}

func (t *TaskHeap) less(i, j int) bool {
	return t.tasks[i].TriggerTime().Before(t.tasks[j].TriggerTime())
}

func (t *TaskHeap) swap(i, j int) {
	t.tasks[i], t.tasks[j] = t.tasks[j], t.tasks[i]
	t.tasks[i].setHeapIndex(i)
	t.tasks[j].setHeapIndex(j)
}

func (t *TaskHeap) up(j int) {
	for {
		i := (j - 1) / 2 // parent
		if i == j || !t.less(j, i) {
			break
		}
		t.swap(i, j)
		j = i
	}
}

func (t *TaskHeap) down(i0, n int) bool {
	i := i0
	for {
		j1 := 2*i + 1
		if j1 >= n || j1 < 0 { // j1 < 0 after int overflow
			break
		}
		j := j1 // left child
		if j2 := j1 + 1; j2 < n && t.less(j2, j1) {
			j = j2 // = 2*i + 2  // right child
		}
		if !t.less(j, i) {
			break
		}
		t.swap(i, j)
		i = j
	}
	return i > i0
}

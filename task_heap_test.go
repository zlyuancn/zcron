package zcron

import (
	"testing"
	"time"
)

func makeTestTaskData() []ITask {
	return []ITask{
		&Task{name: "4", triggerTime: time.Date(2020, 1, 1, 0, 4, 0, 0, time.Local)},
		&Task{name: "9", triggerTime: time.Date(2020, 1, 1, 0, 9, 0, 0, time.Local)},
		&Task{name: "7", triggerTime: time.Date(2020, 1, 1, 0, 7, 0, 0, time.Local)},
		&Task{name: "2", triggerTime: time.Date(2020, 1, 1, 0, 2, 0, 0, time.Local)},
		&Task{name: "6", triggerTime: time.Date(2020, 1, 1, 0, 6, 0, 0, time.Local)},
		&Task{name: "6", triggerTime: time.Date(2020, 1, 1, 0, 6, 0, 0, time.Local)},
		&Task{name: "5", triggerTime: time.Date(2020, 1, 1, 0, 5, 0, 0, time.Local)},
		&Task{name: "3", triggerTime: time.Date(2020, 1, 1, 0, 3, 0, 0, time.Local)},
	}
}

func TestTaskHeap_Empty(t *testing.T) {
	tasks := makeTestTaskData()
	heap := NewTaskHeap()
	for _, task := range tasks {
		heap.Push(task)
	}

	var expectNames = []string{"2", "3", "4", "5", "6", "6", "7", "9"}
	for _, name := range expectNames {
		if heap.Pop().Name() != name {
			t.Fatal("顺序和预期不符")
		}
	}
}

func TestTaskHeap_Sort(t *testing.T) {
	tasks := makeTestTaskData()
	heap := NewTaskHeap(tasks...)
	heap.Sort()
	var expectNames = []string{"2", "3", "4", "5", "6", "6", "7", "9"}
	for _, name := range expectNames {
		if heap.Pop().Name() != name {
			t.Fatal("顺序和预期不符")
		}
	}
}

func TestTaskHeap_Push(t *testing.T) {
	tasks := makeTestTaskData()

	heap := NewTaskHeap(tasks...)
	heap.Sort()

	task := &Task{name: "8", triggerTime: time.Date(2020, 1, 1, 0, 8, 0, 0, time.Local)}
	heap.Push(task)

	var expectNames = []string{"2", "3", "4", "5", "6", "6", "7", "8", "9"}
	for _, name := range expectNames {
		if heap.Pop().Name() != name {
			t.Fatal("顺序和预期不符")
		}
	}
}

func TestTaskHeap_Remove(t *testing.T) {
	tasks := makeTestTaskData()

	heap := NewTaskHeap(tasks...)
	heap.Sort()
	heap.Remove(tasks[4])

	var expectNames = []string{"2", "3", "4", "5", "6", "7", "9"}
	for _, name := range expectNames {
		if heap.Pop().Name() != name {
			t.Fatal("顺序和预期不符")
		}
	}
}

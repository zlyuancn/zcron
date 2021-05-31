package zcron

import (
	"fmt"
	"testing"
)

func TestTask(t *testing.T) {
	task := NewTask("test", "@every 1s", true, func(ctx IContext) (err error) {
		fmt.Println("触发")
		return nil
	})
	task.Trigger(nil)
}

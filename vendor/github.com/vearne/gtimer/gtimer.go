package gtimer

import "time"

var (
	defaultTimer *SuperTimer
)

func init() {
	defaultTimer = NewSuperTimer(10)
}

func Set(timer *SuperTimer) {
	defaultTimer.Stop()
	defaultTimer = timer
}

func Add(d time.Duration, f func()) *Item {
	task := NewDelayedItemFunc(time.Now().Add(d), nil, func(t time.Time, i interface{}) {
		f()
	})
	defaultTimer.Add(task)
	return task
}

func AddComplex(scheduledExecTime time.Time, f func(scheduledExecTime time.Time, param interface{}),
	param interface{}) *Item {
	task := NewDelayedItemFunc(scheduledExecTime, param, f)
	defaultTimer.Add(task)
	return task
}

func Remove(task *Item) {
	defaultTimer.Remove(task)
}

func Stop() {
	defaultTimer.Stop()
}

func Wait() {
	defaultTimer.Wait()
}

func Size() int {
	return defaultTimer.Size()
}

package gtimer

import (
	"time"
)

type Delayed interface {
	GetDelay() time.Duration
}

//nolint: govet
// An Item is something we manage in a priority queue.
type Item struct {
	// when task is ready, execute OnTrigger function
	OnTrigger func(scheduledExecTime time.Time, param interface{})
	priority  int64 // The priority of the item in the queue.
	// The index is needed by update and is maintained by the heap.Interface methods.
	index int         // The index of the item in the heap.
	value interface{} // The value of the item; arbitrary.
}

// triggerTime is time of the task should be execute
func NewDelayedItemFunc(triggerTime time.Time, param interface{}, f func(time.Time, interface{})) *Item {
	item := Item{}
	item.priority = triggerTime.UnixNano()
	item.value = param
	item.OnTrigger = f
	return &item
}

func (item *Item) GetDelay() time.Duration {
	return time.Duration(item.priority - time.Now().UnixNano())
}

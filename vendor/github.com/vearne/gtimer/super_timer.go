package gtimer

import (
	"container/heap"
	"sync"
	"time"
)

//nolint: govet
type SuperTimer struct {
	Wgp         *sync.WaitGroup
	lock        *sync.RWMutex
	UniTimer    *time.Timer
	WorkerCount int
	ExitChan    chan int
	PQ          PriorityQueue
	RunningFlag bool
}

func NewSuperTimer(workCount int) *SuperTimer {
	timer := SuperTimer{}
	timer.lock = &sync.RWMutex{}
	timer.UniTimer = time.NewTimer(time.Second * 10)
	timer.PQ = make(PriorityQueue, 0, 100)
	timer.WorkerCount = workCount
	timer.Wgp = &sync.WaitGroup{}
	timer.ExitChan = make(chan int, 100)
	timer.RunningFlag = true

	for i := 0; i < timer.WorkerCount; i++ {
		go timer.Consume()
	}
	return &timer
}

func (timer *SuperTimer) Consume() {
	timer.Wgp.Add(1)
	defer timer.Wgp.Done()
	for timer.RunningFlag {
		select {
		case <-timer.ExitChan:
			break
		case <-timer.UniTimer.C:
			item := timer.Take()
			if item != nil {
				t := time.Unix(item.priority/int64(time.Second), item.priority%int64(time.Second))
				item.OnTrigger(t, item.value)
			}
		}
	}
}

func (st *SuperTimer) Add(pItem *Item) {
	st.lock.Lock()
	defer st.lock.Unlock()
	heap.Push(&st.PQ, pItem)
	peek := st.PQ.Peek().(*Item)
	if peek == pItem {
		st.UniTimer.Reset(pItem.GetDelay())
	}
}

func (st *SuperTimer) Remove(pItem *Item) {
	st.lock.Lock()
	defer st.lock.Unlock()
	st.PQ.Remove(pItem)
}

func (st *SuperTimer) Take() *Item {
	st.lock.Lock()
	defer st.lock.Unlock()
	if len(st.PQ) <= 0 {
		st.UniTimer.Reset(time.Second * 1)
		return nil
	}

	item := st.PQ.Peek()
	target := item.(*Item)
	if target.GetDelay() > 0 {
		st.UniTimer.Reset(target.GetDelay())
		return nil
	}

	res := heap.Pop(&st.PQ).(*Item)
	// 重置定时器，立刻唤醒其它消费者
	st.UniTimer.Reset(0)
	return res
}

func (st *SuperTimer) Stop() {
	st.lock.Lock()
	defer st.lock.Unlock()
	st.PQ.Clear()
	st.RunningFlag = false
	close(st.ExitChan)
	st.UniTimer.Stop()
}

func (st *SuperTimer) Wait() {
	st.Wgp.Wait()
}

func (st *SuperTimer) Size() int {
	st.lock.RLock()
	defer st.lock.RUnlock()

	return len(st.PQ)
}

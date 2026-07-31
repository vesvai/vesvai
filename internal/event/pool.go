package event

import (
	"sync"
	"time"
)

var (
	deadEventPool = sync.Pool{New: func() interface{} { return &DeadEvent{} }}
)

func GetDeadEvent() *DeadEvent {
	e := deadEventPool.Get().(*DeadEvent)
	e.timestamp = time.Now()
	e.subscribers = e.subscribers[:0]
	return e
}

func PutDeadEvent(e *DeadEvent) {
	e.Event = nil
	e.subscribers = e.subscribers[:0]
	deadEventPool.Put(e)
}

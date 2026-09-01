package event

import (
	"fmt"
	"reflect"
	"sync"
)

type BusSubscriber interface {
	Subscribe(topic string, fn interface{}) error
	SubscribeAsync(topic string, fn interface{}, transactional bool) error
	SubscribeOnce(topic string, fn interface{}) error
	SubscribeOnceAsync(topic string, fn interface{}) error
	Unsubscribe(topic string, handler interface{}) error
}

type BusPublisher interface {
	Publish(topic string, args ...interface{})
}

type BusController interface {
	HasCallback(topic string) bool
	WaitAsync()
}

type Bus interface {
	BusController
	BusSubscriber
	BusPublisher
}

type EventBus struct {
	handlers map[string][]*eventHandler
	lock     sync.Mutex
	pending  int
	cond     *sync.Cond
}

type eventHandler struct {
	callBack      reflect.Value
	flagOnce      bool
	async         bool
	transactional bool
	sync.Mutex
}

func New() Bus {
	b := &EventBus{handlers: make(map[string][]*eventHandler)}
	b.cond = sync.NewCond(&b.lock)
	return Bus(b)
}

func (bus *EventBus) doSubscribe(topic string, fn interface{}, handler *eventHandler) error {
	t := reflect.TypeOf(fn)
	if t == nil {
		return fmt.Errorf("nil is not of type reflect.Func")
	}
	if t.Kind() != reflect.Func {
		return fmt.Errorf("%s is not of type reflect.Func", t.Kind())
	}
	bus.lock.Lock()
	defer bus.lock.Unlock()
	bus.handlers[topic] = append(bus.handlers[topic], handler)
	return nil
}

func (bus *EventBus) Subscribe(topic string, fn interface{}) error {
	return bus.doSubscribe(topic, fn, &eventHandler{
		reflect.ValueOf(fn), false, false, false, sync.Mutex{},
	})
}

func (bus *EventBus) SubscribeAsync(topic string, fn interface{}, transactional bool) error {
	return bus.doSubscribe(topic, fn, &eventHandler{
		reflect.ValueOf(fn), false, true, transactional, sync.Mutex{},
	})
}

func (bus *EventBus) SubscribeOnce(topic string, fn interface{}) error {
	return bus.doSubscribe(topic, fn, &eventHandler{
		reflect.ValueOf(fn), true, false, false, sync.Mutex{},
	})
}

func (bus *EventBus) SubscribeOnceAsync(topic string, fn interface{}) error {
	return bus.doSubscribe(topic, fn, &eventHandler{
		reflect.ValueOf(fn), true, true, false, sync.Mutex{},
	})
}

func (bus *EventBus) HasCallback(topic string) bool {
	bus.lock.Lock()
	defer bus.lock.Unlock()
	_, ok := bus.handlers[topic]
	if ok {
		return len(bus.handlers[topic]) > 0
	}
	return false
}

func (bus *EventBus) Unsubscribe(topic string, handler interface{}) error {
	bus.lock.Lock()
	defer bus.lock.Unlock()
	if _, ok := bus.handlers[topic]; ok && len(bus.handlers[topic]) > 0 {
		bus.removeHandler(topic, bus.findHandlerIdx(topic, reflect.ValueOf(handler)))
		return nil
	}
	return fmt.Errorf("topic %s doesn't exist", topic)
}

func (bus *EventBus) Publish(topic string, args ...interface{}) {
	bus.lock.Lock()
	handlers, ok := bus.handlers[topic]
	if !ok || len(handlers) == 0 {
		bus.lock.Unlock()
		return
	}

	toFire := make([]*eventHandler, len(handlers))
	copy(toFire, handlers)

	survivors := make([]*eventHandler, 0, len(handlers))
	removedOnce := false
	for _, h := range handlers {
		if h.flagOnce {
			removedOnce = true
			continue
		}
		survivors = append(survivors, h)
	}
	if removedOnce {
		if len(survivors) == 0 {
			delete(bus.handlers, topic)
		} else {
			bus.handlers[topic] = survivors
		}
	}
	bus.lock.Unlock()

	for _, handler := range toFire {
		if !handler.async {
			bus.doPublish(handler, topic, args...)
			continue
		}
		bus.lock.Lock()
		bus.pending++
		bus.lock.Unlock()
		if handler.transactional {
			handler.Lock()
		}
		go bus.doPublishAsync(handler, topic, args...)
	}
}

func (bus *EventBus) doPublish(handler *eventHandler, topic string, args ...interface{}) {
	passedArguments := bus.setUpPublish(handler, args...)
	handler.callBack.Call(passedArguments)
}

func (bus *EventBus) doPublishAsync(handler *eventHandler, topic string, args ...interface{}) {
	defer bus.asyncDone()
	if handler.transactional {
		defer handler.Unlock()
	}
	bus.doPublish(handler, topic, args...)
}

func (bus *EventBus) asyncDone() {
	bus.lock.Lock()
	bus.pending--
	if bus.pending == 0 {
		bus.cond.Broadcast()
	}
	bus.lock.Unlock()
}

func (bus *EventBus) removeHandler(topic string, idx int) {
	if _, ok := bus.handlers[topic]; !ok {
		return
	}
	l := len(bus.handlers[topic])

	if !(0 <= idx && idx < l) {
		return
	}

	copy(bus.handlers[topic][idx:], bus.handlers[topic][idx+1:])
	bus.handlers[topic][l-1] = nil
	bus.handlers[topic] = bus.handlers[topic][:l-1]
}

func (bus *EventBus) findHandlerIdx(topic string, callback reflect.Value) int {
	if _, ok := bus.handlers[topic]; ok {
		for idx, handler := range bus.handlers[topic] {
			if handler.callBack.Type() == callback.Type() &&
				handler.callBack.Pointer() == callback.Pointer() {
				return idx
			}
		}
	}
	return -1
}

func (bus *EventBus) setUpPublish(callback *eventHandler, args ...interface{}) []reflect.Value {
	funcType := callback.callBack.Type()
	passedArguments := make([]reflect.Value, len(args))
	for i, v := range args {
		if v == nil {
			passedArguments[i] = reflect.New(funcType.In(i)).Elem()
		} else {
			passedArguments[i] = reflect.ValueOf(v)
		}
	}

	return passedArguments
}

func (bus *EventBus) WaitAsync() {
	bus.lock.Lock()
	defer bus.lock.Unlock()
	for bus.pending > 0 {
		bus.cond.Wait()
	}
}

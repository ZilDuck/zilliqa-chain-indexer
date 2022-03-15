package event

import (
	"go.uber.org/zap"
)

var listeners = make([]*Listener, 0)

type Listener struct {
	eventType Type
	channel   chan interface{}
}

func AddEventListener(eventType Type, callback func(msg interface{})) {
	zap.L().With(zap.String("type", string(eventType))).Debug("EventManager: AddListener")

	listener := Listener{
		eventType: eventType,
		channel:   make(chan interface{}),
	}

	listeners = append(listeners, &listener)

	go func() {
		for {
			msg := <-listener.channel
			callback(msg)
		}
	}()
}

func EmitEvent(eventType Type, msg interface{}) {
	for _, listener := range listeners {
		if listener.eventType == eventType {
			zap.L().With(zap.String("type", string(eventType))).Debug("EventManager: Emitting event")
			go func(handler chan interface{}) {
				handler <- msg
			}(listener.channel)
		}
	}
}
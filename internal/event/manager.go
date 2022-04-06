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
	if len(listeners) == 0 {
		zap.L().Debug("No event listeners available")
	}
	for _, listener := range listeners {
		zap.L().Debug(string(listener.eventType))
		if listener.eventType == eventType {
			zap.L().With(zap.String("type", string(eventType))).Debug("EventManager: Emitting event")
			go func(handler chan interface{}) {
				handler <- msg
			}(listener.channel)
		}
	}
}

package messenger

import "github.com/streadway/amqp"

type exchange struct {
	Name        string
	Type        string
	Durable     bool
	AutoDeleted bool
	Internal    bool
	NoWait      bool
	Arguments   amqp.Table
}

var exchanges = map[string]exchange{
	"metadata.refresh": {
		Name:        "metadata.refresh",
		Type:        "topic",
		Durable:     true,
		AutoDeleted: false,
		Internal:    false,
		NoWait:      true,
		Arguments:   nil,
	},
}


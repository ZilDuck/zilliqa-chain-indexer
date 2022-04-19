package messenger

import (
	"errors"
	"fmt"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/config"
	"github.com/streadway/amqp"
	"go.uber.org/zap"
)

type MessageService interface {
	GetQueue(item Item) (*amqp.Queue, error)
	SendMessage(item Item, body []byte, reliable bool) error
	ConsumeMessages(item Item, callback func(msg string)) error
	GetQueueSize(item Item) (*int, error)
}

type Messenger struct {
	amqpUri string
	conn    *amqp.Connection
	network string
}

type Item string

var (
	MetadataRefresh Item = "metadata.refresh"
)

func (i Item) queue() string {
	return fmt.Sprintf("%s.%s", config.Get().Index, i)
}

func NewMessenger(amqpUri string) MessageService {
	return &Messenger{amqpUri: amqpUri}
}

func (m Messenger) GetQueue(item Item) (*amqp.Queue, error) {
	ch, err := m.openChannel()
	if err != nil {
		return nil, err
	}

	queue, err := ch.QueueDeclare(item.queue(), true, false, false, false, nil)
	if err != nil {
		zap.L().With(zap.Error(err), zap.String("queue", item.queue())).Error("[Queue] Failed to create queue")
		return nil, err
	}

	return &queue, nil
}

func (m Messenger) SendMessage(item Item, body []byte, reliable bool) error {
	ch, err := m.openChannel()
	if err != nil {
		return err
	}

	ex, ok := exchanges[string(item)]
	if !ok {
		zap.L().Error("[Queue] Exchange not found")
		return errors.New("exchange not found")
	}

	if err := ch.ExchangeDeclare(ex.Name, ex.Type, ex.Durable, ex.AutoDeleted, ex.Internal, ex.NoWait, ex.Arguments); err != nil {
		zap.L().With(zap.Error(err)).Error("[Queue] Exchange Declare")
		return err
	}

	if reliable {
		if err := ch.Confirm(false); err != nil {
			zap.L().With(zap.Error(err)).Error("[Queue] Channel could not be put into confirm mode")
			return err
		}

		confirms := ch.NotifyPublish(make(chan amqp.Confirmation, 1))

		defer m.confirmOne(confirms)
	}

	publishing := amqp.Publishing{
		Headers:         amqp.Table{},
		ContentType:     "text/json",
		ContentEncoding: "",
		Body:            body,
		DeliveryMode:    amqp.Transient,
		Priority:        0,
	}

	if err = ch.Publish(ex.Name, item.queue(), false, false, publishing); err != nil {
		zap.L().With(zap.Error(err)).Error("[Queue] Exchange Publish")
		return err
	}

	zap.L().With(zap.String("exchange", ex.Name), zap.String("routingKey", item.queue())).Info("[Queue] Published message")

	return err
}

func (m Messenger) ConsumeMessages(item Item, callback func(msg string)) error {
	ch, err := m.openChannel()
	if err != nil {
		return err
	}

	ex, ok := exchanges[string(item)]
	if !ok {
		return errors.New("exchange not found")
	}

	if err := ch.ExchangeDeclare(ex.Name, ex.Type, ex.Durable, ex.AutoDeleted, ex.Internal, ex.NoWait, ex.Arguments); err != nil {
		zap.L().With(zap.Error(err)).Error("[Queue] Exchange Declare")
		return err
	}

	q, err := ch.QueueDeclare(item.queue(), true, false, false, false, nil)
	if err != nil {
		zap.L().With(zap.Error(err)).Error("[Queue] Failed to declare a queue")
		return err
	}

	err = ch.QueueBind(q.Name, "", ex.Name, false, nil)
	if err != nil {
		zap.L().With(zap.Error(err)).Error("[Queue] Failed to bind a queue")
		return err
	}

	msgs, err := ch.Consume(q.Name, "", true, false, false, false, nil)
	if err != nil {
		zap.L().With(zap.Error(err)).Error("[Queue] Failed to consume the queue")
		return err
	}

	forever := make(chan bool)

	go func() {
		for d := range msgs {
			zap.L().Debug("[Queue] Received message")
			callback(string(d.Body))
		}
	}()

	zap.S().With(zap.String("exchange", ex.Name)).Debugf("[Queue] Waiting for messages")
	<-forever

	return nil
}

func (m Messenger) GetQueueSize(item Item) (*int, error) {
	queue, err := m.GetQueue(item)
	if err != nil {
		return nil, err
	}

	return &queue.Messages, nil
}

func (m Messenger) openConnection() (*amqp.Connection, error) {
	if m.conn != nil && !m.conn.IsClosed() {
		return m.conn, nil
	}

	conn, err := amqp.Dial(m.amqpUri)
	if err != nil {
		zap.L().With(zap.Error(err)).Error("[Queue] Failed to connect to RabbitMQ")
		return nil, err
	}

	m.conn = conn

	return m.conn, nil
}

func (m Messenger) openChannel() (*amqp.Channel, error) {
	conn, err := m.openConnection()
	if err != nil {
		return nil, err
	}

	ch, err := conn.Channel()
	if err != nil {
		zap.S().With(zap.Error(err)).Error("[Queue] Failed to open channel")
	}

	return ch, err
}

func (m Messenger) confirmOne(confirms <-chan amqp.Confirmation) {
	zap.L().Debug("[Queue] Waiting for publish confirmation")

	if confirmed := <-confirms; confirmed.Ack {
		zap.L().Debug("[Queue] Publish confirmed")
	} else {
		zap.L().Debug("[Queue] Publish failed")
	}
}
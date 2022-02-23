package messenger

import (
	"fmt"
	"github.com/ZilDuck/zilliqa-chain-indexer/internal/config"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sqs"
	"go.uber.org/zap"
)

type MessageService interface {
	CreateQueue(queue Queue) (*string, error)
	SendMessage(queue Queue, body []byte) error
	PollMessages(queue Queue, chn chan <- *sqs.Message)
	DeleteMessage(queue Queue, msg *sqs.Message) error
}

type Messenger struct {
	sqsClient *sqs.SQS
}

type Queue string

const (
	MetadataRefresh Queue = "metadata_refresh"
	AssetRefresh Queue = "asset_refresh"
)

func (q *Queue) Get() string {
	return fmt.Sprintf("%s_%s", *q, config.Get().Index)
}

func NewMessenger(sqsClient *sqs.SQS) MessageService {
	return &Messenger{sqsClient}
}

func (m Messenger) CreateQueue(queue Queue) (*string, error) {
	queueName := queue.Get()
	result, err := m.sqsClient.CreateQueue(&sqs.CreateQueueInput{
		QueueName: &queueName,
		Attributes: map[string]*string{
			"DelaySeconds":      aws.String("0"),
			"VisibilityTimeout": aws.String("30"),
		},
	})
	if err != nil {
		zap.L().With(zap.Error(err), zap.String("queue", queueName)).Error("Failed to create queue")
		return nil, err
	}


	return result.QueueUrl, nil
}

func (m Messenger) SendMessage(queue Queue, body []byte) error {
	queueName := queue.Get()

	zap.L().With(zap.String("queue", queueName), zap.String("body", string(body))).Info("Send Message")
	queueUrl, err := m.getQueueUrl(queue)
	if err != nil {
		return err
	}

	_, err = m.sqsClient.SendMessage(&sqs.SendMessageInput{
		QueueUrl:    queueUrl,
		MessageBody: aws.String(string(body)),
	})

	return err
}

func (m Messenger) PollMessages(queue Queue, chn chan <- *sqs.Message) {
	queueUrl, err := m.getQueueUrl(queue)
	if err != nil {
		return
	}

	for {
		output, err := m.sqsClient.ReceiveMessage(&sqs.ReceiveMessageInput{
			QueueUrl: queueUrl,
			MaxNumberOfMessages: aws.Int64(10),
			WaitTimeSeconds:     aws.Int64(15),
		})

		if err != nil {
			zap.L().With(zap.Error(err)).Warn("Failed to fetch message")
			return
		}

		for _, message := range output.Messages {
			chn <- message
		}
	}
}



func (m Messenger) DeleteMessage(queue Queue, msg *sqs.Message) error {
	queueUrl, err := m.getQueueUrl(queue)
	if err != nil {
		return err
	}

	_, err = m.sqsClient.DeleteMessage(&sqs.DeleteMessageInput{
		QueueUrl:      queueUrl,
		ReceiptHandle: msg.ReceiptHandle,
	})

	return err
}

func (m Messenger) getQueueUrl(queue Queue) (*string, error) {
	queueName := queue.Get()
	result, err := m.sqsClient.GetQueueUrl(&sqs.GetQueueUrlInput{QueueName: &queueName})
	if err != nil {
		zap.L().With(zap.Error(err), zap.String("queue", queueName)).Error("Failed to get queue url")
		return nil, err
	}

	return result.QueueUrl, nil
}

package broker

import (
	"context"
	"fmt"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

type ProducerParam struct {
	ConnectionStr string
	AppId         string
	QueueName     string
	MessageId     string
	Data          []byte
	Expiry        time.Duration
}

type ConsumerParam struct {
	ConnectionStr string
	AppId         string
	QueueName     string
	Handler       func(msg amqp.Delivery) error
}

// connection
func connection(connectionStr string) (*amqp.Channel, error) {
	connection, err := amqp.Dial(connectionStr)
	if err != nil {
		return nil, err
	}
	defer connection.Close()

	channel, err := connection.Channel()
	if err != nil {
		return nil, err
	}
	defer channel.Close()

	return channel, nil
}

// Producer -
func Producer(param ProducerParam) error {
	channel, err := connection(param.ConnectionStr)
	if err != nil {
		return nil
	}
	queue, err := channel.QueueDeclare(param.QueueName, true, false, false, false, nil)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(10*time.Second))
	defer cancel()

	err = channel.PublishWithContext(
		ctx,
		"",         // exchange
		queue.Name, // routing key
		false,      // mandatory
		false,      // immediate
		amqp.Publishing{
			AppId:        param.AppId,
			MessageId:    param.MessageId,
			ContentType:  "application/json",
			Body:         param.Data,
			DeliveryMode: amqp.Persistent,
			Expiration:   fmt.Sprint(param.Expiry),
		})
	if err != nil {
		return err
	}

	return nil
}

// Consumer -
func Consumer(param ConsumerParam) error {
	channel, err := connection(param.ConnectionStr)
	if err != nil {
		return nil
	}
	queue, err := channel.QueueDeclare(param.QueueName, true, false, false, false, nil)
	if err != nil {
		return err
	}

	msgs, err := channel.Consume(
		queue.Name, // queue
		"",         // consumer
		false,      // auto-ack
		false,      // exclusive
		false,      // no-local
		false,      // no-wait
		nil,        // args
	)
	if err != nil {
		return err
	}

	go func() {
		for msg := range msgs {
			if msg.AppId == param.AppId {
				if err = param.Handler(msg); err != nil {
					_ = msg.Nack(false, true)
				} else {
					_ = msg.Ack(false)
				}
			}
			_ = msg.Nack(false, true)
		}
	}()

	return nil
}

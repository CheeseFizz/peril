package pubsub

import (
	"bytes"
	"context"
	"encoding/gob"
	"encoding/json"
	"log"

	amqp "github.com/rabbitmq/amqp091-go"
)

type SimpleQueueType int

const (
	Durable SimpleQueueType = iota
	Transient
)

type AckType int

const (
	Ack AckType = iota
	NackRequeue
	NackDiscard
)

func PublishJSON[T any](ch *amqp.Channel, exchange, key string, val T) error {
	jsonData, err := json.Marshal(val)
	if err != nil {
		return err
	}

	message := amqp.Publishing{
		ContentType: "application/json",
		Body:        jsonData,
	}

	err = ch.PublishWithContext(
		context.Background(),
		exchange,
		key,
		false,
		false,
		message,
	)
	if err != nil {
		return err
	}

	return nil
}

func PublishGob[T any](ch *amqp.Channel, exchange, key string, val T) error {
	var gobData bytes.Buffer
	enc := gob.NewEncoder(&gobData)
	err := enc.Encode(val)
	if err != nil {
		return err
	}

	message := amqp.Publishing{
		ContentType: "application/gob",
		Body:        gobData.Bytes(),
	}

	err = ch.PublishWithContext(
		context.Background(),
		exchange,
		key,
		false,
		false,
		message,
	)
	if err != nil {
		return err
	}

	return nil
}

func DeclareAndBind(
	conn *amqp.Connection,
	exchange,
	queueName,
	key string,
	queueType SimpleQueueType, // an enum to represent "durable" or "transient"
) (*amqp.Channel, amqp.Queue, error) {

	amqp_chan, err := conn.Channel()
	if err != nil {
		return nil, amqp.Queue{}, err
	}

	var dur bool
	var ad bool
	var exc bool
	if queueType == Durable {
		dur = true
		ad = false
		exc = false
	} else {
		dur = false
		ad = true
		exc = true
	}

	amqp_table := amqp.Table{}
	amqp_table["x-dead-letter-exchange"] = "peril_dlx"

	q, err := amqp_chan.QueueDeclare(
		queueName,
		dur,
		ad,
		exc,
		false,
		amqp_table,
	)
	if err != nil {
		return nil, amqp.Queue{}, err
	}

	err = amqp_chan.QueueBind(queueName, key, exchange, false, nil)
	if err != nil {
		return nil, amqp.Queue{}, err
	}

	return amqp_chan, q, nil
}

func subscribe[T any](
	conn *amqp.Connection,
	exchange,
	queueName,
	key string,
	simpleQueueType SimpleQueueType,
	handler func(T) AckType,
	unmarshaller func([]byte) (T, error),
) error {

	return nil
}

func SubscribeJSON[T any](
	conn *amqp.Connection,
	exchange,
	queueName,
	key string,
	queueType SimpleQueueType, // an enum to represent "durable" or "transient"
	handler func(T) AckType,
) (*amqp.Channel, error) {
	amqp_chan, amqp_queue, err := DeclareAndBind(conn, exchange, queueName, key, queueType)
	if err != nil {
		return nil, err
	}

	delivery_chan, err := amqp_chan.Consume(amqp_queue.Name, "", false, false, false, false, nil)

	go func() {
		for m := range delivery_chan {
			data := new(T)
			err := json.Unmarshal(m.Body, data)
			if err != nil {
				log.Println(err)
			}

			a := handler(*data)
			switch a {
			case Ack:
				log.Println("Responding Ack")
				err = m.Ack(false)
				if err != nil {
					log.Println(err)
				}
			case NackDiscard:
				log.Println("Responding NackDiscard")
				err = m.Nack(false, false)
				if err != nil {
					log.Println(err)
				}
			case NackRequeue:
				log.Println("Responding NackDiscard")
				err = m.Nack(false, true)
				if err != nil {
					log.Println(err)
				}
			}
		}
	}()

	return amqp_chan, nil
}

func SubscribeGob[T any](
	conn *amqp.Connection,
	exchange,
	queueName,
	key string,
	queueType SimpleQueueType, // an enum to represent "durable" or "transient"
	handler func(T) AckType,
) (*amqp.Channel, error) {
	amqp_chan, amqp_queue, err := DeclareAndBind(conn, exchange, queueName, key, queueType)
	if err != nil {
		return nil, err
	}

	delivery_chan, err := amqp_chan.Consume(amqp_queue.Name, "", false, false, false, false, nil)

	go func() {
		for m := range delivery_chan {
			data := new(T)
			var buf bytes.Buffer

			dec := gob.NewDecoder(&buf)
			err := json.Unmarshal(m.Body, data)
			if err != nil {
				log.Println(err)
			}

			a := handler(*data)
			switch a {
			case Ack:
				log.Println("Responding Ack")
				err = m.Ack(false)
				if err != nil {
					log.Println(err)
				}
			case NackDiscard:
				log.Println("Responding NackDiscard")
				err = m.Nack(false, false)
				if err != nil {
					log.Println(err)
				}
			case NackRequeue:
				log.Println("Responding NackDiscard")
				err = m.Nack(false, true)
				if err != nil {
					log.Println(err)
				}
			}
		}
	}()

	return amqp_chan, nil
}

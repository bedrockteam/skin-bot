package utils

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/sirupsen/logrus"
)

type MQ struct {
	conn       *amqp.Connection
	channel    *amqp.Channel
	skin_queue amqp.Queue
	pub_queue  amqp.Queue
	err        chan *amqp.Error

	uri         string
	want_pubsub bool

	connect_lock *sync.Mutex
}

func NewQueue(want_pubsub bool) *MQ {
	return &MQ{
		connect_lock: &sync.Mutex{},
		want_pubsub:  want_pubsub,
	}
}

func (q *MQ) Start(url string) error {
	q.uri = url
	q.Reconnect()
	return nil
}

// Reconnect connect to the message broker
func (q *MQ) Reconnect() {
	q.connect_lock.Lock()
	defer q.connect_lock.Unlock()
	if q.conn != nil {
		return
	}

	var err error
	for {
		if err != nil {
			logrus.Errorf("Error Connecting to RabbitMQ %s", err)
			logrus.Info("Trying to reconnect to RabbitMQ at %s", q.uri)
			time.Sleep(10 * time.Second)
		}

		q.conn, err = amqp.Dial(q.uri)
		if err != nil {
			continue
		}
		q.err = make(chan *amqp.Error)
		q.conn.NotifyClose(q.err)

		q.channel, err = q.conn.Channel()
		if err != nil {
			continue
		}
		q.skin_queue, err = q.channel.QueueDeclare("player_skins", false, false, false, true, nil)
		if err != nil {
			continue
		}

		if q.want_pubsub {
			err = q.channel.ExchangeDeclare("new_skins", "fanout", false, false, false, true, nil)
			if err != nil {
				continue
			}
		}

		break
	}
}

// AquireConn makes sure there is a connection
func (q *MQ) AquireConn() {
	select { // non blocking channel - if there is no error will go to default where we do nothing
	case err := <-q.err:
		if err != nil {
			q.conn = nil
			q.Reconnect()
		}
	default:
	}
}

// process_message decodes a message to skin
func process_message(d *amqp.Delivery) (*QueuedSkin, error) {
	body := d.Body
	if d.ContentType == "application/json-gz" {
		r, err := gzip.NewReader(bytes.NewBuffer(body))
		if err != nil {
			return nil, err
		}
		body, err = io.ReadAll(r)
		if err != nil {
			return nil, err
		}
	}

	var skin QueuedSkin
	if err := json.Unmarshal(body, &skin); err != nil {
		return nil, err
	}

	return &skin, nil
}

// ReceiveSkins receives skins to a channel
func (q *MQ) ReceiveSkins() (chan *QueuedSkin, error) {
	q.AquireConn()

	msgs, err := q.channel.Consume("player_skins", "", true, false, false, false, nil)
	if err != nil {
		return nil, err
	}

	ch := make(chan *QueuedSkin)

	go func() {
		for d := range msgs {
			skin, err := process_message(&d)
			if err != nil {
				logrus.Errorf("Error processing message %s", err)
				continue
			}
			ch <- skin
		}
	}()

	return ch, err
}

// PubSubSkin puts skin to the pubsub
func (q *MQ) PubSubSkin(ctx context.Context) error {
	q.AquireConn()

	err := q.channel.PublishWithContext(ctx, q.pub_queue.Name, "", false, false, amqp.Publishing{
		ContentType: "application/json",
		Body:        []byte("a"),
	})
	if err != nil {
		return fmt.Errorf("error PubSub: %s", err)
	}

	logrus.Debug("Pub/Sub skin published")
	return nil
}

// PublishSkin publishes a skin to the skin queue
func (q *MQ) PublishSkin(ctx context.Context, skin *QueuedSkin) error {
	q.AquireConn()

	body, _ := json.Marshal(skin)
	buf := bytes.NewBuffer(nil)
	w := gzip.NewWriter(buf)
	w.Write(body)
	w.Close()

	err := q.channel.PublishWithContext(ctx, "", q.skin_queue.Name, false, false, amqp.Publishing{
		ContentType: "application/json-gz",
		Body:        buf.Bytes(),
	})
	if err != nil {
		return fmt.Errorf("error Publishing: %s", err)
	}
	return nil
}

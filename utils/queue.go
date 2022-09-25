package utils

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/sirupsen/logrus"
)

type MQ struct {
	conn               *amqp.Connection
	channel            *amqp.Channel
	skin_queue         amqp.Queue
	isConnected        bool
	done               chan bool
	inital_connect_err chan error
	reopen             chan bool
	notifyClose        chan *amqp.Error

	uri         string
	want_pubsub bool

	had_success bool
}

func NewQueue(uri string, want_pubsub bool) *MQ {
	q := &MQ{
		done:               make(chan bool),
		reopen:             make(chan bool),
		inital_connect_err: make(chan error),
		want_pubsub:        want_pubsub,
		uri:                uri,
	}
	go q.handleReconnect()
	return q
}

var reconnectDelay = 5 * time.Second

func (q *MQ) handleReconnect() {
	for {
		q.isConnected = false
		logrus.Info("Attempting to connect to amqp")
		for {
			if err := q.connect(); err != nil {
				logrus.Errorf("AMQP: %s", err)
				if !q.had_success {
					q.inital_connect_err <- err
				}
				logrus.Info("Failed to connect. Retrying...")
				time.Sleep(reconnectDelay)
				continue
			}
			break
		}

		logrus.Info("Connected to RabbitMQ")

		if !q.had_success {
			q.had_success = true
			q.inital_connect_err <- nil
		}

		// wait for it to close and retry
		// or end and exit
		select {
		case <-q.done:
			return
		case <-q.notifyClose:
		}
	}
}

// Reconnect connect to the message broker
func (q *MQ) connect() error {
	if q.conn != nil && !q.conn.IsClosed() {
		return nil
	}

	conn, err := amqp.Dial(q.uri)
	if err != nil {
		return err
	}
	ch, err := conn.Channel()
	if err != nil {
		return err
	}
	ch.Confirm(false)

	return q.changeConnection(conn, ch)
}

func (q *MQ) changeConnection(connection *amqp.Connection, channel *amqp.Channel) error {
	q.conn = connection
	q.channel = channel

	var err error
	q.skin_queue, err = q.channel.QueueDeclare("player_skins", false, false, false, true, nil)
	if err != nil {
		return err
	}

	if q.want_pubsub {
		err = q.channel.ExchangeDeclare("new_skins", "fanout", false, false, false, true, nil)
		if err != nil {
			return err
		}
	}

	q.notifyClose = make(chan *amqp.Error)
	q.conn.NotifyClose(q.notifyClose)
	q.isConnected = true
	close(q.reopen)
	q.reopen = make(chan bool)
	return nil
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
func (q *MQ) ReceiveSkins() chan *QueuedSkin {
	ch := make(chan *QueuedSkin)

	go func() {
		for {
			if !q.isConnected {
				<-q.reopen
			}
			msgs, err := q.channel.Consume("player_skins", "", true, false, false, false, nil)
			if err != nil {
				logrus.Warn(err)
				continue
			}

			for d := range msgs {
				skin, err := process_message(&d)
				if err != nil {
					logrus.Errorf("Error processing message %s", err)
					continue
				}
				ch <- skin
			}
		}
	}()
	return ch
}

// PubSubSkin puts skin to the pubsub
func (q *MQ) PubSubSkin(ctx context.Context, username, xuid, skin *DBSkinListItem) error {
	if !q.isConnected {
		<-q.reopen
	}

	data, err := json.Marshal(struct {
		Username string          `json:"username"`
		Xuid     string          `json:"xuid"`
		Skin     *DBSkinListItem `json:"skin"`
	}{username, xuid, skin})
	if err != nil {
		return err
	}

	err := q.channel.PublishWithContext(ctx, "new_skins", "", false, false, amqp.Publishing{
		ContentType: "application/json",
		Body:        data,
	})
	if err != nil {
		q.conn = nil
		return fmt.Errorf("error PubSub: %s", err)
	}

	logrus.Debug("Pub/Sub skin published")
	return nil
}

// PublishSkin publishes a skin to the skin queue
func (q *MQ) PublishSkin(ctx context.Context, skin *QueuedSkin) error {
	body, _ := json.Marshal(skin)
	buf := bytes.NewBuffer(nil)
	w := gzip.NewWriter(buf)
	w.Write(body)
	w.Close()

	for {
		// wait for the connection
		if !q.isConnected {
			<-q.reopen
		}

		err := q.channel.PublishWithContext(ctx, "", q.skin_queue.Name, false, false, amqp.Publishing{
			ContentType: "application/json-gz",
			Body:        buf.Bytes(),
		})
		if err != nil {
			logrus.Warnf("Publishing: %s", err)
			continue
		}
		break
	}
	return nil
}

func (q *MQ) Close() {
	q.done <- true
	if q.channel != nil {
		q.channel.Close()
	}
	if q.conn != nil {
		q.conn.Close()
	}
}

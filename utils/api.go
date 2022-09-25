package utils

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

type Metrics interface {
	// Start starts the metric pusher
	Start(url, user, password string) error
	// deletes from pusher
	Delete()
}

type Queue interface {
	Start(url string) error
}

type apiClient struct {
	server string
	key    string

	client *http.Client

	Queue   *MQ
	Metrics Metrics
}

var APIClient *apiClient

// InitAPIClient creates the api client
func InitAPIClient(APIServer, APIKey string, metrics Metrics, queue *MQ) error {
	APIClient = &apiClient{
		server:  APIServer,
		key:     APIKey,
		client:  &http.Client{},
		Queue:   queue,
		Metrics: metrics,
	}
	return nil
}

func (u *apiClient) doRequest(req *http.Request) (resp *http.Response, err error) {
	req.Header.Set("Authorization", u.key)
	return u.client.Do(req)
}

// Start starts the api client
func (u *apiClient) Start() error {
	var response struct {
		AMQPUrl           string
		PrometheusPushURL string
		PrometheusAuth    string
	}

	req, _ := http.NewRequest("GET", APIClient.server+"/routes", nil)
	resp, err := APIClient.doRequest(req)
	if err != nil {
		return nil
	}

	if resp.StatusCode != 200 {
		return fmt.Errorf("API StatusCode: %d", resp.StatusCode)
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return err
	}

	// rabbitmq
	if u.Queue != nil {
		if err := u.Queue.Start(response.AMQPUrl); err != nil {
			return err
		}
	}

	// prometheus
	if u.Metrics != nil {
		auth := strings.Split(response.PrometheusAuth, ":")
		if err := u.Metrics.Start(response.PrometheusPushURL, auth[0], auth[1]); err != nil {
			return err
		}
	}

	return nil
}

var c = 0

// UploadSkin pushes a skin to the message server
func (u *apiClient) UploadSkin(ctx context.Context, skin *Skin, username, xuid string, serverAddress string) {
	c += 1
	logrus.Infof("Uploading Skin %s %s %d", serverAddress, username, c)

	err := u.Queue.PublishSkin(ctx, &QueuedSkin{
		username,
		xuid,
		skin.Json(),
		serverAddress,
		time.Now().Unix(),
	})
	if err != nil {
		logrus.Warn(err)
	}
}

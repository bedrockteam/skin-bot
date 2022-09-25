package main

import (
	"context"
	"flag"
	"fmt"
	"math/rand"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/bedrockteam/skin-bot/utils"
	"github.com/disgoorg/dislog"
	"github.com/disgoorg/snowflake"
	"golang.org/x/exp/maps"

	"github.com/sirupsen/logrus"
)

type Config struct {
	API struct {
		Server string
		Key    string
	}
	Discord struct {
		WebhookId    string
		WebhookToken string
	}
	Users []struct {
		Name    string
		Address string
	}
	ServerAddresses string
	ServerBlacklist []string
}

// ip -> bot
var bots_lock = &sync.Mutex{}
var bots = make(map[string]*Bot)

// ip -> time to retry
var ip_waitlist = make(map[string]time.Time)

func main() {
	logrus.SetLevel(logrus.DebugLevel)

	ctx, cancel := context.WithCancel(context.Background())
	flag.BoolVar(&utils.G_debug, "debug", false, "debug mode")
	flag.Parse()

	// exit cleanup
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigs
		println("cancel")
		cancel()
	}()

	var config Config

	{
		if _, err := os.Stat("config.toml"); err == nil {
			_, err := toml.DecodeFile("config.toml", &config)
			if err != nil {
				logrus.Fatal(err)
			}
		}

		{ // save config
			f, _ := os.Create("config.toml")
			defer f.Close()
			if err := toml.NewEncoder(f).Encode(&config); err != nil {
				logrus.Fatal(err)
			}
		}

		if config.API.Server == "" {
			logrus.Fatal("API.Server undefined")
		}
		if config.API.Key == "" {
			logrus.Fatal("API.Key undefined")
		}
		if len(config.Users) == 0 {
			logrus.Warn("No Users defined")
		}

		if config.Discord.WebhookId != "" {
			logrus.Info("Enabling discord Error logs")
			dlog, err := dislog.New(
				// Sets which logging levels to send to the webhook
				dislog.WithLogLevels(dislog.ErrorLevelAndAbove...),
				// Sets webhook id & token
				dislog.WithWebhookIDToken(snowflake.Snowflake(config.Discord.WebhookId), config.Discord.WebhookToken),
			)
			if err != nil {
				logrus.Fatal("error initializing dislog: ", err)
			}
			defer dlog.Close(ctx)
			logrus.StandardLogger().AddHook(dlog)
		}
	}

	{ // setup api client
		if err := utils.InitAPIClient(config.API.Server, config.API.Key, NewMetrics()); err != nil {
			logrus.Fatal(err)
		}
		if err := utils.APIClient.Start(false); err != nil {
			logrus.Fatal(err)
		}
	}
	defer utils.APIClient.Close()

	// starting the bots
	for {
		servers := strings.Split(config.ServerAddresses, " ")

		user := config.Users[rand.Intn(len(config.Users))]
		for _, server := range servers {
			if ctx.Err() != nil {
				break
			}

			if !strings.Contains(server, ":") {
				server = server + ":19132"
			}
			var IPs []string
			var err error
			for {
				// lookup all instances of this server
				IPs, err = utils.FindAllIps(server)
				if err != nil {
					logrus.Errorf("Failed to lookup ips %s", err)
					select {
					case <-time.After(30 * time.Second):
					case <-ctx.Done():
					}
					continue
				}
				break
			}

			// connect to all ips that dont have an instance yet
			count := 0
			for _, ip := range IPs {
				_address := ip + ":19132"

				if w, ok := ip_waitlist[_address]; ok {
					if time.Now().After(w) {
						delete(ip_waitlist, _address)
					} else {
						continue
					}
				}

				if _, ok := bots[_address]; ok {
					continue
				}
				b := NewBot(user.Name, _address, fmt.Sprintf("%s %s", server, ip))
				go b.Start(ctx)
				count += 1
			}
			time.Sleep(1 * time.Second)
			if count > 0 {
				logrus.Infof("Started %d Bots as %s on %s", count, user.Name, server)
				logrus.Infof("Instances: %d", len(maps.Keys(bots)))
			}
			logrus.Infof("Waiting: %d", len(maps.Keys(ip_waitlist)))
		}

		select {
		case <-time.After(5 * time.Second):
		case <-ctx.Done():
		}

		if ctx.Err() != nil {
			break
		}
	}
}

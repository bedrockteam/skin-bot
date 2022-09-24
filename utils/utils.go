package utils

import (
	"context"
	"encoding/json"
	"net"
	"regexp"
	"strings"
	"time"

	"github.com/sandertv/gophertunnel/minecraft"
	"github.com/sirupsen/logrus"

	//"github.com/sandertv/gophertunnel/minecraft/gatherings"

	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

var (
	G_debug         bool
	G_cleanup_funcs []func() = []func(){}
)

var name_regexp = regexp.MustCompile(`\||(?:§.?)`)

// cleans name so it can be used as a filename
func CleanupName(name string) string {
	name = strings.Split(name, "\n")[0]
	var _tmp struct {
		K string `json:"k"`
	}
	err := json.Unmarshal([]byte(name), &_tmp)
	if err == nil {
		name = _tmp.K
	}
	name = string(name_regexp.ReplaceAll([]byte(name), []byte("")))
	name = strings.TrimSpace(name)
	return name
}

// connections

type (
	PacketFunc func(header packet.Header, payload []byte, src, dst net.Addr)
)

func ConnectServer(ctx context.Context, address, clientName string, packetFunc PacketFunc) (serverConn *minecraft.Conn, err error) {
	var local_addr net.Addr
	packet_func := func(header packet.Header, payload []byte, src, dst net.Addr) {
		if G_debug {
			PacketLogger(header, payload, src, dst, local_addr)
			if packetFunc != nil {
				packetFunc(header, payload, src, dst)
			}
		}
	}

	key, chainData, err := GetChain(clientName)
	if err != nil {
		return nil, err
	}

	logrus.Infof("Connecting to %s", address)
	serverConn, err = minecraft.Dialer{
		TokenSource:   GetTokenSource(clientName),
		PacketFunc:    packet_func,
		DownloadPacks: false,
		Key:           key,
		ChainData:     chainData,
	}.DialContext(ctx, "raknet", address)
	if err != nil {
		return nil, err
	}

	local_addr = serverConn.LocalAddr()
	return serverConn, nil
}

func FindAllIps(Address string) (ret []string, err error) {
	host, _, err := net.SplitHostPort(Address)
	if err != nil {
		return nil, err
	}
	tmp := map[string]bool{}
	for i := 0; i < 5; i++ {
		time.Sleep(1 * time.Second)
		ips, err := net.LookupHost(host)
		if err != nil {
			return nil, err
		}
		for _, v := range ips {
			tmp[v] = true
		}
	}
	for k := range tmp {
		ret = append(ret, k)
	}
	return ret, nil
}

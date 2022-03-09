package control

import (
	"context"
	"math/rand"
	"net"
	"os"
	"path/filepath"
	"time"

	"github.com/pkg/errors"
	"github.com/wangxiaochuang/k3s/pkg/cluster"
	"github.com/wangxiaochuang/k3s/pkg/daemons/config"
	"github.com/wangxiaochuang/k3s/pkg/daemons/control/deps"
)

var localhostIP = net.ParseIP("127.0.0.1")

func Server(ctx context.Context, cfg *config.Control) error {
	rand.Seed(time.Now().UTC().UnixNano())
	runtime := cfg.Runtime

	if err := prepare(ctx, cfg, runtime); err != nil {
		return errors.Wrap(err, "preparing server")
	}

	return errors.New("in Server")
}

func defaults(config *config.Control) {
	if config.ClusterIPRange == nil {
		_, clusterIPNet, _ := net.ParseCIDR("10.42.0.0/16")
		config.ClusterIPRange = clusterIPNet
	}

	if config.ServiceIPRange == nil {
		_, serviceIPNet, _ := net.ParseCIDR("10.43.0.0/16")
		config.ServiceIPRange = serviceIPNet
	}

	if len(config.ClusterDNS) == 0 {
		config.ClusterDNS = net.ParseIP("10.43.0.10")
	}

	if config.AdvertisePort == 0 {
		config.AdvertisePort = config.HTTPSPort
	}

	if config.APIServerPort == 0 {
		if config.HTTPSPort != 0 {
			config.APIServerPort = config.HTTPSPort + 1
		} else {
			config.APIServerPort = 6444
		}
	}

	if config.DataDir == "" {
		config.DataDir = "./management-state"
	}
}

func prepare(ctx context.Context, config *config.Control, runtime *config.ControlRuntime) error {
	var err error

	defaults(config)

	if err := os.MkdirAll(config.DataDir, 0700); err != nil {
		return err
	}

	config.DataDir, err = filepath.Abs(config.DataDir)
	if err != nil {
		return err
	}

	os.MkdirAll(filepath.Join(config.DataDir, "tls"), 0700)
	os.MkdirAll(filepath.Join(config.DataDir, "cred"), 0700)

	deps.CreateRuntimeCertFiles(config, runtime)

	cluster := cluster.New(config)

	if err := cluster.Bootstrap(ctx, false); err != nil {
		return err
	}

	if err := deps.GenServerDeps(config, runtime); err != nil {
		return err
	}

	ready, err := cluster.Start(ctx)
	if err != nil {
		return err
	}

	runtime.ETCDReady = ready
	runtime.EtcdConfig = cluster.EtcdConfig

	return nil
}

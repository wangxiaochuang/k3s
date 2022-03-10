package cluster

import (
	"context"
	"strings"

	"github.com/k3s-io/kine/pkg/endpoint"
	"github.com/pkg/errors"
	"github.com/wangxiaochuang/k3s/pkg/clientaccess"
	"github.com/wangxiaochuang/k3s/pkg/cluster/managed"
	"github.com/wangxiaochuang/k3s/pkg/daemons/config"
)

type Cluster struct {
	clientAccessInfo *clientaccess.Info
	config           *config.Control
	runtime          *config.ControlRuntime
	managedDB        managed.Driver
	EtcdConfig       endpoint.ETCDConfig
	joining          bool
	storageStarted   bool
	saveBootstrap    bool
	shouldBootstrap  bool
}

func (c *Cluster) Start(ctx context.Context) (<-chan struct{}, error) {
	if err := c.initClusterAndHTTPS(ctx); err != nil {
		return nil, errors.Wrap(err, "init cluster datastore and https")
	}
	if c.config.DisableETCD {
		panic("in cluster start")
	}

	// start managed database (if necessary)
	if err := c.start(ctx); err != nil {
		return nil, errors.Wrap(err, "start managed database")
	}

	ready, err := c.testClusterDB(ctx)
	if err != nil {
		return nil, err
	}

	if err := c.startStorage(ctx); err != nil {
		return nil, err
	}

	if c.saveBootstrap {
		if err := Save(ctx, c.config, c.EtcdConfig, false); err != nil {
			return nil, err
		}
	}

	if c.managedDB != nil {
		panic("in cluster Start")
	}
	return ready, nil
}

func (c *Cluster) startStorage(ctx context.Context) error {
	if c.storageStarted {
		return nil
	}
	c.storageStarted = true

	// 启动了unix监听服务
	etcdConfig, err := endpoint.Listen(ctx, c.config.Datastore)
	if err != nil {
		return errors.Wrap(err, "creating storage endpoint")
	}

	c.EtcdConfig = etcdConfig
	c.config.Datastore.BackendTLSConfig = etcdConfig.TLSConfig
	c.config.Datastore.Endpoint = strings.Join(etcdConfig.Endpoints, ",")
	c.config.NoLeaderElect = !etcdConfig.LeaderElect
	return nil
}

// New creates an initial cluster using the provided configuration.
func New(config *config.Control) *Cluster {
	return &Cluster{
		config:  config,
		runtime: config.Runtime,
	}
}

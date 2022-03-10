package cluster

import (
	"context"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/k3s-io/kine/pkg/endpoint"
	"github.com/sirupsen/logrus"
	"github.com/wangxiaochuang/k3s/pkg/cluster/managed"
	"github.com/wangxiaochuang/k3s/pkg/etcd"
	"github.com/wangxiaochuang/k3s/pkg/nodepassword"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
)

func (c *Cluster) testClusterDB(ctx context.Context) (<-chan struct{}, error) {
	result := make(chan struct{})
	if c.managedDB == nil {
		close(result)
		return result, nil
	}
	panic("in testClusterDB")
}
func (c *Cluster) start(ctx context.Context) error {
	resetFile := etcd.ResetFile(c.config)
	if c.managedDB == nil {
		return nil
	}

	switch {
	case c.config.ClusterReset && c.config.ClusterResetRestorePath != "":
		panic("in cluster start reset and 1")
	case c.config.ClusterReset:
		panic("in cluster start reset")
	}

	if _, err := os.Stat(resetFile); err == nil {
		// before removing reset file we need to delete the node passwd secret
		go c.deleteNodePasswdSecret(ctx)
	}
	// removing the reset file and ignore error if the file doesn't exist
	os.Remove(resetFile)

	return c.managedDB.Start(ctx, c.clientAccessInfo)
}

func (c *Cluster) initClusterDB(ctx context.Context, handler http.Handler) (http.Handler, error) {
	if c.managedDB == nil {
		return handler, nil
	}

	if !strings.HasPrefix(c.config.Datastore.Endpoint, c.managedDB.EndpointName()+"://") {
		c.config.Datastore = endpoint.Config{
			Endpoint: c.managedDB.EndpointName(),
		}
	}

	return c.managedDB.Register(ctx, c.config, handler)
}

func (c *Cluster) assignManagedDriver(ctx context.Context) error {
	for _, driver := range managed.Registered() {
		// not ok and err is nil
		if ok, err := driver.IsInitialized(ctx, c.config); err != nil {
			return err
		} else if ok {
			c.managedDB = driver
			return nil
		}
	}

	// 只注册了一个etcd的driver，也没有初始化
	endpointType := strings.SplitN(c.config.Datastore.Endpoint, ":", 2)[0]
	// endpointType没有设置，就是""
	for _, driver := range managed.Registered() {
		if endpointType == driver.EndpointName() {
			c.managedDB = driver
			return nil
		}
	}

	if c.config.Datastore.Endpoint == "" && (c.config.ClusterInit || (c.config.Token != "" && c.config.JoinURL != "")) {
		for _, driver := range managed.Registered() {
			if driver.EndpointName() == managed.Default() {
				c.managedDB = driver
				return nil
			}
		}
	}
	// 没有endpoint，也不是集群初始化，也没配token和JoinURL
	return nil
}

func (c *Cluster) deleteNodePasswdSecret(ctx context.Context) {
	t := time.NewTicker(5 * time.Second)
	defer t.Stop()
	for range t.C {
		nodeName := os.Getenv("NODE_NAME")
		if nodeName == "" {
			logrus.Infof("waiting for node name to be set")
			continue
		}

		if c.runtime.Core == nil {
			logrus.Infof("runtime is not yet initialized")
			continue
		}

		secretsClient := c.runtime.Core.Core().V1().Secret()
		if err := nodepassword.Delete(secretsClient, nodeName); err != nil {
			if apierrors.IsNotFound(err) {
				logrus.Debugf("node password secret is not found for node %s", nodeName)
				return
			}
			logrus.Warnf("failed to delete old node password secret: %v", err)
			continue
		}
		return
	}
}

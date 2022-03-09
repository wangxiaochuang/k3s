package cluster

import (
	"context"
	"crypto/tls"
	"errors"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"

	"github.com/rancher/dynamiclistener"
	"github.com/rancher/dynamiclistener/factory"
	"github.com/rancher/dynamiclistener/storage/file"
	"github.com/rancher/dynamiclistener/storage/kubernetes"
	"github.com/rancher/dynamiclistener/storage/memory"
	"github.com/rancher/wrangler/pkg/generated/controllers/core"
	"github.com/sirupsen/logrus"
	"github.com/wangxiaochuang/k3s/pkg/daemons/config"
	"github.com/wangxiaochuang/k3s/pkg/etcd"
	"github.com/wangxiaochuang/k3s/pkg/version"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (c *Cluster) newListener(ctx context.Context) (net.Listener, http.Handler, error) {
	if c.managedDB != nil {
		if _, err := os.Stat(etcd.ResetFile(c.config)); err == nil {
			// delete the dynamic listener file if it exists after restoration to fix restoration
			// on fresh nodes
			os.Remove(filepath.Join(c.config.DataDir, "tls/dynamic-cert.json"))
		}
	}
	tcp, err := dynamiclistener.NewTCPListener(c.config.BindAddress, c.config.SupervisorPort)
	if err != nil {
		return nil, nil, err
	}
	cert, key, err := factory.LoadCerts(c.runtime.ServerCA, c.runtime.ServerCAKey)
	if err != nil {
		return nil, nil, err
	}
	storage := tlsStorage(ctx, c.config.DataDir, c.runtime)
	return dynamiclistener.NewListener(tcp, storage, cert, key, dynamiclistener.Config{
		ExpirationDaysCheck: config.CertificateRenewDays,
		Organization:        []string{version.Program},
		SANs:                append(c.config.SANs, "kubernetes", "kubernetes.default", "kubernetes.default.svc", "kubernetes.default.svc."+c.config.ClusterDomain),
		CN:                  version.Program,
		TLSConfig: &tls.Config{
			ClientAuth:   tls.RequestClientCert,
			MinVersion:   c.config.TLSMinVersion,
			CipherSuites: c.config.TLSCipherSuites,
		},
		RegenerateCerts: func() bool {
			const regenerateDynamicListenerFile = "dynamic-cert-regenerate"
			dynamicListenerRegenFilePath := filepath.Join(c.config.DataDir, "tls", regenerateDynamicListenerFile)
			if _, err := os.Stat(dynamicListenerRegenFilePath); err == nil {
				os.Remove(dynamicListenerRegenFilePath)
				return true
			}
			return false
		},
	})
}

func (c *Cluster) initClusterAndHTTPS(ctx context.Context) error {
	listener, handler, err := c.newListener(ctx)
	if err != nil {
		return err
	}

	handler, err = c.getHandler(handler)
	if err != nil {
		return err
	}

	handler, err = c.initClusterDB(ctx, handler)
	if err != nil {
		return err
	}

	server := http.Server{
		Handler: handler,
	}

	if logrus.IsLevelEnabled(logrus.DebugLevel) {
		server.ErrorLog = log.New(logrus.StandardLogger().Writer(), "Cluster-Http-Server ", log.LstdFlags)
	} else {
		server.ErrorLog = log.New(ioutil.Discard, "Cluster-Http-Server", 0)
	}

	go func() {
		err := server.Serve(listener)
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			logrus.Fatalf("server stopped: %v", err)
		}
	}()

	go func() {
		<-ctx.Done()
		server.Shutdown(context.Background())
	}()

	return nil
}

func tlsStorage(ctx context.Context, dataDir string, runtime *config.ControlRuntime) dynamiclistener.TLSStorage {
	fileStorage := file.New(filepath.Join(dataDir, "tls/dynamic-cert.json"))
	cache := memory.NewBacked(fileStorage)
	return kubernetes.New(ctx, func() *core.Factory {
		return runtime.Core
	}, metav1.NamespaceSystem, version.Program+"-serving", cache)
}

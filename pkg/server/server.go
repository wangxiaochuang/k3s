package server

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"github.com/wangxiaochuang/k3s/pkg/cli/cmds"
	"github.com/wangxiaochuang/k3s/pkg/daemons/config"
	"github.com/wangxiaochuang/k3s/pkg/daemons/control"
	"github.com/wangxiaochuang/k3s/pkg/datadir"
	"github.com/wangxiaochuang/k3s/pkg/util"
)

const (
	MasterRoleLabelKey       = "node-role.kubernetes.io/master"
	ControlPlaneRoleLabelKey = "node-role.kubernetes.io/control-plane"
	ETCDRoleLabelKey         = "node-role.kubernetes.io/etcd"
)

func ResolveDataDir(dataDir string) (string, error) {
	dataDir, err := datadir.Resolve(dataDir)
	return filepath.Join(dataDir, "server"), err
}

func StartServer(ctx context.Context, config *Config, cfg *cmds.Server) error {
	if err := setupDataDirAndChdir(&config.ControlConfig); err != nil {
		return err
	}

	if err := setNoProxyEnv(&config.ControlConfig); err != nil {
		return err
	}

	if err := control.Server(ctx, &config.ControlConfig); err != nil {
		return errors.Wrap(err, "starting kubernetes")
	}

	return errors.New("xxxxxxx")
}

func setupDataDirAndChdir(config *config.Control) error {
	var (
		err error
	)

	config.DataDir, err = ResolveDataDir(config.DataDir)
	if err != nil {
		return err
	}

	dataDir := config.DataDir

	if err := os.MkdirAll(dataDir, 0700); err != nil {
		return errors.Wrapf(err, "can not mkdir %s", dataDir)
	}

	if err := os.Chdir(dataDir); err != nil {
		return errors.Wrapf(err, "can not chdir %s", dataDir)
	}

	return err
}

func setNoProxyEnv(config *config.Control) error {
	splitter := func(c rune) bool {
		return c == ','
	}
	envList := []string{}
	envList = append(envList, strings.FieldsFunc(os.Getenv("NO_PROXY"), splitter)...)
	envList = append(envList, strings.FieldsFunc(os.Getenv("no_proxy"), splitter)...)
	envList = append(envList,
		".svc",
		"."+config.ClusterDomain,
		util.JoinIPNets(config.ClusterIPRanges),
		util.JoinIPNets(config.ServiceIPRanges),
	)
	os.Unsetenv("no_proxy")
	return os.Setenv("NO_PROXY", strings.Join(envList, ","))
}

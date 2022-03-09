package datadir

import (
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/rancher/wrangler/pkg/resolvehome"
	"github.com/wangxiaochuang/k3s/pkg/version"
)

var (
	DefaultDataDir     = "/var/lib/rancher/" + version.Program
	DefaultHomeDataDir = "${HOME}/.rancher/" + version.Program
	HomeConfig         = "${HOME}/.kube/" + version.Program + ".yaml"
	GlobalConfig       = "/etc/rancher/" + version.Program + "/" + version.Program + ".yaml"
)

func Resolve(dataDir string) (string, error) {
	return LocalHome(dataDir, false)
}

func LocalHome(dataDir string, forceLocal bool) (string, error) {
	if dataDir == "" {
		if os.Getuid() == 0 && !forceLocal {
			dataDir = DefaultDataDir
		} else {
			dataDir = DefaultHomeDataDir
		}
	}

	dataDir, err := resolvehome.Resolve(dataDir)
	if err != nil {
		return "", errors.Wrapf(err, "resolving %s", dataDir)
	}

	return filepath.Abs(dataDir)
}

package configfilearg

import (
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
	"github.com/wangxiaochuang/k3s/pkg/cli/cmds"
	"github.com/wangxiaochuang/k3s/pkg/version"
)

var DefaultParser = &Parser{
	After:         []string{"server", "agent", "etcd-snapshot:1"},
	FlagNames:     []string{"--config", "-c"},
	EnvName:       version.ProgramUpper + "_CONFIG_FILE",
	DefaultConfig: "/etcd/rancher/" + version.Program + "/config.yaml",
	ValidFlags:    map[string][]cli.Flag{"server": cmds.ServerFlags, "etcd-snapshot": cmds.EtcdSnapshotFlags},
}

func MustParse(args []string) []string {
	result, err := DefaultParser.Parse(args)
	if err != nil {
		logrus.Fatal(err)
	}
	return result
}

func MustFindString(args []string, target string) string {
	parser := &Parser{
		After:         []string{},
		FlagNames:     []string{},
		EnvName:       version.ProgramUpper + "_CONFIG_FILE",
		DefaultConfig: "/etc/rancher/" + version.Program + "/config.yaml",
	}
	result, err := parser.FindString(args, target)
	if err != nil {
		logrus.Fatal(err)
	}
	return result
}

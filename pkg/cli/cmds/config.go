package cmds

import (
	"github.com/urfave/cli"
	"github.com/wangxiaochuang/k3s/pkg/version"
)

var (
	ConfigFlag = cli.StringFlag{
		Name:   "config,c",
		Usage:  "(config) Load configuration from `FILE`",
		EnvVar: version.ProgramUpper + "_CONFIG_FILE",
		Value:  "/etc/rancher/" + version.Program + "/config.yaml",
	}
)

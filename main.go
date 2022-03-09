package main

import (
	"context"
	"errors"
	"os"

	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
	"github.com/wangxiaochuang/k3s/pkg/cli/cmds"
	"github.com/wangxiaochuang/k3s/pkg/cli/server"
	"github.com/wangxiaochuang/k3s/pkg/configfilearg"
)

func main() {
	app := cmds.NewApp()
	app.Commands = []cli.Command{
		cmds.NewServerCommand(server.Run),
	}

	if err := app.Run(configfilearg.MustParse(os.Args)); err != nil && !errors.Is(err, context.Canceled) {
		logrus.Fatal(err)
	}

	var c chan struct{}
	<-c
}

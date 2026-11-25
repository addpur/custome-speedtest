package main

import (
	"os"
	"time"

	"github.com/addpur/custome-speedtest/bundlefx"
	"github.com/addpur/custome-speedtest/cli"
	"github.com/addpur/custome-speedtest/common"
	"go.uber.org/fx"
)

func main() {
	// If CLI command is invoked: e.g. "speedtest cli --server http://192.168.1.10:3000"
	if len(os.Args) > 1 && (os.Args[1] == "cli" || os.Args[1] == "--cli") {
		if err := cli.Run(os.Args[2:]); err != nil {
			os.Exit(1)
		}
		return
	}

	// Default: Run High Performance Server with Web UI
	fx.New(
		bundlefx.CoreModules,
		bundlefx.EntityModules,
		fx.StopTimeout(common.StopTimeoutBundleFx*time.Second),
	).Run()
}

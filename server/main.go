package main

import (
	_ "github.com/gogf/gf/contrib/drivers/mysql/v2"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gcfg"
	"github.com/gogf/gf/v2/os/gctx"
	"github.com/gogf/gf/v2/os/gfile"

	"itcodex/server/internal/cmd"
)

func init() {
	if adapter, ok := g.Cfg().GetAdapter().(*gcfg.AdapterFile); ok {
		for _, p := range []string{"manifest/config", "server/manifest/config", "../manifest/config"} {
			if gfile.Exists(p) {
				_ = adapter.AddPath(p)
			}
		}
	}
}

func main() {
	cmd.Main.Run(gctx.GetInitCtx())
}

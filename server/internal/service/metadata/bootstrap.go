package metadata

import (
	"context"
	"strings"

	"github.com/gogf/gf/v2/frame/g"
)

func OptionsFromConfig(ctx context.Context) DatabaseOptions {
	dsn := strings.TrimSpace(g.Cfg().MustGet(ctx, "metadata.dsn").String())
	if dsn == "" {
		link := strings.TrimSpace(g.Cfg().MustGet(ctx, "database.default.link").String())
		dsn = strings.TrimPrefix(link, "mysql:")
	}
	return DatabaseOptions{
		DSN:           dsn,
		TablePrefix:   g.Cfg().MustGet(ctx, "metadata.tablePrefix").String(),
		ScriptsPath:   g.Cfg().MustGet(ctx, "metadata.scriptsPath").String(),
		AllowTruncate: g.Cfg().MustGet(ctx, "metadata.allowTruncate").Bool(),
		Logging:       g.Cfg().MustGet(ctx, "metadata.logging").Bool(),
	}
}

func MustBootstrap(ctx context.Context) *Database {
	db, err := NewDatabase(ctx, OptionsFromConfig(ctx))
	if err != nil {
		g.Log().Fatal(ctx, "创建数据库失败:", err)
	}
	if err := db.Bootstrap(ctx); err != nil {
		g.Log().Fatal(ctx, "初始化数据库失败:", err)
	}
	db.SetYaegi(NewYaegiManager(db))
	return db
}

package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"

	controllermd "itcodex/server/internal/controller/metadata"
	md "itcodex/server/internal/logic/metadata"
	"itcodex/server/internal/middleware"
)

func main() {
	ctx := context.Background()

	db, err := md.NewDatabase(ctx, md.DatabaseOptions{
		StoragePath: "./data/metadata.db",
	})
	if err != nil {
		g.Log().Fatal(ctx, "创建数据库失败:", err)
	}

	if err := db.Bootstrap(ctx); err != nil {
		g.Log().Fatal(ctx, "初始化数据库失败:", err)
	}

	ym := md.NewYaegiManager(db)
	db.SetYaegi(ym)

	s := g.Server()
	s.Use(middleware.MetadataContext(db))

	s.Group("/api", func(g *ghttp.RouterGroup) {
		meta := g.Group("/meta")
		mc := controllermd.NewMetaController(db)
		meta.GET("/collections", mc.Collections)
		meta.POST("/collections", mc.CreateCollection)
		meta.GET("/collections/{collectionName}", mc.GetCollection)
		meta.DELETE("/collections/{collectionName}", mc.DropCollection)
		meta.GET("/collections/{collectionName}/fields", mc.Fields)
		meta.POST("/collections/{collectionName}/fields", mc.AddField)
		meta.DELETE("/collections/{collectionName}/fields/{fieldName}", mc.RemoveField)
		meta.GET("/scripts", mc.Scripts)
		meta.POST("/scripts", mc.LoadScript)
		meta.POST("/scripts/{id}/disable", mc.DisableScript)

		cc := controllermd.NewCRUDController(db)
		g.ALL("/c/*action", cc.Handle)

		g.ALL("/custom/*action", middleware.CustomAPIRouter(db))
	})

	s.SetPort(8000)

	go func() {
		s.Run()
	}()

	fmt.Println("ITCodeX Metadata Server 启动在 http://localhost:8000")
	fmt.Println("API 端点:")
	fmt.Println("  GET    /api/meta/collections              - 列出所有集合")
	fmt.Println("  POST   /api/meta/collections              - 创建集合")
	fmt.Println("  GET    /api/meta/collections/:name        - 获取集合信息")
	fmt.Println("  DELETE /api/meta/collections/:name        - 删除集合")
	fmt.Println("  GET    /api/c/:collection                 - 列出记录")
	fmt.Println("  POST   /api/c/:collection                 - 创建记录")
	fmt.Println("  GET    /api/c/:collection/:id             - 获取单条记录")
	fmt.Println("  PUT    /api/c/:collection/:id             - 更新记录")
	fmt.Println("  DELETE /api/c/:collection/:id             - 删除记录")

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	fmt.Println("\n正在关闭服务器...")
	if err := db.Close(ctx); err != nil {
		g.Log().Error(ctx, "关闭数据库失败:", err)
	}
	s.Shutdown()
	fmt.Println("服务器已关闭")
}

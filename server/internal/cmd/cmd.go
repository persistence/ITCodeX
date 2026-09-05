package cmd

import (
	"context"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/gogf/gf/v2/net/goai"
	"github.com/gogf/gf/v2/os/gcmd"

	controllermd "itcodex/server/internal/controller/metadata"
	"itcodex/server/internal/service/bizctx"
	md "itcodex/server/internal/service/metadata"
	"itcodex/server/internal/service/middleware"
)

var Main = gcmd.Command{
	Name:  "main",
	Usage: "main",
	Brief: "start ITCodeX metadata HTTP server",
	Func:  mainFunc,
}

func mainFunc(ctx context.Context, _ *gcmd.Parser) error {
	db := md.MustBootstrap(ctx)
	s := g.Server()
	s.Use(middleware.HandlerResponse, middleware.MetadataContext(db), ghttp.MiddlewareCORS)

	s.Group("/api", func(group *ghttp.RouterGroup) {
		group.Middleware(bizctx.Ctx)
		group.Group("/meta", func(meta *ghttp.RouterGroup) {
			meta.Bind(controllermd.NewV1(db))
		})
		registerDynamicCRUD(group.Group("/c"), db)
		group.Group("/custom", func(custom *ghttp.RouterGroup) {
			custom.ALL("/*action", middleware.CustomAPIRouter(db))
		})
	})

	enhanceOpenAPIDoc(s)
	s.Run()
	return db.Close(ctx)
}

func registerDynamicCRUD(group *ghttp.RouterGroup, db *md.Database) {
	cc := controllermd.NewCRUDController(db)
	group.GET("/{collection}/count", cc.Count)
	group.POST("/{collection}/batch", cc.CreateMany)
	group.GET("/{collection}", cc.List)
	group.POST("/{collection}", cc.Create)
	group.PUT("/{collection}", cc.UpdateMany)
	group.DELETE("/{collection}", cc.DestroyMany)
	group.GET("/{collection}/{id}", cc.Get)
	group.PUT("/{collection}/{id}", cc.Update)
	group.DELETE("/{collection}/{id}", cc.Destroy)
}

func enhanceOpenAPIDoc(s *ghttp.Server) {
	openapi := s.GetOpenApi()
	if openapi == nil {
		return
	}
	openapi.Config.CommonResponse = ghttp.DefaultHandlerResponse{}
	openapi.Config.CommonResponseDataField = `Data`
	openapi.Info = goai.Info{
		Title:       "ITCodeX Metadata API",
		Description: "元数据管理与动态 Collection CRUD",
	}
}

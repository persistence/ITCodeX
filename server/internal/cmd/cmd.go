package cmd

import (
	"context"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/gogf/gf/v2/net/goai"
	"github.com/gogf/gf/v2/os/gcmd"

	controllerauth "itcodex/server/internal/controller/auth"
	controllermd "itcodex/server/internal/controller/metadata"
	controllersecurity "itcodex/server/internal/controller/security"
	md "itcodex/server/internal/service/metadata"
	"itcodex/server/internal/service/middleware"
	securitysvc "itcodex/server/internal/service/security"
)

var Main = gcmd.Command{
	Name:  "main",
	Usage: "main",
	Brief: "start ITCodeX metadata HTTP server",
	Func:  mainFunc,
}

func mainFunc(ctx context.Context, _ *gcmd.Parser) error {
	db := md.MustBootstrap(ctx)
	security, err := securitysvc.Bootstrap(ctx, db)
	if err != nil {
		return err
	}
	s := g.Server()
	s.Use(middleware.HandlerResponse, middleware.MetadataContext(db), ghttp.MiddlewareCORS, middleware.OptionalAuth(security.Auth))

	s.Group("/api", func(group *ghttp.RouterGroup) {
		group.Group("/auth", func(authGroup *ghttp.RouterGroup) {
			authGroup.Bind(controllerauth.NewV1(security.Auth))
		})
		group.Group("/meta", func(meta *ghttp.RouterGroup) {
			meta.Middleware(securitysvc.MetaHTTPMiddleware(security.ACL))
			meta.Bind(controllermd.NewV1(db))
			meta.Group("/security", func(securityGroup *ghttp.RouterGroup) {
				securityGroup.Bind(controllersecurity.NewV1(security.Store, security.ACL))
			})
		})
		registerDynamicCRUD(group.Group("/c"), db, security)
		group.Group("/custom", func(custom *ghttp.RouterGroup) {
			custom.ALL("/*action", middleware.CustomAPIRouter(db, security.Resources))
		})
	})

	enhanceOpenAPIDoc(s)
	s.Run()
	return db.Close(ctx)
}

func registerDynamicCRUD(group *ghttp.RouterGroup, db *md.Database, security *securitysvc.Runtime) {
	cc := controllermd.NewCRUDController(db, security.Resources)
	group.GET("/{collection}/count", cc.Count)
	group.POST("/{collection}/batch", cc.CreateMany)
	group.POST("/{collection}/upload", cc.Upload)
	group.GET("/{collection}/files/{id}/content", cc.FileContent)
	group.GET("/{collection}", cc.List)
	group.POST("/{collection}", cc.Create)
	group.PUT("/{collection}", cc.UpdateMany)
	group.DELETE("/{collection}", cc.DestroyMany)
	group.GET("/{collection}/{id}/{association}", cc.AssociationList)
	group.POST("/{collection}/{id}/{association}", cc.AssociationAdd)
	group.PUT("/{collection}/{id}/{association}", cc.AssociationSet)
	group.DELETE("/{collection}/{id}/{association}", cc.AssociationRemove)
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

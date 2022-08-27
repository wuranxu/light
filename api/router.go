package api

import (
	"github.com/gin-gonic/gin"
	"github.com/wuranxu/light/middleware"
)

type PityGatewayRouter struct {
	app *gin.Engine
}

func NewRouter(app *gin.Engine) *PityGatewayRouter {
	return &PityGatewayRouter{app: app}
}

func (p *PityGatewayRouter) AddRoute() {
	p.app.GET("/", func(context *gin.Context) {
		context.String(200, "hello, pity gateway!")
	})
	p.app.GET("/vi/health", func(context *gin.Context) {
		context.String(200, "working!")
	})

	p.app.Use(middleware.Auth).POST("/")

}

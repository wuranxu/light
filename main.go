package main

import (
	"flag"
	"fmt"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/wuranxu/light/api"
	"github.com/wuranxu/light/conf"
	"github.com/wuranxu/light/internal/service/etcd"
	"log"
)

var (
	serverHost = flag.String("host", "0.0.0.0", "gateway host")
	serverPort = flag.Int("port", 8080, "gateway port")
	configPath = flag.String("config", "J:\\projects\\github.com\\wuranxu\\light\\resources\\application.yml", "gateway config filepath")
)

func main() {
	flag.Parse()
	if err := conf.Init(*configPath); err != nil {
		log.Fatal("init config error: ", err)
	}
	if err := etcd.Init(conf.Conf.Etcd); err != nil {
		log.Fatal("init etcd error: ", err)
	}
	app := gin.New()
	app.Use(cors.Default())
	app.Use(gin.Logger())
	app.Use(gin.Recovery())

	router := api.NewRouter(app)
	router.AddRoute()

	// Method:   GET
	// Resource: http://localhost:8080
	//app.Handle("POST", "/:service/:method", auth.Auth(handler.CallRpc))
	//app.Handle("POST", "/api/:service/:method", auth.Auth(handler.CallRpcWithAuth))
	//app.Handle("POST", "/:version/:service/:method", handler.CallRpc)

	app.Run(fmt.Sprintf("%s:%d", *serverHost, *serverPort))
}

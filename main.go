package main

import (
	"flag"
	"net/http"
	"os"
	"time"

	"github.com/vicanso/cod"
	"github.com/vicanso/cod/middleware"
	_ "github.com/vicanso/diving/controller"
	"github.com/vicanso/diving/log"
	"github.com/vicanso/diving/router"
	"go.uber.org/zap"
)

var (
	runMode string
)

// 获取监听地址
func getListen() string {
	listen := os.Getenv("LISTEN")
	if listen == "" {
		listen = ":7001"
	}
	return listen
}

func check() {
	listen := getListen()
	url := ""
	if listen[0] == ':' {
		url = "http://127.0.0.1" + listen + "/ping"
	} else {
		url = "http://" + listen + "/ping"
	}
	client := http.Client{
		Timeout: 3 * time.Second,
	}
	resp, err := client.Get(url)
	if err != nil || resp == nil || resp.StatusCode != http.StatusOK {
		os.Exit(1)
		return
	}
	os.Exit(0)
}

func main() {

	flag.StringVar(&runMode, "mode", "", "running mode")
	flag.Parse()

	if runMode == "check" {
		check()
		return
	}
	listen := getListen()

	logger := log.Default()

	d := cod.New()

	d.Use(middleware.NewRecover())

	d.Use(middleware.NewStats(middleware.StatsConfig{
		OnStats: func(statsInfo *middleware.StatsInfo, _ *cod.Context) {
			logger.Info("access log",
				zap.String("ip", statsInfo.IP),
				zap.String("method", statsInfo.Method),
				zap.String("uri", statsInfo.URI),
				zap.Int("status", statsInfo.Status),
				zap.String("consuming", statsInfo.Consuming.String()),
			)
		},
	}))

	d.Use(middleware.NewDefaultErrorHandler())

	d.Use(func(c *cod.Context) error {
		c.NoCache()
		return c.Next()
	})

	// 因为有使用pike做缓存（已包括ETag fresh compress的处理），无需要添加此类中间件
	// d.Use(middleware.NewDefaultFresh())
	// d.Use(middleware.NewDefaultETag())
	// d.Use(middleware.NewDefaultCompress())

	d.Use(middleware.NewDefaultResponder())

	// health check
	d.GET("/ping", func(c *cod.Context) (err error) {
		c.Body = "pong"
		return
	})

	groups := router.GetGroups()
	for _, g := range groups {
		d.AddGroup(g)
	}

	logger.Info("server will listen on " + listen)
	err := d.ListenAndServe(listen)
	if err != nil {
		panic(err)
	}
}
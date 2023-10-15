package server

import (
	"fmt"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/snsinfu/reverse-tunnel/config"
)

// Start starts tunneling server with given configuration.
func Start(conf config.Server) error {
	if err := conf.Check(); err != nil {
		return fmt.Errorf("config error: %w", err)
	}

	e := echo.New()
	e.HideBanner = true

	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	action := NewAction(conf)
	e.GET("/tcp/:port", action.GetTCPPort)
	e.GET("/udp/:port", action.GetUDPPort)
	e.GET("/session/:id", action.GetSession)

	if conf.TLSConf.KeyPath != "" {
		return e.StartTLS(conf.ControlAddress, conf.TLSConf.CertPath, conf.TLSConf.KeyPath)
	}

	// Expose metrics via another port.
	go func() {
		metrics := echo.New()
		metrics.GET("/metrics", echo.WrapHandler(promhttp.Handler()))
		_ = metrics.Start(conf.MetricsAddress)
		// Or, using plain http instead:
		//http.Handle("/metrics", promhttp.Handler())
		//_ = http.ListenAndServe(conf.MetricsPort, nil)
	}()

	return e.Start(conf.ControlAddress)
}

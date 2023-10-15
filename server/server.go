package server

import (
	"fmt"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/snsinfu/reverse-tunnel/config"
	"net/http"
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
		//metrics := echo.New()
		//metrics.GET("/metrics", echo.WrapHandler(promhttp.Handler()))
		//err := metrics.Start(conf.MetricsAddress)
		//if err != nil {
		//	fmt.Printf("Got an error while starting the metrics server at %s, err = %v", conf.MetricsAddress, err)
		//}
		// Or, using plain http instead:
		fmt.Printf("Starting the metrics server at %s", conf.MetricsAddress)
		http.Handle("/metrics", promhttp.Handler())
		err := http.ListenAndServe(conf.MetricsAddress, nil)
		if err != nil {
			fmt.Printf("Got an error while starting the metrics server at %s, err = %v", conf.MetricsAddress, err)
		}
	}()

	return e.Start(conf.ControlAddress)
}

package service

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// UplinkBytesCounterVec counts the uplink bytes
var UplinkBytesCounterVec = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "uplink_bytes",
	Help: "",
}, []string{
	"port",
})

// DownlinkBytesCounterVec counts the downlink bytes
var DownlinkBytesCounterVec = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "downlink_bytes",
	Help: "",
}, []string{
	"port",
})

package main

import (
	"net/http"

	log "github.com/Sirupsen/logrus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/stefanprodan/xmicro/xproxy"
)

// StartProxy starts the HTTP Reverse Proxy server backed by Consul
func StartProxy(address string, proxy *xproxy.ReverseProxy) {

	xproxy.RegisterMetrics()
	err := proxy.StartConsulSync()
	if err != nil {
		log.Fatal(err.Error())
	}

	http.HandleFunc("/", proxy.ReverseHandlerFunc())
	http.HandleFunc("/registry", func(w http.ResponseWriter, req *http.Request) {
		appCtx.Render.JSON(w, http.StatusOK, proxy.ServiceRegistry)
	})
	http.HandleFunc("/ping", func(w http.ResponseWriter, req *http.Request) {
		appCtx.Render.Text(w, http.StatusOK, "pong")
	})
	http.HandleFunc("/health", func(w http.ResponseWriter, req *http.Request) {
		appCtx.Render.JSON(w, http.StatusOK, appCtx)
	})
	http.HandleFunc("/error", func(w http.ResponseWriter, req *http.Request) {
		appCtx.Render.Text(w, http.StatusNotAcceptable, "Not Acceptable")
	})

	http.Handle("/metrics", promhttp.Handler())

	log.Printf("Proxy started on %s", address)
	log.Fatal(http.ListenAndServe(address, nil))
}

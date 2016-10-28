package main

import (
	"net/http"

	log "github.com/Sirupsen/logrus"
	"github.com/stefanprodan/xmicro/xproxy"
)

//StartProxy starts the HTTP Reverse Proxy server backed by Consul
func StartProxy(address string, proxy *xproxy.ReverseProxy) {

	err := proxy.StartConsulSync()
	if err != nil {
		log.Fatal(err.Error())
	}

	http.HandleFunc("/", proxy.HandlerFunc())
	http.HandleFunc("/registry", func(w http.ResponseWriter, req *http.Request) {
		appCtx.Render.JSON(w, http.StatusOK, proxy.ServiceRegistry)
	})
	http.HandleFunc("/ping", func(w http.ResponseWriter, req *http.Request) {
		appCtx.Render.Text(w, http.StatusOK, "pong")
	})
	http.HandleFunc("/health", func(w http.ResponseWriter, req *http.Request) {
		appCtx.Render.JSON(w, http.StatusOK, appCtx)
	})

	log.Printf("Proxy started on %s", address)
	log.Fatal(http.ListenAndServe(address, nil))
}

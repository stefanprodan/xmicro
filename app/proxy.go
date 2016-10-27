package main

import (
	"log"
	"net/http"

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
		render.JSON(w, http.StatusOK, proxy.ServiceRegistry)
	})
	http.HandleFunc("/ping", func(w http.ResponseWriter, req *http.Request) {
		render.Text(w, http.StatusOK, "pong")
	})

	log.Printf("Proxy started on %s", address)
	log.Fatal(http.ListenAndServe(address, nil))
}

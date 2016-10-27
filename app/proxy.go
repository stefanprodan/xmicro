package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/stefanprodan/xmicro/xproxy"
)

//StartProxy starts the HTTP Reverse Proxy server backed by Consul
func StartProxy(address string, keyPrefix string) {

	proxy := &xproxy.ReverseProxy{
		ServiceRegistry:     xproxy.Registry{},
		ElectionKeyPrefix:   keyPrefix,
		Scheme:              "http",
		MaxIdleConnsPerHost: 500,
		DisableKeepAlives:   true,
	}

	log.Fatal(proxy.StartConsulSync())

	http.HandleFunc("/", proxy.HandlerFunc())
	http.HandleFunc("/health", func(w http.ResponseWriter, req *http.Request) {
		fmt.Fprintf(w, "%v\n", proxy.ServiceRegistry)
	})
	http.HandleFunc("/ping", func(w http.ResponseWriter, req *http.Request) {
		w.Write([]byte("pong"))
	})

	log.Printf("Proxy started on %s", address)
	log.Fatal(http.ListenAndServe(address, nil))
}

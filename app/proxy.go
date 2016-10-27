package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/stefanprodan/xmicro/proxy"
)

var ServiceRegistry = proxy.Registry{
	"xmicro-node1": {
		"192.168.1.134:8001",
		//"192.168.1.134:8003",
	},
	"xmicro-node2": {
		"192.168.1.134:8002",
		//"192.168.1.134:8004",
	},
}

//StartProxy starts the HTTP Reverse Proxy server
func StartProxy(address string) {
	http.HandleFunc("/", proxy.NewReverseProxy(ServiceRegistry, "http"))
	http.HandleFunc("/health", func(w http.ResponseWriter, req *http.Request) {
		fmt.Fprintf(w, "%v\n", ServiceRegistry)
	})
	log.Printf("Proxy started on %s", address)
	log.Fatal(http.ListenAndServe(address, nil))
}

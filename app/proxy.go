package main

import (
	"fmt"
	"log"
	"net/http"

	consul "github.com/hashicorp/consul/api"
	watch "github.com/hashicorp/consul/watch"
	"github.com/stefanprodan/xmicro/xproxy"
)

var serviceRegistry = xproxy.Registry{}
var electionKeyPrefix = ""

//StartProxy starts the HTTP Reverse Proxy server
func StartProxy(address string, keyPrefix string) {
	electionKeyPrefix = keyPrefix
	serviceRegistry.GetServices(electionKeyPrefix)
	startConsulWatchers(electionKeyPrefix)
	http.HandleFunc("/", xproxy.NewReverseProxy(serviceRegistry, "http"))
	http.HandleFunc("/health", func(w http.ResponseWriter, req *http.Request) {
		fmt.Fprintf(w, "%v\n", serviceRegistry)
	})
	http.HandleFunc("/ping", func(w http.ResponseWriter, req *http.Request) {
		w.Write([]byte("pong"))
	})

	log.Printf("Proxy started on %s", address)
	log.Fatal(http.ListenAndServe(address, nil))
}

func startConsulWatchers(keyPrefix string) error {
	serviceWatch, err := watch.Parse(map[string]interface{}{"type": "services"})
	if err != nil {
		return err
	}
	serviceWatch.Handler = handleChanges
	config := consul.DefaultConfig()
	go serviceWatch.Run(config.Address)

	leaderWatch, err := watch.Parse(map[string]interface{}{"type": "keyprefix", "prefix": keyPrefix})
	if err != nil {
		return err
	}
	leaderWatch.Handler = handleChanges
	go leaderWatch.Run(config.Address)
	return nil
}

func handleChanges(idx uint64, data interface{}) {
	log.Print("Leader change detected")
	serviceRegistry.GetServices(electionKeyPrefix)
}

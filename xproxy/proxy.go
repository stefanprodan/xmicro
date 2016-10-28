package xproxy

import (
	"fmt"
	"math/rand"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	log "github.com/Sirupsen/logrus"
	consul "github.com/hashicorp/consul/api"
	watch "github.com/hashicorp/consul/watch"
)

// ReverseProxy holds the proxy configuration, registry and Consul watchers
type ReverseProxy struct {
	ServiceRegistry     Registry
	ElectionKeyPrefix   string
	Scheme              string
	MaxIdleConnsPerHost int
	DisableKeepAlives   bool
	serviceWatch        *watch.WatchPlan
	leaderWatch         *watch.WatchPlan
}

// StartConsulSync watches for changes in Consul Registry and syncs with the in memory registry
func (r *ReverseProxy) StartConsulSync() error {
	r.ServiceRegistry.GetServices(r.ElectionKeyPrefix)
	err := r.startConsulWatchers()
	if err != nil {
		return err
	}
	return nil
}

// HandlerFunc creates a http handler that will resolve services from Consul.
// If a service has the cl tag, the proxy will point to the leader.
// If multiple addresses are found for a service then it will load balance between those instances.
func (r *ReverseProxy) HandlerFunc() http.HandlerFunc {
	transport := &http.Transport{
		DisableKeepAlives:   true,
		MaxIdleConnsPerHost: 500,
	}
	return func(w http.ResponseWriter, req *http.Request) {
		name, err := parseServiceName(req.URL)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		//resolve service name address
		endpoints, _ := r.ServiceRegistry.Lookup(name)

		if len(endpoints) == 0 {
			log.Warnf("xproxy: service not found in registry %s", name)
			return
		}

		//random load balancer
		//TODO: implement round robin
		endpoint := endpoints[rand.Int()%len(endpoints)]

		reverseProxy := &httputil.ReverseProxy{
			Director: func(req *http.Request) {
				req.URL.Scheme = r.Scheme
				req.URL.Host = endpoint
			},
			Transport: transport,
		}

		reverseProxy.ServeHTTP(w, req)
	}
}

// extracts the service name from the URL, http://<proxy>/<service_name>/path/to
func parseServiceName(target *url.URL) (name string, err error) {
	path := target.Path
	if len(path) > 1 && path[0] == '/' {
		path = path[1:]
	}
	tmp := strings.Split(path, "/")
	if len(tmp) < 1 {
		return "", fmt.Errorf("xproxy: parse service name failed, invalid path %s", path)
	}
	name = tmp[0]
	target.Path = "/" + strings.Join(tmp[1:], "/")
	return name, nil
}

//watch for services status changes (up/down or leadership changes)
func (r *ReverseProxy) startConsulWatchers() error {
	serviceWatch, err := watch.Parse(map[string]interface{}{"type": "services"})
	if err != nil {
		return err
	}
	r.serviceWatch = serviceWatch
	serviceWatch.Handler = r.handleServiceChanges
	config := consul.DefaultConfig()
	go serviceWatch.Run(config.Address)

	leaderWatch, err := watch.Parse(map[string]interface{}{"type": "keyprefix", "prefix": r.ElectionKeyPrefix})
	if err != nil {
		return err
	}
	r.leaderWatch = leaderWatch
	leaderWatch.Handler = r.handleLeaderChanges
	go leaderWatch.Run(config.Address)
	return nil
}

//reload services from Consul
func (r *ReverseProxy) handleServiceChanges(idx uint64, data interface{}) {
	log.Info("Service change detected")
	r.ServiceRegistry.GetServices(r.ElectionKeyPrefix)
}

//reload leaders from Consul
func (r *ReverseProxy) handleLeaderChanges(idx uint64, data interface{}) {
	log.Info("Leader change detected")
	r.ServiceRegistry.GetServices(r.ElectionKeyPrefix)
}

// Stop stops the Consul watchers
func (r *ReverseProxy) Stop() {
	r.serviceWatch.Stop()
	r.leaderWatch.Stop()
}

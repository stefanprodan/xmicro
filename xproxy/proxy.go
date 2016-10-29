package xproxy

import (
	"fmt"
	"math/rand"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"strings"
	"time"

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
	r.ServiceRegistry.Catalog = make(map[string][]string)
	r.ServiceRegistry.GetServices(r.ElectionKeyPrefix)
	err := r.startConsulWatchers()
	if err != nil {
		return err
	}

	http.DefaultTransport.(*http.Transport).MaxIdleConnsPerHost = r.MaxIdleConnsPerHost
	http.DefaultTransport.(*http.Transport).DisableKeepAlives = r.DisableKeepAlives

	return nil
}

// HandlerFunc creates a http handler that will resolve services from Consul.
// If a service has the cl tag, the proxy will point to the leader.
// If multiple addresses are found for a service then it will load balance between those instances.
func (r *ReverseProxy) LoadBalanceHandlerFunc() http.HandlerFunc {
	transport := &http.Transport{
		DisableKeepAlives:   r.DisableKeepAlives,
		MaxIdleConnsPerHost: r.MaxIdleConnsPerHost,
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

// watch for services status changes (up/down or leadership changes)
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

// reload services from Consul
func (r *ReverseProxy) handleServiceChanges(idx uint64, data interface{}) {
	log.Info("Service change detected")
	r.ServiceRegistry.GetServices(r.ElectionKeyPrefix)
}

// reload leaders from Consul
func (r *ReverseProxy) handleLeaderChanges(idx uint64, data interface{}) {
	log.Info("Leader change detected")
	r.ServiceRegistry.GetServices(r.ElectionKeyPrefix)
}

// Stop stops the Consul watchers
func (r *ReverseProxy) Stop() {
	r.serviceWatch.Stop()
	r.leaderWatch.Stop()
}

// ReverseHandlerFunc creates a http handler that will resolve services from Consul.
// If a service has the cl tag, the proxy will point to the leader.
func (r *ReverseProxy) ReverseHandlerFunc() http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		service, err := parseServiceName(req.URL)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		//resolve service name address
		endpoints, _ := r.ServiceRegistry.Lookup(service)

		if len(endpoints) == 0 {
			log.Warnf("xproxy: service not found in registry %s", service)
			return
		}

		endpoint := endpoints[0]
		redirect, _ := url.ParseRequestURI(r.Scheme + "://" + endpoint)

		rproxy := httputil.NewSingleHostReverseProxy(redirect)
		rproxy.FlushInterval = 100 * time.Microsecond
		rproxy.Transport = &proxyTransport{
			service: service,
		}
		rproxy.ServeHTTP(w, req)
	})
}

// RoundTrip records prometheus metrics. On debug logs the request URL, status code and duration.
func (t *proxyTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	start := time.Now().UTC()
	response, err := http.DefaultTransport.RoundTrip(req)

	if err == nil {
		log.Debugf("Round trip to %v at %v, code: %v, duration: %v", t.service, req.URL, response.StatusCode, time.Now().UTC().Sub(start))
		xproxy_roundtrips_total.WithLabelValues(t.service, strconv.Itoa(response.StatusCode)).Inc()
	} else {
		// set status code 5000 for transport errors
		xproxy_roundtrips_total.WithLabelValues(t.service, strconv.Itoa(5000)).Inc()
		log.Warnf("Round trip error %s", err.Error())
	}

	xproxy_roundtrips_latency.WithLabelValues(t.service).Observe(time.Since(start).Seconds())
	return response, err
}

type proxyTransport struct {
	service string
}

package xproxy

import (
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
)

// NewReverseProxy creates a reverse proxy handler that will resolve
// services from Consul. If a service has the cl tag, the proxy will point to the leader.
// If multiple addreses are found for a service then it will load balance between those instaces.
func NewReverseProxy(reg Registry, scheme string) http.HandlerFunc {
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
		endpoints, _ := reg.Lookup(name)

		if len(endpoints) == 0 {
			log.Printf("xproxy: service not found in registry " + name)
			return
		}

		//random load balancer
		//TODO: implement round robin (a mutex is required and could slow down the proxy)
		endpoint := endpoints[rand.Int()%len(endpoints)]

		reverseProxy := &httputil.ReverseProxy{
			Director: func(req *http.Request) {
				req.URL.Scheme = scheme
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
		return "", fmt.Errorf("xproxy: parse service name faild, invalid path %s", path)
	}
	name = tmp[0]
	target.Path = "/" + strings.Join(tmp[1:], "/")
	return name, nil
}

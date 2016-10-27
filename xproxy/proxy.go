package xproxy

import (
	"errors"
	"fmt"
	"math/rand"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"
)

var dialer = (&net.Dialer{
	Timeout:   2 * time.Second,
	KeepAlive: 10 * time.Second,
}).Dial

func loadBalance(network, service string, reg Registry) (net.Conn, error) {
	endpoints, err := reg.Lookup(service)
	if err != nil {
		return nil, err
	}
	for {
		//stop: no more endpoints
		if len(endpoints) == 0 {
			break
		}
		//select a random endpoint
		i := rand.Int() % len(endpoints)
		endpoint := endpoints[i]

		//try to connect
		conn, err := dialer(network, endpoint)
		if err != nil {
			//failure: remove the endpoint from the current list and try again
			endpoints = append(endpoints[:i], endpoints[i+1:]...)
			continue
		}
		//success: return the connection
		return conn, nil
	}
	return nil, fmt.Errorf("no endpoint found in registry for %s", service)
}

func extractNameVersion(target *url.URL) (name string, err error) {
	path := target.Path
	if len(path) > 1 && path[0] == '/' {
		path = path[1:]
	}
	tmp := strings.Split(path, "/")
	if len(tmp) < 1 {
		return "", fmt.Errorf("invalid path %s", path)
	}
	name = tmp[0]
	target.Path = "/" + strings.Join(tmp[1:], "/")
	return name, nil
}

func NewReverseProxy(reg Registry, scheme string) http.HandlerFunc {
	//set http as default scheme
	if scheme == "" {
		scheme = "http"
	}
	transport := &http.Transport{
		MaxIdleConnsPerHost:   50,
		ResponseHeaderTimeout: 10 * time.Second,
		Proxy: http.ProxyFromEnvironment,
		Dial: func(network, addr string) (net.Conn, error) {
			addr = strings.Split(addr, ":")[0]
			tmp := strings.Split(addr, "/")
			if len(tmp) != 1 {
				return nil, errors.New("invalid service for " + addr)
			}
			return loadBalance(network, tmp[0], reg)
		},
		TLSHandshakeTimeout: 10 * time.Second,
	}
	return func(w http.ResponseWriter, req *http.Request) {
		name, err := extractNameVersion(req.URL)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		reverseProxy := &httputil.ReverseProxy{
			Director: func(req *http.Request) {
				req.URL.Scheme = scheme
				req.URL.Host = name
			},
			Transport:     transport,
			FlushInterval: 2 * time.Second,
		}

		reverseProxy.ServeHTTP(w, req)
	}
}

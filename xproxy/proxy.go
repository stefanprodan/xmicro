package xproxy

import (
	"errors"
	"fmt"
	"log"
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
	KeepAlive: 0,
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
			log.Print("dialer error " + err.Error())
			//failure: remove the endpoint from the current list and try again
			endpoints = append(endpoints[:i], endpoints[i+1:]...)
			continue
		}
		//success: return the connection
		return conn, nil
	}
	return nil, fmt.Errorf("no endpoint found in registry for %s", service)
}

func parseServiceName(target *url.URL) (name string, err error) {
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

func NewReverseProxy2(reg Registry, scheme string) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		name, err := parseServiceName(req.URL)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		reverseProxy := &httputil.ReverseProxy{
			Director: func(req *http.Request) {
				req.URL.Scheme = scheme
				req.URL.Host = name
			},
			Transport: &http.Transport{
				DisableKeepAlives:     true,
				MaxIdleConnsPerHost:   10,
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
			},
		}

		reverseProxy.ServeHTTP(w, req)
	}
}

func NewReverseProxy1(reg Registry, scheme string) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		name, err := parseServiceName(req.URL)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		reverseProxy := &httputil.ReverseProxy{
			Director: func(req *http.Request) {
				req.URL.Scheme = scheme
				req.URL.Host = name
			},
			Transport: &http.Transport{
				MaxIdleConnsPerHost:   1,
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
			},
			FlushInterval: 2 * time.Second,
		}

		reverseProxy.ServeHTTP(w, req)
	}
}

func NewMultipleHostReverseProxy(reg Registry, scheme string) *httputil.ReverseProxy {
	director := func(req *http.Request) {
		name, err := parseServiceName(req.URL)
		if err != nil {
			log.Print(err)
			return
		}
		endpoints := reg[name]
		if len(endpoints) == 0 {
			log.Printf("Service not found ")
			return
		}
		req.URL.Scheme = scheme
		req.URL.Host = endpoints[rand.Int()%len(endpoints)]
	}
	return &httputil.ReverseProxy{
		Director: director,
	}
}

func NewReverseProxy(reg Registry, scheme string) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		name, err := parseServiceName(req.URL)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		endpoints := reg[name]
		if len(endpoints) == 0 {
			log.Printf("Service not found " + name)
			return
		}

		reverseProxy := &httputil.ReverseProxy{
			Director: func(req *http.Request) {
				req.URL.Scheme = scheme
				req.URL.Host = endpoints[0]
			},
		}

		reverseProxy.ServeHTTP(w, req)
	}
}

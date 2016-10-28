package main

import (
	"crypto/rand"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	log "github.com/Sirupsen/logrus"
	"github.com/stefanprodan/xmicro/xconsul"
	"github.com/stefanprodan/xmicro/xproxy"
)

type appFlags struct {
	port                     int
	env                      string
	role                     string
	logLevel                 string
	electionKeyPrefix        string
	proxyScheme              string
	proxyMaxIdleConnsPerHost int
	proxyDisableKeepAlives   bool
}

type stoppableService interface {
	Stop()
}

func main() {
	var flags = appFlags{}
	flag.IntVar(&flags.port, "port", 8000, "HTTP port to listen on")
	flag.StringVar(&flags.env, "env", "DEBUG", "environment: DEBUG, DEV, STG, PROD")
	flag.StringVar(&flags.role, "role", "proxy", "roles: proxy, frontend, backend, storage")
	flag.StringVar(&flags.logLevel, "loglevel", "info", "logging threshold level: debug|info|warn|error|fatal|panic")
	flag.StringVar(&flags.electionKeyPrefix, "electionKeyPrefix", "xmicro/election/", "format: namespace/election/")
	flag.StringVar(&flags.proxyScheme, "proxyScheme", "http", "proxy scheme: http or https")
	flag.IntVar(&flags.proxyMaxIdleConnsPerHost, "proxyMaxIdleConnsPerHost", 500, "proxy max idle connections per host")
	flag.BoolVar(&flags.proxyDisableKeepAlives, "proxyDisableKeepAlives", true, "proxy disable KeepAlive")
	flag.Parse()

	setLogLevel(flags.logLevel)

	var (
		election = &xconsul.Election{}
		proxy    = &xproxy.ReverseProxy{
			ServiceRegistry:     xproxy.Registry{},
			ElectionKeyPrefix:   flags.electionKeyPrefix,
			Scheme:              flags.proxyScheme,
			MaxIdleConnsPerHost: flags.proxyMaxIdleConnsPerHost,
			DisableKeepAlives:   flags.proxyDisableKeepAlives,
		}
	)

	err := initCtx(flags.env, flags.port, flags.role)
	if err != nil {
		log.Fatal(err.Error())
	}

	log.Info("Starting xmicro " + appCtx.Hostname + " role " + appCtx.Role + " on port " + fmt.Sprintf("%v", appCtx.Port) + " in " + appCtx.Env + " mode. Work dir " + appCtx.WorkDir)

	if appCtx.Role == "proxy" {
		go StartProxy(fmt.Sprintf(":%v", appCtx.Port), proxy)

	} else {
		election = xconsul.BeginElection(appCtx.Hostname, flags.electionKeyPrefix, appCtx.Role)
		go StartAPI(fmt.Sprintf(":%v", appCtx.Port), election)
	}

	// wait for OS signal
	osChan := make(chan os.Signal)
	signal.Notify(osChan, syscall.SIGINT, syscall.SIGTERM)
	osSignal := <-osChan
	log.Info("Stoping services. OS signal: %v", osSignal)
	// stop services
	if appCtx.Role == "proxy" {
		stop(proxy)
	} else {
		stop(election)
	}
}

func stop(services ...stoppableService) {
	log.Println("Stopping background services...")
	for _, service := range services {
		service.Stop()
	}
}

func setLogLevel(levelname string) {
	level, err := log.ParseLevel(levelname)
	if err != nil {
		log.Fatal(err)
	}
	log.SetLevel(level)
}

func genServiceName() string {
	host, _ := os.Hostname()
	b := make([]byte, 16)
	rand.Read(b)
	return fmt.Sprintf("%s-%x", strings.ToLower(host), b[0:4])
}

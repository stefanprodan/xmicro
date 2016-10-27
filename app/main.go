package main

import (
	"crypto/rand"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"

	"github.com/stefanprodan/xmicro/xconsul"
)

type stoppableService interface {
	Stop()
}

func main() {

	port := flag.Int("port", 8000, "HTTP port")
	env := flag.String("env", "DEBUG", "ENV: DEBUG, DEV, STG, PROD")
	role := flag.String("role", "proxy", "Roles: proxy, frontend, backend, storage")
	flag.Parse()

	var (
		host, _     = os.Hostname()
		workDir, _  = os.Getwd()
		election    = &xconsul.Election{}
		electionKey = "xmicro/election/"
	)

	if *role != "proxy" {
		election = xconsul.BeginElection(host, electionKey+*role)
		go StartAPI(fmt.Sprintf(":%v", *port), election)
	} else {
		client, _ := xconsul.NewClient()
		xconsul.ListServices(client)
		go StartProxy(fmt.Sprintf(":%v", *port))
	}

	log.Println("Starting xmicro " + host + " role " + *role + " on port " + fmt.Sprintf("%v", *port) + " in " + *env + " mode. Work dir " + workDir)

	// block
	osChan := make(chan os.Signal, 1)
	// trigger with docker kill --signal=SIGINT
	signal.Notify(osChan, os.Interrupt, os.Kill)
	osSignal := <-osChan

	if *role != "proxy" {
		stop(election)
	}

	log.Printf("Exiting! OS signal: %v", osSignal)
}

func stop(services ...stoppableService) {
	log.Println("Stopping background services...")
	for _, service := range services {
		service.Stop()
	}
}

func genServiceName() string {
	host, _ := os.Hostname()
	b := make([]byte, 16)
	rand.Read(b)
	return fmt.Sprintf("%s-%x", strings.ToLower(host), b[0:4])
}

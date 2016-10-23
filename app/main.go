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
	flag.Parse()

	var (
		host       = genServiceName()
		workDir, _ = os.Getwd()
	)

	election := xconsul.BeginElection(host)

	log.Println("Starting xmicro " + host + " in " + workDir + " mode.")

	go StartApi(fmt.Sprintf(":%v", *port), election)

	// block
	osChan := make(chan os.Signal, 1)
	signal.Notify(osChan, os.Interrupt, os.Kill)
	osSignal := <-osChan
	stop(election)

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

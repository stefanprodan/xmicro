package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/stefanprodan/xmicro/xconsul"
)

const electionContextKey = "election"

//StartAPI starts the HTTP API server
func StartAPI(address string, election *xconsul.Election) {

	electionStatusHandler := ElectionMiddleware(election, http.HandlerFunc(statusResponse))
	pingHandler := ElectionMiddleware(election, http.HandlerFunc(pingResponse))

	mux := new(http.ServeMux)
	mux.Handle("/", electionStatusHandler)
	mux.Handle("/ping", pingHandler)
	log.Printf("API started on %s", address)
	err := http.ListenAndServe(address, mux)
	if err != nil {
		log.Panic(err.Error())
	}
}

//ElectionMiddleware injects the election pointer
func ElectionMiddleware(election *xconsul.Election, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Server", "xmicro")
		ctx := context.WithValue(r.Context(), electionContextKey, election)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func pingResponse(w http.ResponseWriter, r *http.Request) {
	render.Text(w, http.StatusOK, "pong")
}

func statusResponse(w http.ResponseWriter, r *http.Request) {
	election := r.Context().Value(electionContextKey).(*xconsul.Election)
	status := ""
	if election.Leader() == "" {
		status = "Leader election in process"
	} else {
		status = fmt.Sprintf("Acting as leader %v", election.IsLeader())
	}
	hostname, _ := os.Hostname()
	render.JSON(w, http.StatusOK, map[string]string{"status": status, "hostname": hostname, "leader": election.Leader()})
}

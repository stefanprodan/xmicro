package main

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/stefanprodan/xmicro/xconsul"
)

var electionContextKey = "election"

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
		ctx := context.WithValue(r.Context(), electionContextKey, election)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func pingResponse(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("pong"))
}

func statusResponse(w http.ResponseWriter, r *http.Request) {
	election := r.Context().Value(electionContextKey).(*xconsul.Election)
	status := ""
	if election.Leader() == "" {
		status = "Leader election in process"
	} else {
		status = fmt.Sprintf("Leader name is %s. Leader %v", election.Leader(), election.IsLeader())
	}
	w.Write([]byte(status))
}

package main

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/stefanprodan/xmicro/xconsul"
)

func statusResponse(w http.ResponseWriter, r *http.Request) {
	election := r.Context().Value(ElectionContextKey).(*xconsul.Election)
	status := fmt.Sprintf("Leader name is %s. Leader %v", election.Leader(), election.IsLeader())
	w.Write([]byte(status))
}

func StartApi(address string, election *xconsul.Election) {

	electionStatusHandler := ElectionMiddleware(election, http.HandlerFunc(statusResponse))

	mux := new(http.ServeMux)
	mux.Handle("/", electionStatusHandler)
	log.Printf("API started on %s", address)
	err := http.ListenAndServe(address, mux)
	if err != nil {
		log.Panic(err.Error())
	}
}

const ElectionContextKey = "election"

func ElectionMiddleware(election *xconsul.Election, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := context.WithValue(r.Context(), ElectionContextKey, election)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

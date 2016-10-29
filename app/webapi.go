package main

import (
	"context"
	"fmt"
	"net/http"

	log "github.com/Sirupsen/logrus"
	"github.com/stefanprodan/xmicro/xconsul"
)

const electionContextKey = "election"

// StartAPI starts the HTTP API server
func StartAPI(address string, election *xconsul.Election) {

	electionStatusHandler := HeadersMiddleware(ElectionMiddleware(election, http.HandlerFunc(statusResponse)))
	pingHandler := HeadersMiddleware(http.HandlerFunc(pingResponse))
	healthHandler := HeadersMiddleware(http.HandlerFunc(healthResponse))
	errorHandler := HeadersMiddleware(http.HandlerFunc(errorResponse))

	mux := new(http.ServeMux)
	mux.Handle("/", electionStatusHandler)
	mux.Handle("/ping", pingHandler)
	mux.Handle("/health", healthHandler)
	mux.Handle("/error", errorHandler)
	log.Printf("API started on %s", address)
	log.Fatal(http.ListenAndServe(address, mux))
}

// ElectionMiddleware injects the election pointer
func ElectionMiddleware(election *xconsul.Election, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := context.WithValue(r.Context(), electionContextKey, election)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// HeadersMiddleware injects server headers
func HeadersMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Server", "xmicro")
		w.Header().Set("X-Version", appCtx.Version)
		next.ServeHTTP(w, r)
	})
}

func pingResponse(w http.ResponseWriter, r *http.Request) {
	appCtx.Render.Text(w, http.StatusOK, "pong")
}

func healthResponse(w http.ResponseWriter, r *http.Request) {
	appCtx.Render.JSON(w, http.StatusOK, appCtx)
}

func errorResponse(w http.ResponseWriter, r *http.Request) {
	appCtx.Render.Text(w, http.StatusNotAcceptable, "Not Acceptable")
}

func statusResponse(w http.ResponseWriter, r *http.Request) {
	election := r.Context().Value(electionContextKey).(*xconsul.Election)
	leader := election.GetLeader()
	status := ""
	if leader == "" {
		status = "Leader election in process"
	} else {
		status = fmt.Sprintf("Acting as leader %v", election.IsLeader())
	}
	appCtx.Render.JSON(w, http.StatusOK, map[string]string{"status": status, "hostname": appCtx.Hostname, "leader": leader})
}

package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/mux"
)

const listenAddr = ":8080"

var wg = &sync.WaitGroup{}

func handleRequest(tasks chan<- JobRequest) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var jobRequest JobRequest
		if err := json.NewDecoder(r.Body).Decode(&jobRequest); err != nil {
			http.Error(w, "error decoding json", http.StatusBadRequest)
			return
		}

		log.Printf("scheduled task at %s\n", time.Now())
		schedule(tasks, jobRequest)
	}
}

type httpServer struct {
	http *http.Server
}

func startHttp(tasks chan<- JobRequest, quit <-chan struct{}) error {
	r := mux.NewRouter()

	r.HandleFunc("/", handleRequest(tasks)).Methods("POST")

	srv := &http.Server{
		Addr:         listenAddr,
		WriteTimeout: time.Second * 10,
		ReadTimeout:  time.Second * 15,
		IdleTimeout:  time.Second * 15,
		Handler:      r,
	}
	// /////////////////////////////////////////////

	idleConnsClosed := make(chan struct{})
	go func() {
		<-quit

		// We received an interrupt signal, shut down.
		if err := srv.Shutdown(context.Background()); err != nil {
			// Error from closing listeners, or context timeout:
			log.Printf("HTTP server Shutdown: %v", err)
		}
		close(idleConnsClosed)
	}()

	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		// Error starting or closing listener:
		log.Printf("starting http server: %v", err)
	}

	<-idleConnsClosed

	// /////////////////////////
	return nil
}

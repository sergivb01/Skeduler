package main

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"gitlab-bcds.udg.edu/sergivb01/skeduler/internal/database"
)

func startHttp(quit <-chan struct{}, conf HttpConfig, db database.Database, finished chan<- struct{}) error {
	r := mux.NewRouter()

	r.HandleFunc("/", handleNewJob(db)).Methods("POST")
	r.HandleFunc("/{id}", handleGetById(db)).Methods("GET")
	r.HandleFunc("/{id}", handleJobUpdate(db)).Methods("PUT")
	r.HandleFunc("/logs/{id}", handleGetLogs()).Methods("GET")
	r.HandleFunc("/logs/{id}/tail", handleFollowLogs()).Methods("GET")

	r.HandleFunc("/workers/poll", handleWorkerFetch(db)).Methods("GET")
	r.HandleFunc("/workers/logs", handleWorkerLogs(db)).Methods("PATCH")

	srv := &http.Server{
		Addr: conf.Listen,
		// deshabilitat per poder fer streaming de logs
		// WriteTimeout: conf.WriteTimeout,
		ReadTimeout: conf.ReadTimeout,
		IdleTimeout: conf.IdleTimeout,
		Handler:     r,
	}

	idleConnsClosed := make(chan struct{})
	go func() {
		<-quit

		// We received an interrupt signal, shut down.
		if err := srv.Shutdown(context.Background()); err != nil {
			// Error from closing listeners, or context timeout:
			log.Printf("error http server shutdown: %v", err)
		}
		close(idleConnsClosed)
	}()

	log.Printf("started http server: %s\n", conf.Listen)
	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		// Error starting or closing listener:
		return fmt.Errorf("running http server: %w", err)
	}

	<-idleConnsClosed
	finished <- struct{}{}

	return nil
}

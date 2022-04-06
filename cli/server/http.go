package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/gofrs/uuid"
	"github.com/gorilla/mux"
	"gitlab-bcds.udg.edu/sergivb01/skeduler/internal/database"
	"gitlab-bcds.udg.edu/sergivb01/skeduler/internal/jobs"
)

func handlePost(db database.Database) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var jobRequest jobs.Job
		if err := json.NewDecoder(r.Body).Decode(&jobRequest); err != nil {
			http.Error(w, "error decoding json", http.StatusBadRequest)
			return
		}

		if err := db.InsertJob(r.Context(), &jobRequest); err != nil {
			http.Error(w, fmt.Sprintf("error inserting job: %v\n", err), http.StatusInternalServerError)
			return
		}

		_ = json.NewEncoder(w).Encode(jobRequest)
	}
}

func handleGet(db database.Database) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		id, err := uuid.FromString(vars["id"])
		if err != nil {
			http.Error(w, "invalid uuid", http.StatusBadRequest)
			return
		}

		job, err := db.GetJobById(r.Context(), id)
		if err != nil {
			http.Error(w, "job does not exist", http.StatusNotFound)
			return
		}

		_ = json.NewEncoder(w).Encode(job)
	}
}

func handleLogs() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		id, err := uuid.FromString(vars["id"])
		if err != nil {
			http.Error(w, "invalid uuid", http.StatusBadRequest)
			return
		}

		f, err := os.OpenFile(fmt.Sprintf("./logs/%v.log", id), os.O_RDONLY, 0644)
		if err != nil {
			http.Error(w, fmt.Sprintf("error opening log file: %v", err), http.StatusInternalServerError)
			return
		}

		if _, err := io.Copy(w, f); err != nil {
			http.Error(w, fmt.Sprintf("error copying log file to body: %v", err), http.StatusInternalServerError)
			return
		}
	}
}

func handleLogsTail() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		// p, ok := w.(http.Pusher)
		// if !ok {
		// 	http.Error(w, "No pusher", http.StatusBadRequest)
		// 	return
		// }
		// p.Push("/")
	}
}

func startHttp(quit <-chan struct{}, conf HttpConfig, db database.Database) error {
	r := mux.NewRouter()

	r.HandleFunc("/", handlePost(db)).Methods("POST")
	r.HandleFunc("/{id}", handleGet(db)).Methods("GET")
	r.HandleFunc("/logs/{id}", handleLogs()).Methods("GET")
	r.HandleFunc("/logs/{id}/tail", handleLogs()).Methods("GET")

	srv := &http.Server{
		Addr:         conf.Listen,
		WriteTimeout: conf.WriteTimeout,
		ReadTimeout:  conf.ReadTimeout,
		IdleTimeout:  conf.IdleTimeout,
		Handler:      r,
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
		log.Printf("error running http server: %v\n", err)
	}

	<-idleConnsClosed

	return nil
}
